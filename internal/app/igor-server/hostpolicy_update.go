// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"strings"

	zl "github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"

	"igor2/internal/pkg/common"
)

func doUpdateHostPolicy(hostPolicyName string, editParams map[string]interface{}, r *http.Request) (code int, err error) {

	clog := hlog.FromRequest(r)
	code = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		// reject update request if target host policy is the default
		if hostPolicyName == DefaultPolicyName {
			code = http.StatusForbidden
			return fmt.Errorf("changes to the default host policy are not permitted; MaxResTime will be auto-updated by changing the server config setting")
		}

		hpList, status, ghpErr := getHostPolicies([]string{hostPolicyName}, tx, clog)
		if ghpErr != nil {
			code = status
			return ghpErr
		}

		return dbEditHostPolicy(hpList, editParams, tx) // uses default err status

	}); err == nil {
		code = http.StatusOK
	}
	return
}

func parseHostPolicyEditParams(editParams map[string]interface{}, clog *zl.Logger) (map[string]interface{}, int, error) {

	changes := map[string]interface{}{}

	// determine change to name
	if val, ok := editParams["name"].(string); ok {
		existing, err := dbReadHostPoliciesTx(map[string]interface{}{"name": val}, clog)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		} else if len(existing) > 0 {
			return nil, http.StatusConflict, fmt.Errorf("host policy '%s' already exists", val)
		} else {
			changes["name"] = val
		}
	}

	// determine changes to maxResTime
	if val, ok := editParams["maxResTime"].(string); ok {
		dur, _ := common.ParseDuration(val)
		changes["maxResTime"] = dur
	}

	// determine changes to removeGroup
	if val, ok := editParams["removeGroups"].([]interface{}); ok {
		var rGroupNames []string
		for _, n := range val {
			rGroupNames = append(rGroupNames, n.(string))
		}
		groupsToRemove, status, err := getGroupsTx(rGroupNames, true)
		if err != nil {
			return nil, status, err
		} else if len(groupsToRemove) == 0 {
			return nil, http.StatusNotFound, fmt.Errorf("requested group(s) to remove not found")
		}
		changes["removeGroups"] = groupsToRemove
	}

	// determine changes to addGroup
	// do not allow admin or all group or pug to be user specified
	if val, ok := editParams["addGroups"].([]interface{}); ok {
		var aGroupNames []string
		for _, n := range val {
			nm := n.(string)
			if nm != GroupAdmins && nm != GroupAll && !strings.HasPrefix(nm, GroupUserPrefix) {
				aGroupNames = append(aGroupNames, nm)
			} else {
				return nil, http.StatusConflict, fmt.Errorf("group not allowed as access group: %v", nm)
			}
		}
		groupToAdd, status, err := getGroupsTx(aGroupNames, true)
		if err != nil {
			return nil, status, err
		} else if len(groupToAdd) == 0 {
			return nil, http.StatusNotFound, fmt.Errorf("requested group(s) to add not found")
		}
		changes["addGroups"] = groupToAdd
	}

	// determine changes to addNotAvailable
	sbAdd := ScheduleBlockArray{}
	sbaList, ok := editParams["addNotAvailable"].([]interface{})
	if ok {
		for _, sbInstance := range sbaList {
			thisSB, _ := sbInstance.(map[string]interface{})
			sbAdd = append(sbAdd, common.ScheduleBlock{Start: thisSB["start"].(string), Duration: thisSB["duration"].(string)})
		}
		changes["addNotAvailable"] = sbAdd
	}

	// determine changes to removeNotAvailable
	sbRemove := ScheduleBlockArray{}
	sbdList, ok := editParams["removeNotAvailable"].([]interface{})
	if ok {
		for _, sbInstance := range sbdList {
			thisSB, _ := sbInstance.(map[string]interface{})
			sbRemove = append(sbRemove, common.ScheduleBlock{Start: thisSB["start"].(string), Duration: thisSB["duration"].(string)})
		}
		changes["removeNotAvailable"] = sbRemove
	}

	return changes, http.StatusOK, nil
}
