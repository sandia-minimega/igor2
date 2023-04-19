// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

// doReadDistros performs a DB lookup of Distro records that match the provided queryParams. It will
// return these as a list which can also be empty/nil if no matches were found. It will also pass back any
// encountered GORM errors with status code 500.
func doReadDistros(queryMap map[string]interface{}, r *http.Request) (distros []Distro, status int, err error) {

	user := getUserFromContext(r)
	lookingForDefault, ok := queryMap["is_default"].(bool)
	if ok && lookingForDefault && !userElevated(user.Name) {
		return nil, http.StatusBadRequest, fmt.Errorf("must be elevated to search for default distro")
	}

	distros, err = dbReadDistrosTx(queryMap)
	if err != nil {
		return distros, http.StatusInternalServerError, err
	}

	// filter the distro search to what is allowed for the user if not elevated
	distros = scopeDistrosToUser(distros, user)

	return distros, http.StatusOK, nil
}

func parseDistroReadParams(queryMap map[string][]string) (map[string]interface{}, int, error) {
	searchParams := make(map[string]interface{})

	for key, val := range queryMap {

		switch key {
		case "name", "description", "kernel", "initrd":
			// these can be passed directly as []string
			searchParams[key] = val
		case "ki-hash":
			searchParams["ki_hash"] = val
		case "kernelArgs":
			searchParams["kernel_args"] = val
		case "owner":
			// convert owner name to User object
			owners, status, err := doReadUsers(map[string]interface{}{"name": val})
			if err != nil {
				return searchParams, status, err
			} else {
				searchParams["owner_id"] = userIDsOfUsers(owners)
			}
		case "group":
			// covert group name(s) to group objects
			if groups, status, err := doReadGroups(map[string]interface{}{"name": val}); err != nil {
				return searchParams, status, err
			} else if len(groups) > 0 {
				searchParams["groups"] = groupIDsOfGroups(groups)
			}
		case "default":
			if val[0] == "true" {
				searchParams["is_default"] = true
			}
		default:
			return searchParams, http.StatusBadRequest, fmt.Errorf("cannot search for distro with a %s parameter at this time", key)
		}
	}
	if len(searchParams) == 0 && len(queryMap) > 0 {
		return searchParams, http.StatusNotFound, nil
	}
	return searchParams, http.StatusOK, nil
}

// getDistrosTx runs getDistros within a new transaction.
func getDistrosTx(distroNames []string) ([]Distro, int, error) {

	distros, err := dbReadDistrosTx(map[string]interface{}{"name": distroNames})
	if err != nil {
		return nil, http.StatusInternalServerError, err
	} else if len(distros) == 0 {
		return nil, http.StatusNotFound, fmt.Errorf("distro(s) '%s' not found", strings.Join(distroNames, ","))
	}

	return distros, http.StatusOK, nil
}

// getDistros is a convenience method to perform a lookup of distros based on list of provided names.
// It will be successful as long as at least one distro is found, otherwise it will return a NotFound error.
//
//	list,200,nil if any named distro found
//	nil,404,err if no named distro found
//	nil,500,err if db error
func getDistros(distroNames []string, tx *gorm.DB) ([]Distro, int, error) {

	distros, err := dbReadDistros(map[string]interface{}{"name": distroNames}, tx)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	} else if len(distros) == 0 {
		return nil, http.StatusNotFound, fmt.Errorf("distro(s) '%s' not found", strings.Join(distroNames, ","))
	}

	return distros, http.StatusOK, nil
}

// distroExists will perform a simple query to see if a distro exists in the
// database. It will pass back any encountered GORM errors.
func distroExists(name string, tx *gorm.DB) (found bool, err error) {
	queryParams := map[string]interface{}{"name": name}
	dList, findErr := dbReadDistros(queryParams, tx)
	if findErr != nil {
		return false, findErr
	}
	if len(dList) > 0 {
		return true, nil
	}
	return false, nil
}
