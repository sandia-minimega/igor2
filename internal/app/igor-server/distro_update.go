// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/rs/zerolog/hlog"

	"gorm.io/gorm"
)

func doUpdateDistro(target *Distro, r *http.Request) (code int, err error) {

	clog := &logger
	if r != nil {
		clog = hlog.FromRequest(r)
	}
	var updateParams map[string]interface{}
	code = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		// convert request changes from multiform to msi, + some validation
		updateParams, code, err = parseDistroUpdateParams(target, r, tx)
		if err != nil {
			return err
		}

		// make sure proposed changes will result in a valid distro
		if vdupStatus, vdupErr := validateDistroUpdatePermissions(target, updateParams, tx); vdupErr != nil {
			code = vdupStatus
			return vdupErr
		}
		// execute change
		return dbEditDistro(target, updateParams, tx) // uses default err code

	}); err == nil {

		// if the distro update was successful and the update included a name change, record this history with any
		// affected reservations. don't stop if the record doesn't update properly
		_, hasName := updateParams["name"]
		if hasName {
			rList, _ := dbReadReservationsTx(map[string]interface{}{"distro_id": target.ID}, nil)
			for _, res := range rList {
				if hErr := res.HistCallback(&res, HrUpdated+":distro-rename"); hErr != nil {
					clog.Error().Msgf("failed to record reservation '%s' distro rename to history", res.Name)
				} else {
					clog.Debug().Msgf("distro renamed - recorded historical change to reservation '%s'", res.Name)
				}
			}
		}

		code = http.StatusOK
	}
	return
}

func parseDistroUpdateParams(target *Distro, r *http.Request, tx *gorm.DB) (map[string]interface{}, int, error) {
	changes := map[string]interface{}{}

	// establish requesting user
	reqUser := getUserFromContext(r)

	// check for existing distro name
	name := r.FormValue("name")
	if name != "" {
		found, findErr := distroExists(name, tx)
		if findErr != nil {
			return nil, http.StatusInternalServerError, findErr // uses default err code
		} else if found {
			return nil, http.StatusConflict, fmt.Errorf("%s already in use as distro name", name)
		}
		changes["Name"] = name
	}

	if _, ok := r.PostForm["deprecate"]; ok {

		if !groupSliceContains(reqUser.Groups, GroupAdmins) {
			return nil, http.StatusBadRequest, fmt.Errorf("deprecating a distro is restricted to admins")
		}

		if !userElevated(reqUser.Name) {
			return nil, http.StatusBadRequest, fmt.Errorf("non-elevated user cannot deprecate a distro")
		}

		if target.isPublic() {
			changes["removeGroup"] = []string{GroupAll}
			changes["addGroup"] = []string{GroupAdmins}
			return changes, http.StatusOK, nil
		} else {
			return nil, http.StatusBadRequest, fmt.Errorf("cannot deprecate a non-public distro")
		}
	}

	// check desc
	if desc, ok := r.PostForm["description"]; ok {
		changes["Description"] = strings.TrimSpace(desc[0])
	}

	// check info
	if initrd, ok := r.PostForm["initrdInfo"]; ok {
		changes["initrdInfo"] = strings.TrimSpace(initrd[0])
	}

	// check kernel args
	if ka, ok := r.PostForm["kernelArgs"]; ok {
		// make sure distro isn't currently being used
		if activeRes := target.hasActiveReservations(); len(activeRes) > 0 {
			status := http.StatusConflict
			err := fmt.Errorf("distro kernel args cannot be updated while associated to active Reservations: %s", activeRes)
			return nil, status, err
		} else {
			changes["kernel_args"] = strings.TrimSpace(ka[0])
		}
	}
	// check kickstart
	if ks, ok := r.PostForm["kickstart"]; ok {
		// can't assign a kickstart in distro is not installed
		if !target.DistroImage.LocalBoot {
			return nil, http.StatusBadRequest, fmt.Errorf("kickstart script can only be assigned to a distro with a local boot image")
		}
		// make sure distro isn't currently being used
		if activeRes := target.hasActiveReservations(); len(activeRes) > 0 {
			status := http.StatusConflict
			err := fmt.Errorf("distro kickstart cannot be updated while associated to active Reservations: %s", activeRes)
			return nil, status, err
		} else {
			kickstarts, err := dbReadKickstartTx(map[string]interface{}{"filename": ks})
			if err != nil {
				return changes, http.StatusInternalServerError, err
			}
			if len(kickstarts) == 0 {
				return changes, http.StatusBadRequest, fmt.Errorf("no kickstart files found using name %s", ks)
			}
			changes["kickstart"] = kickstarts[0]
		}
	}
	// check if public
	isPublic := strings.ToLower(r.FormValue("public")) == "true"
	if isPublic {
		changes["isPublic"] = isPublic
	}
	// check new owner
	newOwnerName := r.FormValue("owner")
	if newOwnerName == IgorAdmin && !userElevated(reqUser.Name) {
		return nil, http.StatusBadRequest, fmt.Errorf("non-elevated user cannot transfer ownership of a distro to '%s'", IgorAdmin)
	}
	if newOwnerName != "" && !isPublic {
		uList, status, err := getUsers([]string{newOwnerName}, false, tx)
		if err != nil {
			return nil, status, err
		}
		newOwner := &uList[0]
		changes["owner"] = newOwner
	}
	// group_add should be an array of valid group names
	if groupAdd, ok := r.PostForm["addGroup"]; ok {
		if len(groupAdd) > 0 {
			if isPublic || target.isPublic() {
				return nil, http.StatusBadRequest, fmt.Errorf("a public distro cannot be assigned to specific groups: [%v]", strings.Join(groupAdd, ","))
			} else {
				for _, gName := range groupAdd {
					if gName == GroupAll {
						return nil, http.StatusForbidden, fmt.Errorf("cannot add the group '%s' to a non-public group", gName)
					}
				}
			}
			// if any group names are invalid, getGroupsTx currently includes them in returned err
			_, code, err := getGroups(groupAdd, true, tx)
			if err != nil {
				return nil, code, err
			}
			changes["addGroup"] = groupAdd
		}
	}

	// group_remove should be an array of valid group names
	if groupRemove, ok := r.PostForm["removeGroup"]; ok {
		if len(groupRemove) > 0 {
			// if any group names are invalid, getGroupsTx currently includes them in returned err
			_, code, err := getGroups(groupRemove, true, tx)
			if err != nil {
				return nil, code, err
			}
			if isPublic || target.isPublic() {
				if slices.Contains(groupRemove, GroupAll) {
					code := http.StatusBadRequest
					return nil, code, fmt.Errorf("cannot remove the '%s' group from a public distro", GroupAll)
				}
			}

			// abort if Distro already isn't associated with the requested Group
			targetGroups := target.Groups
			for _, g := range groupRemove {
				if !groupSliceContains(targetGroups, g) {
					code := http.StatusBadRequest
					return nil, code, fmt.Errorf("target distro \"%v\" does not contain group \"%v\" - edit operation aborted", target.Name, g)
				}
			}
			changes["removeGroup"] = groupRemove
		}
	}
	// check if default
	isDefault := strings.ToLower(r.FormValue("default")) == "true"
	if isDefault {
		// reject if user not elevated
		if !userElevated(reqUser.Name) {
			return changes, http.StatusBadRequest, fmt.Errorf("setting a Distro as default is restricted to admins")
		}
		// make sure any existing default distro is set to false
		currentDefaultDistros, err := dbReadDistrosTx(map[string]interface{}{"is_default": true})
		if err != nil {
			return changes, http.StatusInternalServerError, fmt.Errorf("unexpected error searching for existing default distro, please notify admin")
		}
		change := map[string]interface{}{"is_default": false}
		for _, cdd := range currentDefaultDistros {
			if err := dbEditDistro(&cdd, change, tx); err != nil {
				return changes, http.StatusInternalServerError, fmt.Errorf("unexpected error updating existing default distro, please notify admin")
			}
		}
		// make our new distro the new default
		changes["is_default"] = isDefault

		// change the owner to Igor-Admin
		admin, status, findErr := getIgorAdmin(tx)
		if findErr != nil {
			return changes, status, findErr
		} else {
			changes["owner"] = admin
		}
		// remove all existing groups (except user's pug)
		targetGroups := target.Groups
		var groupRemove []string
		userPugID, _ := target.Owner.getPugID()
		for _, group := range targetGroups {
			// allow db to handle removal of old user pug
			if group.ID != userPugID {
				groupRemove = append(groupRemove, group.Name)
			}
		}
		if len(groupRemove) > 0 {
			changes["removeGroup"] = groupRemove
		}

		// set distro groups with Igor-Admin pug
		// pug, err := admin.getPug()
		// if err != nil {
		// 	return changes, http.StatusBadRequest, fmt.Errorf("error retrieving owner's personal group to add to distro")
		// }
		// changes["addGroup"] = []string{pug.Name}
	}

	// check if we're removing a default
	defaultRemove := strings.ToLower(r.FormValue("default_remove")) == "true"
	if defaultRemove {
		changes["is_default"] = false
	}
	return changes, http.StatusOK, nil
}

func validateDistroUpdatePermissions(target *Distro, updateParams map[string]interface{}, tx *gorm.DB) (int, error) {
	// if is_public included, make sure owner and group fields are removed
	if _, ok := updateParams["public"]; ok {
		delete(updateParams, "owner")
		delete(updateParams, "addGroup")
		delete(updateParams, "removeGroup")
	}

	users, code, err := getUsers([]string{target.Owner.Name}, true, tx)
	if err != nil {
		return code, err
	}
	owner := &users[0]

	// start with existing groups
	currentGroups := target.Groups

	var intendedOwner *User
	if newOwner, ok := updateParams["owner"].(*User); ok {
		intendedOwner = newOwner
		// if owner is changing, make sure we also remove the old owner's pug
		pug, pugErr := owner.getPug()
		if pugErr != nil {
			return http.StatusInternalServerError, pugErr
		}
		currentGroups = removeGroup(currentGroups, pug)
	} else {
		intendedOwner = owner
	}
	// establish what the complete final group list should be after changes
	// if groups are being added, make sure owner is in new groups
	// (and if owner is changing too, make sure new user is also in current groups)

	// remove groups user intends to remove
	if gRemove, ok := updateParams["removeGroup"].([]string); ok {
		toRemove, ggCode, ggErr := getGroups(gRemove, true, tx)
		if ggErr != nil {
			return ggCode, ggErr
		}
		updateParams["removeGroup"] = toRemove
		for _, rg := range toRemove {
			currentGroups = removeGroup(currentGroups, &rg)
		}
	}
	// add those which user intends to add
	if gAdd, ok := updateParams["addGroup"].([]string); ok {
		toAdd, ggCode, ggErr := getGroups(gAdd, true, tx)
		if ggErr != nil {
			return ggCode, ggErr
		}
		updateParams["addGroup"] = toAdd
		currentGroups = append(currentGroups, toAdd...)
	}
	// now we can check if the intended owner is a member of all intended outcome groups
	member, badGroup := intendedOwner.isMemberOfGroups(currentGroups)
	if !member && intendedOwner.Name != IgorAdmin {
		return http.StatusBadRequest, fmt.Errorf("intended distro owner %s is not a member of group(s) %s", intendedOwner.Name, badGroup)
	}

	return http.StatusOK, nil
}
