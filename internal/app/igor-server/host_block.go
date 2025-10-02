// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"igor2/internal/pkg/common"
	"net/http"
	"sort"
	"time"

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
func doUpdateBlockHosts(blockAction bool, hostList []string, r *http.Request) (status int, err error) {

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

		if blockAction {

			blockedRes := make(map[string]Reservation)
			for _, h := range hList {
				if h.State == HostReserved {
					for _, res := range h.Reservations {
						if res.IsActive(time.Now()) {
							blockedRes[res.Name] = res
						}
					}
				}
			}

			blockErr := dbEditHosts(hList, map[string]interface{}{"State": HostBlocked}, tx)
			if blockErr != nil {
				return blockErr
			}
			// if host is in maintenance mode, set restore state to blocked so it remains blocked when finished.
			for _, host := range hList {
				if len(host.MaintenanceRes) > 0 {
					blockErr := dbEditHosts([]Host{host}, map[string]interface{}{"RestoreState": HostBlocked}, tx)
					if blockErr != nil {
						return blockErr
					}
				}
			}

			if len(blockedRes) > 0 {
				actionUser := getUserFromContext(r)
				isElevated := userElevated(actionUser.Name)

				for _, bRes := range blockedRes {
					var blockList []string
					var clusterName = ""
					for _, host := range hList {
						for _, hostRes := range host.Reservations {
							if bRes.Name == hostRes.Name {
								blockList = append(blockList, host.HostName)
								clusterName = host.Cluster.Name
							}
						}
					}

					res, _, _ := getReservations([]string{bRes.Name}, tx)

					blockEvent := makeResEditNotifyEvent(EmailResBlock, &res[0], clusterName, actionUser, isElevated, common.UnsplitList(blockList))
					if blockEvent != nil {
						resNotifyChan <- *blockEvent
					}
				}
			}

			return nil
		} else {

			for _, h := range hList {
				if h.State != HostBlocked {
					status = http.StatusConflict
					return fmt.Errorf("cannot un-block a non-blocked host: '%s'", h.HostName)
				}
			}

			var hAvailList []Host
			var hReservedList []Host

			for _, h := range hList {
				if len(h.Reservations) == 0 {
					hAvailList = append(hAvailList, h) // no reservations, set to available
				} else {
					for _, r := range h.Reservations {
						if r.IsActive(time.Now()) {
							hReservedList = append(hReservedList, h) // current reservation, set to reserved
						} else {
							hAvailList = append(hAvailList, h) // has a future reservation, set to available
						}
					}
				}
			}

			var unblockErr error

			// do unreserved first
			if len(hAvailList) > 0 {
				unblockErr = dbEditHosts(hAvailList, map[string]interface{}{"State": HostAvailable}, tx)
				if unblockErr != nil {
					return unblockErr
				}
			}

			if len(hReservedList) > 0 {
				unblockErr = dbEditHosts(hReservedList, map[string]interface{}{"State": HostReserved}, tx)
			}
			return unblockErr
		}

	}); err == nil {
		status = http.StatusOK
	}
	return
}
