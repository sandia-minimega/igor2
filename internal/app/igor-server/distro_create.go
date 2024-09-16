// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

func doCreateDistro(r *http.Request) (distro *Distro, code int, err error) {
	// Check for included existing kernel or initrd hash
	distroName := r.FormValue("name")
	copyDistro := r.FormValue("copyDistro")
	useDistroImage := r.FormValue("useDistroImage")
	imageRef := r.FormValue("imageRef")
	distroDescription := r.FormValue("description")
	kernelArgs := r.FormValue("kernelArgs")
	public := strings.ToLower(r.FormValue("public")) == "true"
	kickstart := r.FormValue("kickstart")
	isDefault := strings.ToLower(r.FormValue("default")) == "true"

	// get the requesting user
	user := getUserFromContext(r)
	code = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		// verify distro name is unique
		if found, findErr := distroExists(distroName, tx); findErr != nil {
			return findErr // uses default err code
		} else if found {
			code = http.StatusConflict
			return fmt.Errorf("distro name already in use: %s", distroName)
		}

		distro = &Distro{Name: distroName}

		// determine image to use in distro
		if copyDistro != "" {
			// EXISTING DISTRO: user may want to base new distro on an existing distro
			// if user is owner of the existing distro, new distro inherits Description,
			// DistroImage, and KernelArgs from existing distro
			// If DistroImage is a LocalBoot, will also copy kickstart
			dList, status, findErr := getDistros([]string{copyDistro}, tx)
			if findErr != nil {
				code = status
				return findErr
			}
			edName := dList[0].Owner.Name
			if user.Name != edName && !userElevated(user.Name) {
				code = http.StatusForbidden
				return fmt.Errorf("must be the owner of the existing distro %s when using to create new", copyDistro)
			}
			doCopyDistro(dList[0], distro)
		} else if distro.DistroImage.ImageID == "" && useDistroImage != "" {
			// USE DISTRO IMAGE: represents the name of an existing Distro
			// if user is the owner of the existing distro, new distro inherits its Image only
			dList, status, gdErr := getDistros([]string{useDistroImage}, tx)
			if gdErr != nil {
				code = status
				return gdErr
			}
			// user must be owner of existing distro to use its image
			exDistro := dList[0]
			edName := exDistro.Owner.Name
			if user.Name != edName && !userElevated(user.Name) {
				code = http.StatusForbidden
				return fmt.Errorf("must be the owner of the existing distro %s to use its image in a new distro", useDistroImage)
			}
			distro.DistroImage = exDistro.DistroImage
		} else if distro.DistroImage.ImageID == "" && imageRef != "" {
			// get the image associated with the ref
			if images, status, err := getImages([]string{imageRef}, tx); err != nil {
				code = status
				return err
			} else {
				image := images[0]
				distro.DistroImage = image
			}
		} else if distro.DistroImage.ImageID == "" {
			// Register files and generate hash/image if files were included with these params
			if len(r.MultipartForm.File) > 0 {
				// check to make sure this is allowed
				if !igor.Server.AllowImageUpload {
					return fmt.Errorf("uploading images is not permitted, see an admin for assistance with registering a new image to get an image reference value")
				}
				// if they included a kickstart, assume they're trying to register a local image
				// reject and inform that must be done as a separate step
				if kickstart != "" {
					return fmt.Errorf("distro image intended for local install/boot must be registered as a seperate step")
				}
				image, status, err := registerImage(r, tx)
				if err != nil {
					code = status
					return err
				}
				if image != nil {
					distro.DistroImage = *image
				} else {
					return fmt.Errorf("received empty image object when registering image files") // uses default err code
				}
			} else {
				code = http.StatusBadRequest
				return fmt.Errorf("existing distro, existing image, image ref, or uploaded image files are required to create a new distro")
			}
		}

		// set owner and groups whether public or private
		// PUBLIC: If user decided to make distro public,
		// swap user out with admin and set group to all
		if public {
			// set owner as igor-admin
			if admin, status, findErr := getIgorAdmin(tx); findErr != nil {
				code = status
				return findErr
			} else {
				distro.Owner = *admin
			}

			if groups, ok := r.Form["distroGroups"]; ok {
				for _, gName := range groups {
					if gName != GroupAll {
						code = http.StatusBadRequest
						return fmt.Errorf("a public distro cannot be assigned to specific groups: %v", strings.Join(groups, ","))
					}
				}
			}

			// set group as all
			if allGroup, status, err := getAllGroup(tx); err != nil {
				code = status
				return err
			} else {
				distro.Groups = []Group{*allGroup}
			}
		} else {
			// OWNER: set user as distro owner
			distro.Owner = *user

			// GROUP: set at least user to group, look for additional and add
			var groupNames []string
			if pfErr := r.ParseForm(); pfErr != nil {
				return pfErr
			}
			if groups, ok := r.Form["distroGroups"]; ok {
				for _, gName := range groups {
					if gName != GroupAll {
						groupNames = append(groupNames, gName)
					} else {
						code = http.StatusBadRequest
						return fmt.Errorf("cannot add group '%s' to a non-public distro", GroupAll)
					}
				}
			}

			foundGroups, rgErr := dbReadGroups(map[string]interface{}{"name": groupNames}, true, tx)
			if rgErr != nil {
				return rgErr // uses default err code
			}
			if len(foundGroups) != len(groupNames) {
				var missingGroups []string
				for _, gname := range groupNames {
					if !groupSliceContains(foundGroups, gname) {
						missingGroups = append(missingGroups, gname)
					}
				}
				code = http.StatusNotFound
				return fmt.Errorf("error finding group(s) for distro: %s", strings.Join(missingGroups, ","))
			}
			// user must be a member of all groups to be added
			if member, badGroup := user.isMemberOfGroups(foundGroups); !member {
				code = http.StatusForbidden
				return fmt.Errorf("user is not a member of group %s to include in new distro", badGroup)
			}
			// now add the owner's pug
			pug, err := distro.Owner.getPug()
			if err != nil {
				return fmt.Errorf("error retrieving owner's personal group to add to distro")
			}
			foundGroups = append(foundGroups, *pug)

			distro.Groups = foundGroups
		}

		// DESCRIPTION: set optional distro description
		distro.Description = strings.TrimSpace(distroDescription)

		// KERNELARGS: set optional kernel args
		distro.KernelArgs = strings.TrimSpace(kernelArgs)

		// If distro is using a Local Boot image, kickstart is required
		if distro.DistroImage.LocalBoot {
			// first check if local boot distro creation restricted to admin only
			if !igor.Server.UserLocalBootDC && !userElevated(user.Name) {
				return fmt.Errorf("creation of local boot Distro is restricted to admins")
			}
			if distro.Kickstart.Name == "" && kickstart == "" {
				return fmt.Errorf("kickstart file name required when creating a local boot distro")
			}
			if kickstart != "" {
				kss, err := dbReadKS(map[string]interface{}{"name": kickstart}, tx)
				if err != nil {
					return err
				}
				if len(kss) > 1 {
					return fmt.Errorf("multiple kickstarts returned for given name %s", kickstart)
				}
				if len(kss) == 0 {
					return fmt.Errorf("no kickstarts returned for given name %s", kickstart)
				}
				ks := kss[0]
				distro.Kickstart = ks
				distro.KickstartID = ks.ID
			}
		}

		// user indicated distro to be default
		if isDefault {
			// reject if user not elevated
			if !userElevated(user.Name) {
				return fmt.Errorf("setting a Distro as default is restricted to admins")
			}
			// make sure any existing default distro is set to false
			currentDefaultDistros, err := dbReadDistrosTx(map[string]interface{}{"is_default": true})
			if err != nil {
				return fmt.Errorf("unexpected error searching for existing default distro, please notify admin")
			}
			change := map[string]interface{}{"is_default": false}
			for _, cdd := range currentDefaultDistros {
				err := dbEditDistro(&cdd, change, tx)
				if err != nil {
					return fmt.Errorf("unexpected error updating existing default distro, please notify admin - %v", err)
				}
			}
			// make our new distro the new default
			distro.IsDefault = true

			// change the owner to Igor-Admin
			if admin, status, findErr := getIgorAdmin(tx); findErr != nil {
				code = status
				return findErr
			} else {
				distro.Owner = *admin
			}
			// set distro groups with only Igor-Admin pug
			pug, err := distro.Owner.getPug()
			if err != nil {
				return fmt.Errorf("error retrieving owner's personal group to add to distro")
			}
			distro.Groups = []Group{*pug}
		}

		dbAccess.Lock()
		defer dbAccess.Unlock()
		// commit to DB
		return dbCreateDistro(distro, tx) // uses default err code

	}); err == nil {
		code = http.StatusCreated
	}
	return
}

func doCopyDistro(src Distro, target *Distro) {
	// Does not copy name, owner, or group
	target.Description = src.Description
	target.DistroImage = src.DistroImage
	target.KernelArgs = src.KernelArgs
	if src.DistroImage.LocalBoot {
		target.Kickstart = src.Kickstart
		target.KickstartID = src.KickstartID
	}

}
