// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"

	"gorm.io/gorm"
)

func doCreateProfile(createProfileParams map[string]interface{}, r *http.Request) (profile *Profile, code int, err error) {

	profileName := createProfileParams["name"].(string)
	owner := getUserFromContext(r)

	code = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		// Make sure this user doesn't already have a profile with this name
		queryParams := map[string]interface{}{"name": profileName, "owner_id": owner.ID}
		profiles, findErr := dbReadProfiles(queryParams, tx)
		if findErr != nil {
			return findErr
		} else if len(profiles) > 0 {
			return fmt.Errorf("profile '%s' already exists", profileName)
		}

		var distro *Distro
		distroName := createProfileParams["distro"].(string)
		if distroList, status, dErr := getDistros([]string{distroName}, tx); dErr != nil {
			code = status
			return dErr
		} else {
			distro = &distroList[0]
		}

		var desc string
		desc, _ = createProfileParams["description"].(string)
		var kernelArgs string
		kernelArgs, _ = createProfileParams["kernelArgs"].(string)

		profile = &Profile{
			Name:        profileName,
			Description: desc,
			Owner:       *owner,
			Distro:      *distro,
			KernelArgs:  kernelArgs,
		}

		return dbCreateProfile(profile, tx) // uses default err code

	}); err == nil {
		code = http.StatusCreated
	}

	return
}
