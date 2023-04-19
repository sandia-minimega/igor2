// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"igor2/internal/pkg/common"

	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"
)

func doCreateHostPolicy(createHostPolicyParams map[string]interface{}, r *http.Request) (hostPolicy *HostPolicy, code int, err error) {

	clog := hlog.FromRequest(r)
	code = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		// Determine if given HostPolicy name is unique
		hostPolicyName := createHostPolicyParams["name"].(string)

		// cannot allow name of new policy to be default
		if strings.ToLower(hostPolicyName) == DefaultPolicyName {
			code = http.StatusConflict
			return fmt.Errorf("'%s' is the system default host policy name", hostPolicyName)
		}

		exists, exErr := hostPolicyExists(hostPolicyName, tx, clog)
		if exErr != nil {
			return err // uses default 500
		}
		if exists {
			// the hostPolicy already exists
			code = http.StatusConflict
			return fmt.Errorf("host policy '%s' already exists", hostPolicyName)
		}

		// Determine maxDurationTime

		var maxResTime time.Duration
		durStr, ok := createHostPolicyParams["maxResTime"].(string)
		if !ok {
			maxResTime = time.Minute * time.Duration(igor.Scheduler.MaxReserveTime)
		} else {
			maxResTime, err = common.ParseDuration(durStr)
			if err != nil {
				return err // uses default err status
			}
		}

		// Determine AccessGroups
		var groups []Group
		names, ok2 := createHostPolicyParams["accessGroups"].([]interface{})
		if ok2 {
			// do not allow admin or all group or pug to be user specified
			var grNames []string
			for _, name := range names {
				nm := name.(string)
				if !(nm == GroupAll || nm == GroupAdmins || strings.HasPrefix(nm, GroupUserPrefix) || nm == GroupNoneAlias) {
					grNames = append(grNames, nm)
				} else {
					code = http.StatusConflict
					return fmt.Errorf("group not allowed as access group: %v", nm)
				}
			}
			foundGroups, status, gErr := getGroups(grNames, true, tx)
			if gErr != nil {
				code = status
				return gErr
			}
			if len(foundGroups) > 0 && len(foundGroups) != len(grNames) {
				code = http.StatusBadRequest
				return fmt.Errorf("invalid group(s) included in request")
			}
			groups = foundGroups
		}
		if len(groups) == 0 {
			allGroup, status, gaErr := getAllGroupTx()
			if err != nil {
				code = status
				return gaErr
			}
			groups = []Group{*allGroup}
		}

		// Determine notAvailable entries
		sba := ScheduleBlockArray{}
		sbList, ok3 := createHostPolicyParams["notAvailable"].([]interface{})
		if ok3 {
			for _, sbInstance := range sbList {
				thisSB, _ := sbInstance.(map[string]interface{})
				sba = append(sba, common.ScheduleBlock{Start: thisSB["start"].(string), Duration: thisSB["duration"].(string)})
			}
		}

		hostPolicy = &HostPolicy{
			Name:         hostPolicyName,
			MaxResTime:   maxResTime,
			AccessGroups: groups,
			NotAvailable: sba,
		}

		return dbCreateHostPolicy(hostPolicy, tx) // uses default err status

	}); err == nil {
		code = http.StatusCreated
	}

	return
}
