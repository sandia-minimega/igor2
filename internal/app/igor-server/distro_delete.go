// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"
)

// doDeleteDistro steps through the process of making an update to a distro record.
//
// Returns:
//
//	204,nil if delete was successful
//	400,err if delete was not allowed
//	404,err if user cannot be found
//	500,err if an internal error occurred
func doDeleteDistro(distroName string, r *http.Request) (code int, err error) {

	clog := hlog.FromRequest(r)
	code = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		// get the distro object first
		distros, status, gdErr := getDistros([]string{distroName}, tx)
		if gdErr != nil {
			code = status
			return gdErr
		}
		distro := &distros[0]

		// fail if distro is linked to any profile other than default
		clog.Debug().Msgf("checking distro '%s' for linked profiles", distroName)
		linked, profs, lnkErr := distro.isLinkedToProfiles(tx)
		if lnkErr != nil {
			return lnkErr // uses default err code
		}
		if linked {
			code = http.StatusBadRequest
			return fmt.Errorf("cannot delete distro, currently attached to profile(s) %s. Delete these profile(s) before deleting this distro", profs)
		}

		// get the distro image name for later
		imageName := distro.DistroImage.Name

		// destroy distro
		clog.Debug().Msgf("deleting distro '%s'", distroName)
		deleteErr := dbDeleteDistro(distro, tx)
		if deleteErr != nil {
			return deleteErr
		}

		// destroy the image if it's now an orphan
		images, _, err := getImages([]string{imageName}, tx)
		if err != nil {
			return err
		}
		if len(images) == 0 {
			return fmt.Errorf("error retrieving image object %v from deleted Distro", imageName)
		}
		image := images[0]

		if len(image.Distros) == 0 {
			// this image no longer has a distro attached to it, destroy it
			return deleteDistroImage(&image, tx, clog)
		}

		return nil
	}); err == nil {
		code = http.StatusOK
	}
	return
}
