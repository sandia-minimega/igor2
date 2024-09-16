// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"
	"net/http"
)

// doDeleteUser steps through the process of making an update to a user record.
//
// Returns:
//
//	200,nil if delete was successful
//	403,err if tried to delete a system account
//	404,err if user could not be found
//	409,err if delete was not allowed due to its current state
//	500,err if an internal error occurred
func doDeleteUser(username string, r *http.Request) (status int, err error) {

	if err = denySysUserDelete(username); err != nil {
		return http.StatusForbidden, err
	}

	if getUserFromContext(r).Name == username {
		return http.StatusForbidden,
			fmt.Errorf("user not allowed to delete self - do this as %s or another admin", IgorAdmin)
	}

	// remove user from elevate map if they are in it
	igor.ElevateMap.Remove(username)
	clog := hlog.FromRequest(r)
	status = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		// fetch the user record
		userList, guStatus, guErr := getUsers([]string{username}, true, tx)
		if guErr != nil {
			status = guStatus
			return guErr
		}
		user := &userList[0]

		//  *** check for deletion conflicts with owned resources ***

		ogList := user.singleOwnedGroups()
		clog.Debug().Msgf("checking for '%s' owned groups", username)
		if len(ogList) > 0 {
			var ogNames []string
			for _, og := range ogList {
				ogNames = append(ogNames, og.Name)
			}
			status = http.StatusConflict
			return fmt.Errorf("cannot delete user - remove or transfer ownership of groups in list first: %v", ogNames)
		}

		searchByOwnerID := map[string]interface{}{"owner_id": user.ID}

		clog.Debug().Msgf("checking for '%s' current and future reservations", username)
		if orList, orErr := dbReadReservations(searchByOwnerID, nil, tx); orErr != nil {
			return orErr // uses default err status
		} else {
			if len(orList) > 0 {
				var orNames []string
				for _, or := range orList {
					orNames = append(orNames, or.Name)
				}
				status = http.StatusConflict
				return fmt.Errorf("cannot delete user - remove or transfer ownership of reservations in list first: %v", orNames)
			}
		}

		clog.Debug().Msgf("checking for '%s' owned distros", username)
		if odList, rdErr := dbReadDistros(searchByOwnerID, tx); rdErr != nil {
			return rdErr // uses default err status
		} else {
			if len(odList) > 0 {
				var odNames []string
				for _, od := range odList {
					odNames = append(odNames, od.Name)
				}
				status = http.StatusConflict
				return fmt.Errorf("cannot delete user - remove or transfer ownership of distros in list first: %v", odNames)
			}
		}

		// *** All good! let's start deleting stuff ***

		// remove owned profiles
		clog.Debug().Msgf("finding '%s' owned profiles", username)
		if opList, opErr := dbReadProfiles(searchByOwnerID, tx); opErr != nil {
			return opErr // uses default err status
		} else {
			for _, p := range opList {
				clog.Debug().Msgf("deleting profile '%s'", p.Name)
				if err = dbDeleteProfile(&p, tx); err != nil {
					return err // uses default err status
				}
			}
		}

		// get user PUG
		pug, pugErr := user.getPug()
		if pugErr != nil {
			return pugErr // uses default err status
		}

		// delete user PUG permissions
		clog.Debug().Msgf("deleting '%s' group permissions", pug.Name)
		if err = dbDeletePermissionsByGroup(pug, tx); err != nil {
			return err // uses default err status
		}

		// delete user PUG
		clog.Debug().Msgf("deleting '%s' group", pug.Name)
		if err = dbDeleteGroup(pug, tx); err != nil {
			return err // uses default err status
		}

		// delete the user (and their group memberships)
		clog.Debug().Msgf("deleting '%s' from the database and removing group memberships", username)
		return dbDeleteUser(user, tx)

	}); err == nil {
		clog.Debug().Msgf("user '%s' deletion complete", username)
		status = http.StatusOK
	}
	return
}

func denySysUserDelete(u string) error {
	if u == IgorAdmin {
		return fmt.Errorf("user '%s' %s", u, SysForbidDelete)
	}
	return nil
}
