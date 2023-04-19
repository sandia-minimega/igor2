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

// delete the given host policy
func doDeleteHostPolicy(hpName string, r *http.Request) (code int, err error) {

	clog := hlog.FromRequest(r)
	code = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {
		// do not allow delete to happen if policy is named DefaultPolicyName
		if hpName == DefaultPolicyName {
			code = http.StatusForbidden
			return fmt.Errorf("deleting the default host policy is not permitted")
		}

		hpList, status, ghErr := getHostPolicies([]string{hpName}, tx, clog)
		if ghErr != nil {
			code = status
			return ghErr
		}
		target := &hpList[0]

		// do not allow delete to happen if policy is still attached to a host
		if len(target.Hosts) > 0 {
			code = http.StatusConflict
			return fmt.Errorf("deleting host policy while still assigned to a host is not permitted")
		}

		return dbDeleteHostPolicy(target, tx) // uses default err status

	}); err == nil {
		code = http.StatusOK
	}
	return
}
