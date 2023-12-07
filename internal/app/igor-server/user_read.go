// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"
)

// doReadUsers performs a DB lookup of User records that match the provided queryParams. It will return these as
// a list which can also be empty/nil if no matches were found. It will also pass back any encountered GORM
// errors with status code 500.
func doReadUsers(queryParams map[string]interface{}) ([]User, int, error) {

	uList, err := dbReadUsersTx(queryParams)
	if err != nil {
		return uList, http.StatusInternalServerError, err
	} else {
		return uList, http.StatusOK, nil
	}
}

// getUsersTx performs getUsers within a new transaction.
func getUsersTx(names []string, findAll bool) (uList []User, status int, err error) {
	status = http.StatusInternalServerError
	err = performDbTx(func(tx *gorm.DB) error {
		uList, status, err = getUsers(names, findAll, tx)
		return err
	})
	return uList, status, err
}

// getUsers is a convenience method to perform a lookup of users based on a list of provided names and
// does this within an existing database transaction.
//
// The findAll parameter is useful when the lookup queries multiple names. It has no effect for single
// name lookups.
//
// If findAll is false then the call is successful as long as at least one user is found. Other user names
// that did not have matching records are silently ignored going forward. If findAll is true, then
// every user in the list must be found, otherwise the returned error specifies which users couldn't
// be located.
//
// Returns:
//
//	list,200,nil if successful
//	nil,404,err if any named user not found (findAll = true)
//	nil,404,err if no named users found (findAll = false)
//	nil,500,err if there was a database error
func getUsers(names []string, findAll bool, tx *gorm.DB) ([]User, int, error) {

	found, err := dbReadUsers(map[string]interface{}{"name": names}, tx)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if len(names) == 1 && len(found) == 0 {
		return nil, http.StatusNotFound, fmt.Errorf("user '%s' not found", names[0])
	}

	if findAll {
		notFound := filterNonMembers(found, names)

		if len(notFound) != 0 {
			return nil, http.StatusNotFound, fmt.Errorf("user(s) '%s' not found", strings.Join(notFound, ","))
		}
	} else {
		if len(found) == 0 {
			return nil, http.StatusNotFound, fmt.Errorf("user(s) '%s' not found", strings.Join(names, ","))
		}
	}

	return found, http.StatusOK, nil
}

// getIgorAdminTx is a shortcut method that returns the 'igor-admin' user via a new transaction.
func getIgorAdminTx() (admin *User, status int, err error) {
	status = http.StatusInternalServerError
	err = performDbTx(func(tx *gorm.DB) error {
		admin, status, err = getIgorAdmin(tx)
		return err
	})
	return
}

// getIgorAdmin is a shortcut method that returns the 'igor-admin' user.
func getIgorAdmin(tx *gorm.DB) (admin *User, status int, err error) {
	var uList []User
	uList, status, err = getUsers([]string{IgorAdmin}, true, tx)
	if err != nil {
		return nil, status, err
	}
	return &uList[0], http.StatusOK, err
}

// userExists performs a simple query to see if a username exists in the database. It will pass back
// any encountered GORM errors.
func userExists(name string, tx *gorm.DB) (ok bool, err error) {
	queryParams := map[string]interface{}{"name": name}
	found, findErr := dbReadUsers(queryParams, tx)
	if findErr != nil {
		return false, findErr
	}
	if len(found) > 0 {
		return true, nil
	}
	return false, nil
}

// checkUniqueUserAttributes verifies both the name and the email are unique to the given params
func checkUniqueUserAttributes(username, email string) (ok bool, status int, err error) {
	status = http.StatusInternalServerError
	ok = true
	if err = performDbTx(func(tx *gorm.DB) error {
		exists, ueErr := userExists(username, tx)
		if ueErr != nil {
			return ueErr // uses default err status
		}
		if exists {
			status = http.StatusConflict
			return fmt.Errorf("user '%s' already exists", username)
		} else {
			emailList, emErr := dbReadUsers(map[string]interface{}{"email": email}, tx)
			if emErr != nil {
				return emErr // uses default err status
			}
			if len(emailList) > 0 {
				status = http.StatusConflict
				return fmt.Errorf("email '%s' already used by '%s'", email, emailList[0].Name)
			}
		}
		status = http.StatusOK
		return nil
	}); err != nil {
		ok = false
	}
	return ok, status, err
}

func parseUserSearchParams(queryMap map[string][]string, r *http.Request) (map[string]interface{}, int, error) {
	clog := hlog.FromRequest(r)
	queryParams := map[string]interface{}{}

	for key, val := range queryMap {
		switch key {
		case "name":
			queryParams[key] = val
		default:
			clog.Warn().Msgf("parameter '%s' with args '%v' not included in search", key, val)
		}
	}

	return queryParams, http.StatusOK, nil
}
