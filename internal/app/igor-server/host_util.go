// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"time"
)

// getReservedHosts returns a list of hosts that are currently
// associated to a reservation
func getReservedHosts() ([]Host, error) {
	var result []Host
	reservations, err := dbReadReservationsTx(map[string]interface{}{}, map[string]time.Time{})
	if err != nil {
		return result, err
	}
	for _, reservation := range reservations {
		result = append(result, reservation.Hosts...)
	}
	return result, nil
}

// getHostIDsFromNames takes a list of host names and returns their IDs if all names are valid
func getHostIDsFromNames(hostNames []string) ([]int, int, error) {
	hostQuery := map[string]interface{}{"name": hostNames}
	if hostList, err := dbReadHostsTx(hostQuery); err != nil {
		return nil, http.StatusInternalServerError, err
	} else if len(hostNames) != len(hostList) {
		return nil, http.StatusBadRequest, fmt.Errorf("number of hosts retrieved from DB does not equal number of host names given")
	} else {
		return hostIDsOfHosts(hostList), http.StatusOK, nil
	}
}

// namesOfHosts returns a list of host names from
// the provided list of hosts.
func namesOfHosts(hosts []Host) []string {
	hostNames := make([]string, len(hosts))
	for i, h := range hosts {
		hostNames[i] = h.Name
	}
	return hostNames
}

// hostNamesOfHosts returns a list of host hostnames from
// the provided list of hosts.
func hostNamesOfHosts(hosts []Host) []string {
	hostNames := make([]string, len(hosts))
	for i, h := range hosts {
		hostNames[i] = h.HostName
	}
	return hostNames
}

// hostIDsOfHosts returns a list of host IDs from
// the provided list of hosts.
func hostIDsOfHosts(hosts []Host) []int {
	hostIDs := make([]int, len(hosts))
	for i, h := range hosts {
		hostIDs[i] = h.ID
	}
	return hostIDs
}

// hostSliceContains returns true if one of the hosts in the slice has the given name, false otherwise.
func hostSliceContains(hosts []Host, name string) (bool, Host) {
	for _, h := range hosts {
		if h.Name == name {
			return true, h
		}
	}
	return false, Host{}
}

func getHostIntersection(hostNames []string, hosts []Host) []Host {
	targetHosts := []Host{}
	for _, hostName := range hostNames {
		if contains, host := hostSliceContains(hosts, hostName); contains {
			targetHosts = append(targetHosts, host)
		}
	}
	return targetHosts
}

// removeHost removes the given target Host from the given slice of hosts.
func removeHost(hSlice []Host, target *Host) []Host {
	for idx, v := range hSlice {
		if v.Name == target.Name {
			return append(hSlice[0:idx], hSlice[idx+1:]...)
		}
	}
	return hSlice
}

// hasActiveReservations take a slice of hosts and returns the subset of hosts
// currently part of any active reservation.
func hasActiveReservations(hosts []Host) (activeHosts []Host) {
	reservations, err := dbReadReservationsTx(map[string]interface{}{}, map[string]time.Time{"to-start": time.Now()})
	if err != nil {
		logger.Error().Msgf("error getting active reservations - %v", err.Error())
		return
	}
	for _, res := range reservations {
		for _, host := range res.Hosts {
			if contains, _ := hostSliceContains(hosts, host.Name); contains {
				activeHosts = append(activeHosts, host)
			}
		}
	}
	return
}

func checkEthRules(ref string) error {
	if len(ref) == 0 {
		return fmt.Errorf("eth value name cannot be empty")
	}
	if !stdEthCheckPattern.MatchString(ref) {
		return fmt.Errorf("'%s' is not a legal ethernet (eth) value", ref)
	}
	return nil
}
