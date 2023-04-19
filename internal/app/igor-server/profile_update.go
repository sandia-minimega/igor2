// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"github.com/rs/zerolog/hlog"
	"net/http"

	"gorm.io/gorm"
)

func doUpdateProfile(profileName string, editParams map[string]interface{}, r *http.Request) (code int, err error) {

	clog := hlog.FromRequest(r)
	code = http.StatusInternalServerError // default status, overridden at end if no errors
	var p *Profile

	if err = performDbTx(func(tx *gorm.DB) error {

		pList, gpStatus, gpErr := getProfiles([]string{profileName}, tx)
		if gpErr != nil {
			code = gpStatus
			return gpErr
		}
		p = &pList[0]

		changes, pStatus, pErr := parseProfileEditParams(p, editParams)
		if pErr != nil {
			code = pStatus
			return pErr
		}

		if name, ok := changes["name"].(string); ok {
			if snpList, status, findErr := getProfiles([]string{name}, tx); findErr != nil {
				code = status
				return findErr // uses default err status
			} else if len(snpList) > 0 {
				code = http.StatusConflict
				return fmt.Errorf("profile name '%s' already in use", name)
			}
		}

		return dbEditProfile(p, changes, tx) // uses default err status

	}); err == nil {

		// if the profile update was successful and the update included a name change, record this history with any
		// affected reservations. don't stop if the record doesn't update properly
		_, hasName := editParams["name"]
		if hasName {
			rList, _ := dbReadReservationsTx(map[string]interface{}{"profile_id": p.ID}, nil)
			for _, res := range rList {
				if hErr := res.HistCallback(&res, HrUpdated+":profile-rename"); hErr != nil {
					clog.Error().Msgf("failed to record reservation '%s' profile rename to history", res.Name)
				} else {
					clog.Debug().Msgf("profile renamed - recorded historical change to reservation '%s'", res.Name)
				}
			}
		}

		code = http.StatusOK
	}
	return
}

// parseProfileEditParams creates a new map from editParams that contains the information required to update
// the profile record.
func parseProfileEditParams(p *Profile, editParams map[string]interface{}) (map[string]interface{}, int, error) {

	changes := map[string]interface{}{}

	if name, ok := editParams["name"].(string); ok {
		changes["Name"] = name
	}
	if desc, ok := editParams["description"].(string); ok {
		changes["Description"] = desc
	}
	if ka, ok := editParams["kernelArgs"].(string); ok {
		changes["kernel_args"] = ka
	}

	// if profile is default and user making valid changes,
	// then make the profile permanent for the user
	if p.IsDefault && len(changes) > 0 {
		changes["is_default"] = false
	}
	return changes, http.StatusOK, nil
}
