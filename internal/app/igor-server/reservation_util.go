// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"sort"
	"time"

	"igor2/internal/pkg/common"

	"gorm.io/gorm"
)

func checkTimeLimit(nodeCount int, limit time.Duration, resDur time.Duration) error {
	// nodeCount is ignored at the moment.
	// In old igor, a formula was created that lowered the amount of time allowed for the res
	// based on how many nodes were requested. More nodes = less reservation time. Doubtful we
	// will ever go back to that, but the node count is there if needed.

	if limit <= 0 {
		// no time limit defined, so return nil
		return nil
	}

	logger.Debug().Msgf("checkTimeLimit: requested res duration: %v", resDur)

	// lop off some seconds to ensure requesting max allowable time doesn't exceed limit by some tiny fraction
	if resDur-(time.Second*5) > limit {
		return fmt.Errorf("max allowable time is %s (you requested %s)", limit.Round(time.Second), resDur.Round(time.Second))
	}

	return nil
}

func meetsMinResDuration(duration time.Duration) bool {
	minReserveTime := time.Duration(igor.Scheduler.MinReserveTime) * time.Minute
	return duration >= minReserveTime
}

func getScheduleEnd(isElevated bool) time.Time {
	timeAllowed := MaxScheduleMinutes
	if isElevated {
		timeAllowed = MaxScheduleDays * 60 * 24
	}
	f := time.Now().Add(time.Minute * time.Duration(timeAllowed))
	return time.Date(f.Year(), f.Month(), f.Day(), f.Hour(), f.Minute(), 0, 0, time.Local)
}

func checkScheduleLimit(endTime time.Time, isElevated bool) error {
	schedEndFromNow := getScheduleEnd(isElevated)
	if endTime.After(schedEndFromNow) {
		if isElevated {
			return fmt.Errorf("reservation can't fall outside the maximum end datetime allowed %s",
				schedEndFromNow.Format(common.DateTimeCompactFormat))
		}
		return fmt.Errorf("reservation would fall outside igor's scheduling window that ends on %s",
			schedEndFromNow.Format(common.DateTimeCompactFormat))
	}
	return nil
}

func makeResGroupPermStrings(res *Reservation) []string {
	dpstr := NewPermissionString(PermReservations, res.Name, PermDeleteAction)
	epstr := NewPermissionString(PermReservations, res.Name, PermEditAction, "extend")

	return []string{dpstr, epstr}
}

// Create the permission string for allowing power commands to be performed on a group of hosts.
func makeNodePowerPerm(hostList []Host) string {
	var hostsStrList string

	sort.Slice(hostList, func(i, j int) bool {
		return hostList[i].Name < hostList[j].Name
	})

	for i := 0; i < len(hostList)-1; i++ {
		hostsStrList += hostList[i].Name + PermSubpartToken
	}
	hostsStrList += hostList[len(hostList)-1].Name

	return NewPermissionString(PermPowerAction, hostsStrList)
}

// resNamesOfResList returns a list of Reservation names from
// the provided list of reservations.
func resNamesOfResList(resList []Reservation) []string {
	resNames := make([]string, len(resList))
	for i, r := range resList {
		resNames[i] = r.Name
	}
	return resNames
}

// resIDsOfResList returns a list of Reservation IDs from
// the provided list of reservations.
func resIDsOfResList(res []Reservation) []int {
	resIDs := make([]int, len(res))
	for i, r := range res {
		resIDs[i] = r.ID
	}
	return resIDs
}

// determineNodeResetTime checks the image type and
// sets a resetEnd time based on the configured
// duration based on image time relative to the
// reservation end time
func determineNodeResetTime(resEnd time.Time) time.Time {
	resetEnd := resEnd.Add(time.Minute * time.Duration(igor.Config.Maintenance.HostMaintenanceDuration))
	return resetEnd
}

// getActiveReservation returns a Reservation the given host
// Host is associated with
func getActiveReservation(h *Host) *Reservation {
	for _, res := range h.Reservations {
		result := res
		if res.IsActive(time.Now()) {
			// this res is a shallow copy, we want the whole thing
			if err := performDbTx(func(tx *gorm.DB) error {
				reses, _, err := getReservations([]string{res.Name}, tx)
				if err != nil {
					return err
				}
				if len(reses) > 0 {
					result = reses[0]
					return nil
				}
				return fmt.Errorf("no reservations found")
			}); err != nil {
				return &res
			}
			return &result
		}
	}
	return nil
}
