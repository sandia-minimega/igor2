// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	zl "github.com/rs/zerolog"
	"gorm.io/gorm"
	"net/http"
	"strings"

	"github.com/rs/zerolog/hlog"
)

// doReadHostPolicies performs a DB lookup of HostPolicy records that match the provided queryParams. It will
// return these as a list which can also be empty/nil if no matches were found. It will also pass back any
// encountered GORM errors with status code 500.
func doReadHostPolicies(queryParams map[string]interface{}, r *http.Request) ([]HostPolicy, int, error) {
	clog := hlog.FromRequest(r)
	hpList, err := dbReadHostPoliciesTx(queryParams, clog)
	if err != nil {
		return hpList, http.StatusInternalServerError, err
	}

	return hpList, http.StatusOK, nil
}

// getHostPolicies is a convenience method to perform a lookup of host policies based on list of provided names.
// It will be successful as long as at least one policy is found, otherwise it will return a NotFound error.
//
//	list,200,nil if any named host policy found
//	nil,404,err if no named host policy found
//	nil,500,err if db error
func getHostPolicies(hpNames []string, tx *gorm.DB, clog *zl.Logger) ([]HostPolicy, int, error) {

	hpList, err := dbReadHostPolicies(map[string]interface{}{"name": hpNames}, tx, clog)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	} else if len(hpList) == 0 {
		return nil, http.StatusNotFound, fmt.Errorf("policy(s) '%s' not found", strings.Join(hpNames, ","))
	}

	return hpList, http.StatusOK, nil
}

// hostPolicyExists will perform a simple query to see if a host policy exists in the
// database. It will pass back any encountered GORM errors.
func hostPolicyExists(name string, tx *gorm.DB, clog *zl.Logger) (ok bool, err error) {
	queryParams := map[string]interface{}{"name": name}
	found, findErr := dbReadHostPolicies(queryParams, tx, clog)
	if findErr != nil {
		return false, findErr
	}
	if len(found) > 0 {
		return true, nil
	}
	return false, nil
}

func parseHostPolicySearchParams(queryMap map[string][]string, r *http.Request) (map[string]interface{}, int, error) {
	// parse resParams and convert []string vals to proper corresponding types
	// template: db.Where(map[string]interface{}{"name": "jinzhu", "age": 20}).Find(&users)

	clog := hlog.FromRequest(r)
	status := http.StatusOK

	queryParams := map[string]interface{}{}
	// extract and convert each attribute, if present, and add to query

	for key, val := range queryMap {
		switch key {
		case "name":
			queryParams["name"] = val
		case "accessGroups":
			if groupIDs, status, err := getGroupIDsFromNames(val); err != nil {
				return nil, status, err
			} else {
				queryParams["access_groups"] = groupIDs
			}
		case "hosts":
			if hostIDs, status, err := getHostIDsFromNames(val); err != nil {
				return nil, status, err
			} else {
				queryParams["hosts"] = hostIDs
			}
		default:
			clog.Warn().Msgf("unrecognized search parameter '%s' with args '%v'", key, val)
		}
	}

	return queryParams, status, nil
}
