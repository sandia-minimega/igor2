// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"sort"

	"gorm.io/gorm"
)

// Maps the power command parameters to a list of hosts and checks permissions to ensure the user
// can actually issue a power command for those hosts.
func checkBlockParams(powerParams map[string]interface{}) (bool, []string, int, error) {

	block := powerParams["block"].(bool)
	val := powerParams["hosts"].(string)

	hostList := igor.splitRange(val)
	if len(hostList) == 0 {
		return block, nil, http.StatusBadRequest, fmt.Errorf("can't parse hosts - %v", val)
	}
	sort.Slice(hostList, func(i, j int) bool {
		return hostList[i] < hostList[j]
	})

	return block, hostList, http.StatusOK, nil
}

// Runs the actual power command for the service that controls host power options.
func doUpdateBlockHosts(blockAction bool, hostList []string) (status int, err error) {

	status = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		hList, ghStatus, ghErr := getHostsTx(hostList, true)
		if err != nil {
			status = ghStatus
			return ghErr
		}
		if len(hList) == 0 {
			return fmt.Errorf("no hosts found for given host list: %v", hostList)
		}

		var hostChangeState HostState
		if blockAction {
			hostChangeState = HostBlocked
		} else {
			hostChangeState = HostAvailable
		}

		for _, h := range hList {
			if h.State == HostReserved || len(h.Reservations) > 0 {
				// you can't change the state to blocked on an active reservation
				if hostChangeState == HostBlocked {
					status = http.StatusConflict
					return fmt.Errorf("cannot block a host with an active reservation")
				}
				// you can't change the state to available if the host isn't currently blocked or available
				if hostChangeState == HostAvailable && h.State != HostBlocked && h.State != HostAvailable {
					status = http.StatusConflict
					return fmt.Errorf("cannot un-block a non-blocked host")
				}
			}
		}

		return dbEditHosts(hList, map[string]interface{}{"State": hostChangeState}, tx) // uses default err status

	}); err == nil {
		status = http.StatusOK
	}
	return
}
