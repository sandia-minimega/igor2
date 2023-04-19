// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"
	"net/http"
	"strings"
)

// doDeleteGroup steps through the process of making an update to a user record.
//
// returns:
//
//	200,nil if delete was successful
//	403,error if tried to delete a system group
//	404,error if user cannot be found
//	409,error if delete was not allowed due to its current state
//	500,error if an internal error occurred
func doDeleteGroup(groupName string, r *http.Request) (status int, err error) {

	clog := hlog.FromRequest(r)

	if err = denySysGroupDelete(groupName); err != nil {
		return http.StatusForbidden, err
	}

	status = http.StatusInternalServerError // default status, overridden at end if no errors

	var rList []Reservation
	var rErr error

	if err = performDbTx(func(tx *gorm.DB) error {

		var group *Group

		if gList, gStatus, gErr := getGroups([]string{groupName}, true, tx); gErr != nil {
			status = gStatus
			return gErr
		} else {
			group = &gList[0]
		}

		// check for things that can stop the group from being deleted

		if hpList, hpErr := dbReadHostPoliciesTx(map[string]interface{}{"access_groups": []int{group.ID}}, clog); hpErr != nil {
			status = http.StatusInternalServerError
			return hpErr
		} else if len(hpList) > 0 {
			status = http.StatusConflict
			return fmt.Errorf("cannot delete '%s' - must edit or remove host policies defining access for this group", group.Name)
		}

		// all set -- let's try to delete the group

		// drop the group from any reservations -- handle like a res update from the client as this will also
		// set permissions properly
		if rList, rErr = dbReadReservations(map[string]interface{}{"group_id": group.ID}, nil, tx); rErr != nil {
			return rErr // uses default err status
		} else if len(rList) > 0 {
			// add param that removes the group from any matching reservations
			editParams := map[string]interface{}{"group": GroupNoneAlias}
			for _, res := range rList {
				if changes, pStatus, prErr := parseResEditParams(&res, editParams, tx); prErr != nil {
					status = pStatus
					return prErr
				} else {
					if erErr := dbEditReservation(&res, changes, tx); erErr != nil {
						status = http.StatusInternalServerError
						return erErr
					}
				}
			}
		}

		// remove from distro group lists
		if dList, dErr := dbReadDistros(map[string]interface{}{"groups": []int{group.ID}}, tx); dErr != nil {
			return dErr // uses default err status
		} else if len(dList) > 0 {
			for _, d := range dList {
				if err = dbEditDistro(&d, map[string]interface{}{"removeGroup": []Group{*group}}, tx); err != nil {
					return err // uses default err status
				}
			}
		}

		return dbDeleteGroup(group, tx) // uses default err status

	}); err == nil {

		status = http.StatusOK

		for _, res := range rList {
			pug, _ := res.Owner.getPug()
			res.Group = *pug
			res.GroupID = pug.ID
			if hErr := res.HistCallback(&res, HrUpdated+":group-delete"); hErr != nil {
				clog.Error().Msgf("failed to record reservation '%s' group change to history", res.Name)
			} else {
				clog.Debug().Msgf("group deleted - recorded historical change to reservation '%s'", res.Name)
			}
		}
	}
	return
}

func denySysGroupDelete(g string) error {
	if g == GroupAdmins || g == GroupAll || g == GroupUserPrefix+IgorAdmin {
		return fmt.Errorf("group '%s' %s", g, SysForbidDelete)
	}
	if strings.HasPrefix(g, GroupUserPrefix) {
		return fmt.Errorf("cannot directly delete private group '%s'; delete user instead", g)
	}
	return nil
}
