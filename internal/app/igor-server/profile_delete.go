// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"

	"gorm.io/gorm"
)

func doDeleteProfile(profileName string) (code int, err error) {

	code = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		pList, status, gpErr := getProfiles([]string{profileName}, tx)
		if gpErr != nil {
			code = status
			return gpErr
		}
		p := &pList[0]

		// make sure profile isn't attached to an active reservation
		res, rrErr := dbReadReservations(map[string]interface{}{"profile_id": p.ID}, nil, tx)
		if rrErr != nil {
			return rrErr
		}
		if len(res) > 0 {
			code = http.StatusConflict
			return fmt.Errorf("cannot delete profile associated with a reservation")
		}

		return dbDeleteProfile(p, tx) // uses default err code

	}); err == nil {
		code = http.StatusOK
	}
	return
}
