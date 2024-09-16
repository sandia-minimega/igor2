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

func doReadProfiles(queryParams map[string]interface{}) ([]Profile, int, error) {
	pList, err := dbReadProfilesTx(queryParams)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	} else {
		// user := getUserFromContext(r)
		// // remove profiles where owner is igor-admin that are not default profiles
		// if !userElevated(user.Name) {
		// 	for _, p := range pList {
		// 		if p.Owner.Name == IgorAdmin && !p.IsDefault {
		// 			pList = removeProfile(pList, &p)
		// 		}
		// 	}
		// }

		return pList, http.StatusOK, nil
	}
}

// parseProfileSearchParams takes the query map provided by the route and moves its expected
// parameters into a map[string]interface that can be passed directly to a GORM db query. Along
// the way it performs queries of other db objects specified in the search params (owner, group and distro) and
// will report failure if those objects are not found.
func parseProfileSearchParams(queryMap map[string][]string, r *http.Request) (map[string]interface{}, int, error) {

	status := http.StatusOK
	var err error
	var ownerList []User
	var distroList []Distro
	clog := hlog.FromRequest(r)
	queryParams := map[string]interface{}{}
	user := getUserFromContext(r)

	// if requesting user isn't admin/elevated, filter results by allowed distros
	allowedDistros, err := getAllowedDistros(user)
	if err != nil {
		logger.Warn().Msgf("error retrieving allowed distro for user during profile read action")
		return queryParams, http.StatusInternalServerError, err
	}

	// admin, code, err := getIgorAdminTx()
	// if err != nil {
	// 	return nil, code, err
	// }
	allowedOwners := []User{*user}

	// extract and convert each attribute, if present, and add to query
	for key, val := range queryMap {
		switch key {
		case "name":
			queryParams["name"] = val
		case "kernelArgs":
			queryParams["KernelArgs"] = val
		case "distro":
			// user may not search by distros they have no access to distro
			if !userElevated(user.Name) {
				for _, dName := range val {
					allowed := false
					for _, dis := range allowedDistros {
						if dis.Name == dName {
							allowed = true
						}
					}
					if !allowed {
						return queryParams, http.StatusConflict, fmt.Errorf("distro %v not available to user and may not be used for search", dName)
					}
				}
			}
			if distroList, status, err = doReadDistros(map[string]interface{}{"name": val}, nil); err != nil {
				return nil, status, err
			} else {
				queryParams["distro_id"] = distroIDsOfDistros(distroList)
			}
		case "owner":
			// user may not search profiles of other users than themselves or defaults from igor-admin
			// if they're not admins
			// user may not search by distros they have no access to
			if !userElevated(user.Name) {
				for _, oName := range val {
					allowed := false
					for _, own := range allowedOwners {
						if own.Name == oName {
							allowed = true
						}
					}
					if !allowed {
						return queryParams, http.StatusConflict, fmt.Errorf("user cannot search for profiles under owner name %v", oName)
					}
				}
			}
			if ownerList, status, err = doReadUsers(map[string]interface{}{"name": val}); err != nil {
				return nil, status, err
			} else {
				ownerIDs := userIDsOfUsers(ownerList)
				if userElevated(user.Name) {
					queryParams["owner_id"] = ownerIDs
				} else if len(ownerList) == 1 && ownerList[0].Name == user.Name {
					queryParams["owner_id"] = ownerIDs
				} else {
					return nil, http.StatusForbidden, fmt.Errorf("user cannot search other user's profiles")
				}
			}
		default:
			clog.Warn().Msgf("unrecognized search parameter '%s' with args '%v'", key, val)
		}
	}

	// if no distros were already specified, restrict search to user's allowed distros and owners if not an admin
	if !userElevated(user.Name) {
		if _, ok := queryParams["distro_id"].([]int); !ok {
			queryParams["distro_id"] = distroIDsOfDistros(allowedDistros)
		}
		if _, ok := queryParams["owner_id"].([]int); !ok {
			queryParams["owner_id"] = userIDsOfUsers(allowedOwners)
		}

	}

	return queryParams, status, nil
}

// getProfiles is a convenience method to perform a lookup of profiles based on list of provided names.
// It will be successful as long as at least one profile is found, otherwise it will return a NotFound error.
//
//	list,200,nil if any named profile found
//	nil,404,err if no named profile found
//	nil,500,err if db error
func getProfiles(profileNames []string, tx *gorm.DB) ([]Profile, int, error) {
	// profile search by name must always be in the context of their owner
	profiles, err := dbReadProfiles(map[string]interface{}{"name": profileNames}, tx)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	} else if len(profiles) == 0 {
		return nil, http.StatusNotFound, fmt.Errorf("profile(s) '%s' not found", strings.Join(profileNames, ","))
	}

	return profiles, http.StatusOK, nil
}

// profileExists will perform a simple query to see if a profile exists in the
// database. It will pass back any encountered GORM errors.
func profileExists(name string, tx *gorm.DB) (found bool, err error) {
	queryParams := map[string]interface{}{"name": name}
	pList, findErr := dbReadProfiles(queryParams, tx)
	if findErr != nil {
		return false, findErr
	}
	if len(pList) > 0 {
		return true, nil
	}
	return false, nil
}
