// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"github.com/rs/zerolog/hlog"
)

// doReadHosts performs a DB lookup of Host records that match the provided queryParams. It will
// return these as a list which can also be empty/nil if no matches were found. It will also pass back any
// encountered GORM errors with status code 500.
func doReadHosts(queryParams map[string]interface{}) ([]Host, int, error) {
	hList, err := dbReadHostsTx(queryParams)
	if err != nil {
		return hList, http.StatusInternalServerError, err
	} else {
		return hList, http.StatusOK, nil
	}
}

// getHostsTx runs getHosts within a new transaction.
func getHostsTx(hostNames []string, findAll bool) (hList []Host, status int, err error) {
	status = http.StatusInternalServerError
	err = performDbTx(func(tx *gorm.DB) error {
		hList, status, err = getHosts(hostNames, findAll, tx)
		return err
	})
	return
}

// getHosts is a convenience method to perform a lookup of hosts based on list of provided names.
//
// The findAll parameter is useful when the lookup queries multiple names. It has no effect for single
// name lookups.
//
// If findAll is false it will be successful as long as at least one host is found, otherwise it will
// return a NotFound error. If findAll is true, then every host must be found, otherwise it will
// return a NotFound error specifying which hosts couldn't be located.
//
//	list,200,nil if named hosts found
//	nil,404,err if any named host not found (findAll = true)
//	nil,404,err if no named hosts found (findAll = false)
//	nil,500,err if db error
func getHosts(hostNames []string, findAll bool, tx *gorm.DB) ([]Host, int, error) {

	found, err := dbReadHosts(map[string]interface{}{"name": hostNames}, tx)

	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if len(hostNames) == 1 && len(found) == 0 {
		return nil, http.StatusNotFound, fmt.Errorf("host '%s' not found", hostNames[0])
	}

	if findAll {
		var inList bool
		var notFound []string
		for _, v := range hostNames {
			inList = false
			for _, u := range found {
				if v == u.Name {
					inList = true
					continue
				}
			}
			if !inList {
				notFound = append(notFound, v)
			}
		}

		if len(notFound) != 0 {
			return nil, http.StatusNotFound, fmt.Errorf("host(s) '%s' not found", strings.Join(notFound, ","))
		}
	} else {
		if len(found) == 0 {
			return nil, http.StatusNotFound, fmt.Errorf("host(s) '%s' not found", strings.Join(hostNames, ","))
		}
	}

	return found, http.StatusOK, nil
}

// hostExists will perform a simple query to see if a host exists in the
// database. It will pass back any encountered GORM errors.
func hostExists(name string, tx *gorm.DB) (found bool, err error) {
	queryParams := map[string]interface{}{"name": name}
	hList, findErr := dbReadHosts(queryParams, tx)
	if findErr != nil {
		return false, findErr
	}
	if len(hList) > 0 {
		return true, nil
	}
	return false, nil
}

func parseHostSearchParams(queryMap map[string][]string, r *http.Request) (map[string]interface{}, int, error) {

	clog := hlog.FromRequest(r)
	queryParams := map[string]interface{}{}
	// extract and convert each attribute, if present, and add to query

	var nameRange []string
	if len(queryMap["name"]) > 0 {
		for _, n := range queryMap["name"] {
			nList := igor.splitRange(n)
			nameRange = append(nameRange, nList...)
		}
	}

	for key, val := range queryMap {
		switch key {
		case "name":
			queryParams["name"] = nameRange
		case "eth":
			queryParams["eth"] = val
		case "hostname":
			queryParams["host_name"] = val
		case "ip":
			queryParams["ip"] = val
		case "state":
			var stateList []HostState
			for i := range val {
				stateList = append(stateList, resolveHostState(strings.TrimSpace(val[i])))
			}
			queryParams["state"] = stateList
		case "hostPolicy":
			policyQuery := map[string]interface{}{"name": val}
			if policyList, err := dbReadHostPoliciesTx(policyQuery, clog); err != nil {
				return nil, http.StatusInternalServerError, err
			} else {
				queryParams["host_policy_id"] = hostPolicyIDsOfHostPolicies(policyList)
			}
		case "reservation":
			resQuery := map[string]interface{}{"name": val}
			if resList, err := dbReadReservationsTx(resQuery, nil); err != nil {
				return nil, http.StatusInternalServerError, err
			} else {
				queryParams["reservations"] = resIDsOfResList(resList)
			}
		default:
			clog.Warn().Msgf("unrecognized search parameter '%s' with args '%v'", key, val)
		}
	}

	return queryParams, http.StatusOK, nil
}
