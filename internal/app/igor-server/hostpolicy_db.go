// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	zl "github.com/rs/zerolog"

	"igor2/internal/pkg/common"

	"gorm.io/gorm"
)

func dbCreateHostPolicy(policy *HostPolicy, tx *gorm.DB) error {
	result := tx.Create(&policy)
	return result.Error
}

func dbReadHostPoliciesTx(queryParams map[string]interface{}, clog *zl.Logger) (policyList []HostPolicy, err error) {
	err = performDbTx(func(tx *gorm.DB) error {
		policyList, err = dbReadHostPolicies(queryParams, tx, clog)
		return err
	})

	return policyList, err
}

func dbReadHostPolicies(queryParams map[string]interface{}, tx *gorm.DB, clog *zl.Logger) (policies []HostPolicy, err error) {

	tx = tx.Preload("AccessGroups").Preload("Hosts")

	// if no params given, return all host policies
	if len(queryParams) == 0 {
		result := tx.Find(&policies)
		return policies, result.Error
	}

	for key, val := range queryParams {
		switch val.(type) {
		case string, int:
			tx = tx.Where(key, val)
		case []int:
			if strings.ToLower(key) == "access_groups" || strings.ToLower(key) == "accessgroups" {
				tx = tx.Joins("JOIN groups_policies ON groups_policies.host_policy_id = ID AND group_id IN ?", val)
			} else if strings.ToLower(key) == "hosts" {
				tx = tx.Joins("JOIN hosts ON hosts.host_policy_id = host_policies.ID AND hosts.id IN ?", val)
			} else {
				tx = tx.Where(key+" IN ?", val)
			}
		case []string:
			tx = tx.Where(key+" IN ?", val)
		default:
			clog.Error().Msgf("dbReadHostPolicies: incorrect parameter type %T received for %s: %v", val, key, val)
		}
	}

	result := tx.Group("host_policies.name").Find(&policies) // group-by required for no dupes
	return policies, result.Error
}

// dbEditHostPolicy iterates through a HostPolicy list applying the same changes to each.
func dbEditHostPolicy(targets []HostPolicy, changes map[string]interface{}, tx *gorm.DB) error {

	for _, h := range targets {

		if name, ok := changes["name"]; ok {
			// if result := db.Model(&h).Update("Name", name); result.Error != nil {
			// 	return result.Error
			// }
			h.Name = name.(string)
		}
		if maxResTime, ok := changes["maxResTime"]; ok {
			// if result := db.Model(&h).Update("MaxResTime", maxResTime); result.Error != nil {
			// 	return result.Error
			// }
			h.MaxResTime = maxResTime.(time.Duration)
		}
		policyGroups := h.AccessGroups
		if remGroups, ok := changes["removeGroups"]; ok {
			rGroups := remGroups.([]Group)
			for _, group := range rGroups {
				policyGroups = removeGroup(policyGroups, &group)
				if daErr := tx.Model(&h).Association("AccessGroups").Delete(group); daErr != nil {
					return daErr
				}
			}
			h.AccessGroups = policyGroups
		}

		if addGroups, ok := changes["addGroups"]; ok {
			aGroups := addGroups.([]Group)
			for _, group := range aGroups {
				if !groupSliceContains(policyGroups, group.Name) {
					policyGroups = append(policyGroups, group)
				}
			}
			// if 2+ groups are now assigned to the policy and one of them
			// is all, then remove the all group.
			if (len(policyGroups) > 1) && groupSliceContains(policyGroups, GroupAll) {
				all, _, agErr := getAllGroupTx()
				if agErr != nil {
					return agErr
				}
				policyGroups = removeGroup(policyGroups, all)
				if daErr := tx.Model(&h).Association("AccessGroups").Delete(all); daErr != nil {
					return daErr
				}
			}
			h.AccessGroups = policyGroups
		}

		// if we removed a group, didn't add one, and now we have no group assigned
		// then add the all group
		if len(h.AccessGroups) == 0 {
			allGroup, _, err := getAllGroup(tx)
			if err != nil {
				return err
			}
			h.AccessGroups = []Group{*allGroup}
		}

		if addSBs, ok := changes["addNotAvailable"]; ok {
			mySBs := h.NotAvailable
			for _, sba := range addSBs.(ScheduleBlockArray) {
				duplicate := false
				for _, mySB := range mySBs {
					if mySB.Start == sba.Start && mySB.Duration == sba.Duration {
						duplicate = true
						break
					}
				}
				if !duplicate {
					mySBs = append(mySBs, sba)
				}
			}
			h.NotAvailable = mySBs
		}

		if removeSBs, ok := changes["removeNotAvailable"]; ok {
			for _, sbi := range removeSBs.(ScheduleBlockArray) {
				h.NotAvailable = h.removeSBInstance(sbi)
			}
		}
		// save any changes made
		if result := tx.Save(&h); result.Error != nil {
			return result.Error
		}
	}
	return nil
}

// dbDeleteHostPolicy removes the given host policy from the DB
func dbDeleteHostPolicy(target *HostPolicy, tx *gorm.DB) error {

	if daErr := tx.Model(&target).Association("AccessGroups").Clear(); daErr != nil {
		return daErr
	}
	if result := tx.Delete(&target); result.Error != nil {
		return result.Error
	}

	return nil
}

// dbCheckHostPolicyConflicts determines whether the given list of hosts are associated with
// hot policies conflict with the given access-group and time-window parameters for a new
// or extending reservation with named hosts.
//
//	500/ServerError if there was an internal problem.
//	409/Conflict if one or more policies were found to conflict with the given access groups or time window.
//	200/OK if no conflicts were found.
func dbCheckHostPolicyConflicts(hostNames []string, groupAccessList []string, isElevated bool,
	startTime time.Time, currentEndTime time.Time, newEndTime time.Time, clog *zl.Logger) (int, error) {
	// get all policies associated with the given list of host names
	myHostPolicies, err := getHostPoliciesFromHostNames(hostNames)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// determine if any policies do not contain at least one group from groupAccessList
	if membership, policy := dbCheckHostPolicyGroupConflicts(myHostPolicies, groupAccessList); !membership {
		// get the intersection of affected policy hosts and requested hosts
		offendingHosts := getHostIntersection(hostNames, policy.Hosts)
		return http.StatusConflict, &HostPolicyConflictError{"", true, false, false, time.Time{}, time.Time{}, offendingHosts}
	}

	// determine if any policies conflict based on maxResDuration or unavailability
	totalResDuration := newEndTime.Sub(startTime)
	for _, policy := range myHostPolicies {
		clog.Debug().Msgf("checking HostPolicy: %s", policy.Name)
		// check that each host can support the desired reservation duration
		if !isElevated {
			if err = checkTimeLimit(len(hostNames), policy.MaxResTime, totalResDuration); err != nil {
				clog.Warn().Msgf("%v", err)
				// get the intersection of affected policy hosts and requested hosts
				offendingHosts := getHostIntersection(hostNames, policy.Hosts)
				return http.StatusConflict, &HostPolicyConflictError{err.Error(), false, true, false, time.Time{}, time.Time{}, offendingHosts}
			}
		}
		// iterate through any policy ScheduleBlocks to determine if a conflict exists with the given times
		// If context is an extend request, use reservation's current end time as the "start" time for the policy check
		contextStart := startTime
		if currentEndTime.Before(newEndTime) {
			contextStart = currentEndTime
		}
		if conflict, start, end := hasScheduleBlockConflict(policy.NotAvailable, contextStart, newEndTime, clog); conflict {
			// get the intersection of affected policy hosts and requested hosts
			offendingHosts := getHostIntersection(hostNames, policy.Hosts)
			return http.StatusConflict, &HostPolicyConflictError{"", false, false, true, start, end, offendingHosts}
		}
	}
	return http.StatusOK, nil
}

func dbCheckHostPolicyGroupConflicts(hostPolicies []HostPolicy, groupAccessList []string) (bool, HostPolicy) {
	// determine if any policies do not contain at least one group from groupAccessList
	for _, policy := range hostPolicies {
		logger.Debug().Msgf("Looking at policy: %s", policy.Name)
		policyGroups := policy.AccessGroups
		membership := false
		logger.Debug().Msgf("Policy groups: %v", policyGroups)
		for _, userGroup := range groupAccessList {
			logger.Debug().Msgf("Checking user group: %s", userGroup)
			if groupSliceContains(policyGroups, userGroup) {
				membership = true
				continue
			}
		}
		if !membership {
			return false, policy
		}
	}
	return true, HostPolicy{}
}

// dbGetAccessibleHosts determines and returns the Host collections associated with a HostPolicy that
// does not conflict with the given accessGroupList, startTime or endTime.
func dbGetAccessibleHosts(accessGroupList []string, isElevated bool, startTime, endTime time.Time, numHostsReq int, tx *gorm.DB, clog *zl.Logger) (map[string][]Host, int, error) {

	// get all the hostPolicies that contain at least one of the given accessGroups
	groupsIDs, status, err := getGroupIDsFromNames(accessGroupList)
	if err != nil {
		return nil, status, err
	}
	potentialPolicies, err := dbReadHostPolicies(map[string]interface{}{"access_groups": groupsIDs}, tx, clog)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	exceededTimes := make([]bool, len(potentialPolicies))
	maxPolicyTime := time.Duration(0)
	validPolicyIDs := map[string]int{} // potential hostpolicy is valid if maxResDuration > givenDuration and no ScheduleBlock conflicts exist
	givenDuration := endTime.Sub(startTime)
	for i, policy := range potentialPolicies {

		// check for legal policy MaxResTime duration
		if !isElevated {
			if policy.MaxResTime > maxPolicyTime {
				maxPolicyTime = policy.MaxResTime
			}
			if err = checkTimeLimit(numHostsReq, policy.MaxResTime, givenDuration); err != nil {
				exceededTimes[i] = true
				continue
			}
		}

		if policy.Name == DefaultPolicyName {
			// we don't need to check for schedule blocks on the default policy
			validPolicyIDs[policy.Name] = policy.ID
			continue
		}

		// iterate through any policy ScheduleBlocks to determine if a conflict exists with the given times
		if conflict, _, _ := hasScheduleBlockConflict(policy.NotAvailable, startTime, endTime, clog); !conflict {
			validPolicyIDs[policy.Name] = policy.ID
		}
	}

	// check if we exceeded the max time constraints on all policies
	finalTimeExceeded := true
	for _, et := range exceededTimes {
		finalTimeExceeded = finalTimeExceeded && et
	}
	if finalTimeExceeded {
		return nil, http.StatusConflict, fmt.Errorf("max allowable time is %s (you requested %s)", maxPolicyTime.Round(time.Second), givenDuration.Round(time.Second))
	}

	// collect all hosts attached to valid hostPolicies and states
	validStates := []HostState{HostAvailable, HostReserved}

	validAccessHosts := make(map[string][]Host)
	totalValidHosts := 0

	for key, id := range validPolicyIDs {
		if hosts, rhErr := dbReadHosts(map[string]interface{}{"host_policy_id": id, "state": validStates}, tx); rhErr != nil {
			return nil, http.StatusInternalServerError, rhErr
		} else {
			validAccessHosts[key] = hosts
			totalValidHosts += len(hosts)
		}
	}

	if numHostsReq > totalValidHosts {
		return nil, http.StatusConflict, fmt.Errorf("reservation requires more cluster nodes than available")
	}

	return validAccessHosts, http.StatusOK, nil
}

func hasScheduleBlockConflict(sba ScheduleBlockArray, start time.Time, end time.Time, clog *zl.Logger) (bool, time.Time, time.Time) {
	for _, sb := range sba {
		sbDuration, _ := common.ParseDuration(sb.Duration)
		sbStart, _ := parseSBInstance(sb.Start)
		// start our instance search at the given start time - the current scheduleblock's duration
		startingPoint := start.Add(sbDuration * -1)
		nextInstanceStart := sbStart.Next(startingPoint)
		nextInstanceEnd := nextInstanceStart.Add(sbDuration)
		clog.Debug().Msgf("checking start: %v", start)
		clog.Debug().Msgf("and end: %v", end)
		// check every instance of the current scheduleblock until we know we're beyond the given end time
		for nextInstanceStart.Before(end) {
			clog.Debug().Msgf("sbniStart: %v", nextInstanceStart)
			clog.Debug().Msgf("sbniEnd: %v", nextInstanceEnd)

			// if the scheduleblock start/end overlaps with the given start/end, report a conflict
			if !((nextInstanceStart.Before(start) && nextInstanceEnd.Before(start)) || (nextInstanceStart.After(end) && nextInstanceEnd.After(end))) {
				clog.Debug().Msgf("Conflict detected")
				return true, nextInstanceStart, nextInstanceEnd
			}
			nextInstanceStart = sbStart.Next(nextInstanceEnd)
			nextInstanceEnd = nextInstanceStart.Add(sbDuration)
		}
	}
	return false, time.Time{}, time.Time{}
}
