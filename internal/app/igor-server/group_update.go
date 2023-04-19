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

// doUpdateGroup steps through the process of making an update to a group.
//
// Returns:
//
//	200,nil if update was successful
//	400,error if the request doesn't make sense from an operational standpoint
//	403,error if the request would destabilize the application
//	404,error if group, new owner or member cannot be found
//	409,error if attempting to rename the group and that name is already in use
//	500,error if an internal error occurred
func doUpdateGroup(groupName string, editParams map[string]interface{}, r *http.Request) (status int, err error) {

	// validate changes that don't require DB lookups
	clog := hlog.FromRequest(r)
	var groupId int

	_, hasName := editParams["name"].(string)
	if hasName {
		if groupName == GroupAll || groupName == GroupAdmins || strings.HasPrefix(groupName, GroupUserPrefix) {
			return http.StatusForbidden, fmt.Errorf("cannot change the name of group '%s'", groupName)
		}
	}

	newOwnerName, hasOwner := editParams["owner"].(string)
	if hasOwner {
		if newOwnerName == IgorAdmin {
			ruser := getUserFromContext(r)
			if !userElevated(ruser.Name) {
				return http.StatusForbidden, fmt.Errorf("must have admin status to assign group ownership to %s", IgorAdmin)
			}
		}
	}

	var addMemNames []string
	add, hasAdd := editParams["add"].([]interface{})
	if hasAdd {
		for _, u := range add {
			newMem := u.(string)
			if newMem == IgorAdmin {
				return http.StatusForbidden, fmt.Errorf("cannot add %s to any group", IgorAdmin)
			}
			addMemNames = append(addMemNames, newMem)
		}
	}

	var rmMemNames []string
	remove, hasRemove := editParams["remove"].([]interface{})
	if hasRemove {
		if groupName == GroupAll {
			return http.StatusForbidden, fmt.Errorf("cannot remove members from the '%s' group", GroupAll)
		}
		for _, u := range remove {
			rmName := u.(string)
			if rmName == IgorAdmin && groupName == GroupAdmins {
				return http.StatusForbidden, fmt.Errorf("cannot remove %s from the '%s' group", IgorAdmin, GroupAdmins)
			}
			if rmName == newOwnerName {
				return http.StatusBadRequest, fmt.Errorf("cannot assign a new owner who is also removed from the group")
			}
			for _, adName := range addMemNames {
				if rmName == adName {
					return http.StatusBadRequest, fmt.Errorf("the same user appears in both add and remove oldOwner params")
				}
			}
			rmMemNames = append(rmMemNames, rmName)
		}
	}

	status = http.StatusInternalServerError // default status, overridden at end if no errors

	var addUsers, removeUsers []User
	var group *Group
	var oldOwner *User
	var newName, oldName string

	if err = performDbTx(func(tx *gorm.DB) error {

		changes := map[string]interface{}{}

		if hasName {
			name := editParams["name"].(string)
			if found, findErr := groupExists(name, tx); findErr != nil {
				return findErr // uses default err status
			} else if found {
				status = http.StatusConflict
				return fmt.Errorf("group name '%s' already in use", name)
			}
			changes["name"] = name
			newName = name
			oldName = groupName
		}

		if desc, hasDesc := editParams["description"]; hasDesc {
			changes["description"] = desc.(string)
		}

		if gList, gStatus, gErr := getGroups([]string{groupName}, true, tx); gErr != nil {
			status = gStatus
			return gErr
		} else {
			group = &gList[0]
			groupId = group.ID
		}

		if hasAdd {
			if nml, guStatus, guErr := getUsers(addMemNames, true, tx); guErr != nil {
				status = guStatus
				return guErr
			} else {
				addUsers = nml
			}
		}

		if hasOwner {
			nown, guStatus, guErr := getUsers([]string{newOwnerName}, true, tx)
			if err != nil {
				status = guStatus
				return guErr
			}
			changes["owner"] = nown[0]
			temp := group.Owner
			oldOwner = &temp

			// We will add the new owner in case they didn't already belong to the group
			addUsers = append(addUsers, nown...)

			ownerPermList, gpmErr := dbGetResourceOwnerPermissions(PermGroups, group.Name, &group.Owner, tx)
			if gpmErr != nil {
				return gpmErr // uses default err status
			}
			changes["owner-permissions"] = ownerPermList

			ownerPugID, gpErr := nown[0].getPugID()
			if gpErr != nil {
				return gpErr // uses default err status
			}
			changes["newowner-groupId"] = ownerPugID
		}

		if hasRemove {
			if rml, guStatus, guErr := getUsers(rmMemNames, true, tx); guErr != nil {
				status = guStatus
				return guErr
			} else {
				removeUsers = rml
			}

			// If we are removing the group's current owner but not naming a new one... denied!
			if userSliceContains(removeUsers, group.Owner.Name) && newOwnerName == "" {
				status = http.StatusBadRequest
				return fmt.Errorf("cannot remove the owner of the group without designating a new one")
			}

			// find out the distros that are accessible by this group and if the current owner of the distro is
			// being removed from the group then it will have to be removed from the distro's group list
			if dList, dErr := dbReadDistros(map[string]interface{}{"groups": []int{group.ID}}, tx); dErr != nil {
				return dErr // uses default err status
			} else if len(dList) > 0 {
				for _, d := range dList {
					for _, rmName := range rmMemNames {
						if d.Owner.Name == rmName {
							err = dbEditDistro(&d, map[string]interface{}{"removeGroup": []Group{*group}}, tx)
							if err != nil {
								return err // uses default err status
							}
						}
					}
				}
			}
		}

		if len(addUsers) > 0 {
			changes["add"] = addUsers
		}
		if len(removeUsers) > 0 {
			changes["remove"] = removeUsers
		}

		return dbEditGroup(group, changes, tx) // uses default err status

	}); err == nil {

		status = http.StatusOK

		var notifyList []*GroupNotifyEvent

		// if the group update was successful and the update included a name change, record this history with any
		// affected reservations. don't stop if the record doesn't update properly
		if hasName {
			rList, _ := dbReadReservationsTx(map[string]interface{}{"group_id": groupId}, nil)
			for _, res := range rList {
				if hErr := res.HistCallback(&res, HrUpdated+":group-rename"); hErr != nil {
					clog.Error().Msgf("failed to record reservation '%s' group rename to history", res.Name)
				} else {
					clog.Debug().Msgf("group renamed - recorded historical change to reservation '%s'", res.Name)
				}
			}

			gList, _ := dbReadGroupsTx(map[string]interface{}{"name": newName, "showMembers": true}, true)
			group = &gList[0]
			if grpEvent := makeGroupNotifyEvent(EmailGroupChangeName, group, nil, oldName); grpEvent != nil {
				notifyList = append(notifyList, grpEvent)
			}
		} else {
			gList, _ := dbReadGroupsTx(map[string]interface{}{"name": groupName, "showMembers": true}, true)
			group = &gList[0]
		}

		if oldOwner != nil {
			if grpEvent := makeGroupNotifyEvent(EmailGroupChangeOwn, group, oldOwner, oldName); grpEvent != nil {
				notifyList = append(notifyList, grpEvent)
			}
		}

		if len(addUsers) > 0 {
			for _, u := range addUsers {
				if grpEvent := makeGroupNotifyEvent(EmailGroupAddMem, group, &u, oldName); grpEvent != nil {
					notifyList = append(notifyList, grpEvent)
				}
			}
		}
		if len(removeUsers) > 0 {
			for _, u := range removeUsers {
				if grpEvent := makeGroupNotifyEvent(EmailGroupRmvMem, group, &u, oldName); grpEvent != nil {
					notifyList = append(notifyList, grpEvent)
				}
			}
		}

		if len(notifyList) > 0 {
			for _, m := range notifyList {
				groupNotifyChan <- *m
			}
		}
	}

	return
}
