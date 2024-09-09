// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/zerolog/hlog"
)

// doReadGroups performs a DB lookup of Group records that match the provided queryParams. It will return these as
// a list which can also be empty/nil if no matches were found. It will also pass back any encountered GORM
// errors with status code 500.
func doReadGroups(queryParams map[string]interface{}) ([]Group, int, error) {

	found, err := dbReadGroupsTx(queryParams, true)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return found, http.StatusOK, err
}

func parseGroupSearchParams(queryMap map[string][]string, r *http.Request) (map[string]interface{}, int, error) {

	clog := hlog.FromRequest(r)
	queryParams := map[string]interface{}{}
	var status int
	var err error
	var ownerList []User

	for key, val := range queryMap {
		switch key {
		case "name":
			// these can be passed directly as []string
			queryParams[key] = val
		case "owner":
			if ownerList, status, err = doReadUsers(map[string]interface{}{"name": val}); err != nil {
				return nil, status, err
			} else {
				queryParams["owners"] = userIDsOfUsers(ownerList)
			}
		case "showMembers":
			showMembers, _ := strconv.ParseBool(val[0])
			queryParams["showMembers"] = showMembers
		default:
			clog.Warn().Msgf("parameter '%s' with args '%v' not included in search", key, val)
		}
	}

	return queryParams, status, nil
}

// getGroupsTx does the same thing as getGroups within a new transaction.
func getGroupsTx(groupNames []string, excludePugs bool) (gList []Group, status int, err error) {
	status = http.StatusInternalServerError
	err = performDbTx(func(tx *gorm.DB) error {
		gList, status, err = getGroups(groupNames, excludePugs, tx)
		return err
	})
	return
}

// getGroups is a convenience method to perform a lookup of groups based on list of provided names.
// If no search parameters are given, it will return all groups in the database. The excludePugs
// parameter modifies the search to either exclude (true) or include (false) user-private groups
// in the results. As such if there is any reason that groupNames would include a PUG this parameter
// must be set to false in order to succeed.
//
// Returns:
//
//	list,200,nil if all groups are found
//	nil,404,err if any group is not found
//	nil,500,err if there was a database error
func getGroups(groupNames []string, excludePugs bool, tx *gorm.DB) ([]Group, int, error) {

	found, err := dbReadGroups(map[string]interface{}{"name": groupNames}, excludePugs, tx)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	} else if len(groupNames) == 1 && len(found) == 0 {
		return nil, http.StatusNotFound, fmt.Errorf("group '%s' not found", groupNames[0])
	} else {

		var inList bool
		var notFound []string
		for _, v := range groupNames {
			inList = false
			for _, g := range found {
				if v == g.Name {
					inList = true
					continue
				}
			}
			if !inList {
				notFound = append(notFound, v)
			}
		}

		if len(notFound) != 0 {
			return nil, http.StatusNotFound, fmt.Errorf("group(s) '%s' not found", strings.Join(notFound, ","))
		}
	}

	return found, http.StatusOK, nil
}

// getAllGroupTx is a shortcut method that returns the 'all' group in a new transaction.
func getAllGroupTx() (allGroup *Group, status int, err error) {
	status = http.StatusInternalServerError
	err = performDbTx(func(tx *gorm.DB) error {
		allGroup, status, err = getAllGroup(tx)
		return err
	})
	return
}

// getAllGroup is a shortcut method that returns the 'all' group.
func getAllGroup(tx *gorm.DB) (allGroup *Group, status int, err error) {
	var gList []Group
	gList, status, err = getGroups([]string{GroupAll}, true, tx)
	if err != nil {
		return nil, status, err
	}
	return &gList[0], http.StatusOK, err
}

// groupExists will perform a simple query to see if a group exists in the
// database. It will pass back any encountered GORM errors.
func groupExists(name string, tx *gorm.DB) (found bool, err error) {
	queryParams := map[string]interface{}{"name": name}
	gList, findErr := dbReadGroups(queryParams, true, tx)
	if findErr != nil {
		return false, findErr
	}
	if len(gList) > 0 {
		return true, nil
	}
	return false, nil
}
