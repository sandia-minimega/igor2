// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"igor2/internal/pkg/common"

	"github.com/rs/zerolog/hlog"
)

// doReadReservations performs a DB lookup of Reservation records that match the provided queryParams. It will return
// these as a list which can also be empty/nil if no matches were found. It will also pass back any encountered GORM
// errors with status code 500.
func doReadReservations(queryParams map[string]interface{}, timeParams map[string]time.Time) ([]Reservation, int, error) {

	result, err := dbReadReservationsTx(queryParams, timeParams)
	if err != nil {
		return result, http.StatusInternalServerError, err
	}
	return result, http.StatusOK, nil
}

// parseResSearchParams should just be converting string inputs to the appropriate type
// using basic validation related to that purpose
func parseResSearchParams(queryMap map[string][]string, r *http.Request) (map[string]interface{}, map[string]time.Time, int, error) {

	status := http.StatusOK
	clog := hlog.FromRequest(r)

	queryParams := map[string]interface{}{}
	queryTimeParams := map[string]time.Time{}

	for key, val := range queryMap {

		switch key {
		case "name":
			// these can be passed directly as []string
			queryParams[key] = val
		case "owner":
			ownerQuery := map[string]interface{}{"name": val}
			if ownerList, status, err := doReadUsers(ownerQuery); err != nil {
				return nil, nil, status, err
			} else {
				queryParams["owner_id"] = userIDsOfUsers(ownerList)
			}
		case "distro":
			distroQuery := map[string]interface{}{"name": val}
			if distroList, status, err := doReadDistros(distroQuery, r); err != nil {
				return nil, nil, status, err
			} else {
				queryParams["distro_id"] = distroIDsOfDistros(distroList)
			}
		case "profile":
			profileQuery := map[string]interface{}{"name": val}
			if profileList, status, err := doReadProfiles(profileQuery); err != nil {
				return nil, nil, status, err
			} else {
				queryParams["profile_id"] = profileIDsOfProfiles(profileList)
			}
		case "group":
			groupQuery := map[string]interface{}{"name": val}
			if groupList, status, err := doReadGroups(groupQuery); err != nil {
				return nil, nil, status, err
			} else {
				queryParams["group_id"] = groupIDsOfGroups(groupList)
			}
		case "host":
			if hostIDs, status, err := getHostIDsFromNames(val); err != nil {
				return nil, nil, status, err
			} else {
				queryParams["host"] = hostIDs
			}
		case "installed":
			if val[0] == "0" {
				queryParams[key] = false
			} else {
				queryParams[key] = true
			}
		case "from-start", "to-end":
			dts, _ := time.Parse(common.DateTimeCompactFormat, val[0])
			queryTimeParams[key] = dts
		case "vlan":
			var vlanList []int
			for _, vlanNum := range val {
				vlan, _ := strconv.Atoi(vlanNum)
				vlanList = append(vlanList, vlan)
			}
			queryParams[key] = vlanList
		case "gte-extendNum", "lte-extendNum", "eq-extendNum", "gte-nodeCount", "lte-nodeCount", "nodeCount":
			num, _ := strconv.Atoi(val[0])
			queryParams["x-"+key] = num
		default:
			clog.Warn().Msgf("parameter '%s' with args '%v' not included in search", key, val)
		}
	}
	if len(queryTimeParams) == 0 {
		queryTimeParams = nil
	}
	return queryParams, queryTimeParams, status, nil
}

// getReservations is a convenience method to perform a lookup of reservations based on list of provided names.
// It will be successful as long as at least one reservation is found, otherwise it will return a NotFound error.
//
//	resList,200,nil if any named reservation found
//	nil,404,err if no named reservation found
//	nil,500,err if db error
func getReservations(resNames []string, tx *gorm.DB) ([]Reservation, int, error) {

	rList, err := dbReadReservations(map[string]interface{}{"name": resNames}, nil, tx)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	} else if len(rList) == 0 {
		return nil, http.StatusNotFound, fmt.Errorf("reservation(s) '%s' not found", strings.Join(resNames, ","))
	}

	return rList, http.StatusOK, nil
}

// resvExists will perform a simple query to see if a reservation exists in the
// database. It will pass back any encountered GORM errors.
func resvExists(name string, tx *gorm.DB) (found bool, err error) {
	queryParams := map[string]interface{}{"name": name}
	rList, findErr := dbReadReservations(queryParams, nil, tx)
	if findErr != nil {
		return false, findErr
	}
	if len(rList) > 0 {
		return true, nil
	}
	return false, nil
}
