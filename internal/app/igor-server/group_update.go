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
	var group *Group

	if err = performDbTx(func(tx *gorm.DB) error {

		if gList, gStatus, gErr := getGroups([]string{groupName}, true, tx); gErr != nil {
			status = gStatus
			return gErr
		} else {
			group = &gList[0]
			groupId = group.ID
			if group.IsLDAP {
				clog.Warn().Msgf("user issued a group update command on an LDAP-synced group.")
				status = http.StatusForbidden
				return fmt.Errorf("cannot change details of LDAP-synced group '%s' within igor", groupName)
			}
		}
		return nil
	}); err != nil {
		return
	}

	_, hasName := editParams["name"].(string)
	if hasName {
		if groupName == GroupAll || groupName == GroupAdmins || strings.HasPrefix(groupName, GroupUserPrefix) {
			return http.StatusForbidden, fmt.Errorf("cannot change the name of group '%s'", groupName)
		}
	}

	var addOwnerNames []string
	addOwners, hasOwners := editParams["addOwners"].([]interface{})
	if hasOwners {
		for _, u := range addOwners {
			newOwn := u.(string)
			if newOwn == IgorAdmin {
				return http.StatusForbidden, fmt.Errorf("cannot add %s to any group", IgorAdmin)
			}
			addOwnerNames = append(addOwnerNames, newOwn)
		}
	}

	var rmvOwnerNames []string
	rmvOwners, rmvOwner := editParams["rmvOwners"].([]interface{})
	if rmvOwner {
		for _, u := range rmvOwners {
			rmvOwn := u.(string)
			if rmvOwn == IgorAdmin && groupName == GroupAdmins {
				return http.StatusForbidden, fmt.Errorf("cannot remove %s from the '%s' group", IgorAdmin, GroupAdmins)
			}
			rmvOwnerNames = append(rmvOwnerNames, rmvOwn)
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
			for _, oName := range addOwnerNames {
				if oName == rmName {
					return http.StatusBadRequest, fmt.Errorf("cannot assign a new owner who is also removed from the group")
				}
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
	var addNewOwners, rmvOldOwners []User
	//var newOwner *User
	//var oldOwner *User
	var newGroupName, oldGroupName string

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
			newGroupName = name
			oldGroupName = groupName
		}

		if desc, hasDesc := editParams["description"]; hasDesc {
			changes["description"] = desc.(string)
		}

		if hasAdd {
			if nml, guStatus, guErr := getUsers(addMemNames, true, tx); guErr != nil {
				status = guStatus
				return guErr
			} else {
				addUsers = nml
			}
		}

		if hasOwners {
			addNewOwners, guStatus, guErr := getUsers(addOwnerNames, true, tx)
			if err != nil {
				status = guStatus
				return guErr
			}
			changes["addOwners"] = addNewOwners

			// We will add the new owner in case they didn't already belong to the group
			addUsers = append(addUsers, addNewOwners...)
		}

		if rmvOwner {
			rmvOldOwners, guStatus, guErr := getUsers(rmvOwnerNames, true, tx)
			if err != nil {
				status = guStatus
				return guErr
			}

			if hasOwners {
				for _, o := range addOwnerNames {
					if userSliceContains(rmvOldOwners, o) {
						status = http.StatusBadRequest
						return fmt.Errorf("cannot add and remove the same owner '%s' from group", o)
					}
				}
			} else if len(group.Owners)+len(addOwnerNames) <= len(rmvOldOwners) {
				status = http.StatusBadRequest
				return fmt.Errorf("cannot remove all owners from a group")
			}

			changes["rmvOwners"] = rmvOldOwners
		}

		if hasRemove {
			if rml, guStatus, guErr := getUsers(rmMemNames, true, tx); guErr != nil {
				status = guStatus
				return guErr
			} else {
				removeUsers = rml
			}

			var ownersRemoved = 0
			for _, o := range group.Owners {
				for _, u := range removeUsers {
					if o.Name == u.Name {
						ownersRemoved++
						rmvOldOwners = append(rmvOldOwners, u)
					}
				}
			}

			if len(group.Owners) <= ownersRemoved && len(addOwnerNames) == 0 {
				status = http.StatusBadRequest
				return fmt.Errorf("cannot remove the last owner of a group without designating a new one")
			} else if len(rmvOldOwners) > 0 {
				changes["rmvOwners"] = rmvOldOwners
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

		// only send these notifications if the group is NOT an LDAP-synced group.

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

			gList, _ := dbReadGroupsTx(map[string]interface{}{"name": newGroupName, "showMembers": true}, true)
			group = &gList[0]
			if grpEvent := makeGroupNotifyEvent(EmailGroupChangeName, group, nil, oldGroupName); grpEvent != nil {
				notifyList = append(notifyList, grpEvent)
			}
		} else {
			gList, _ := dbReadGroupsTx(map[string]interface{}{"name": groupName, "showMembers": true}, true)
			group = &gList[0]
		}

		if len(addOwnerNames) > 0 {
			for _, o := range addNewOwners {
				if grpEvent := makeGroupNotifyEvent(EmailGroupAddOwner, group, &o, o.Name); grpEvent != nil {
					notifyList = append(notifyList, grpEvent)
				}
			}
		}

		if len(rmvOldOwners) > 0 {
			for _, o := range rmvOldOwners {
				if grpEvent := makeGroupNotifyEvent(EmailGroupRmvOwner, group, &o, o.Name); grpEvent != nil {
					notifyList = append(notifyList, grpEvent)
				}
			}
		}

		if len(addUsers) > 0 {
			for _, u := range addUsers {
				if grpEvent := makeGroupNotifyEvent(EmailGroupAddMem, group, &u, oldGroupName); grpEvent != nil {
					notifyList = append(notifyList, grpEvent)
				}
			}
		}
		if len(removeUsers) > 0 {
			for _, u := range removeUsers {
				if grpEvent := makeGroupNotifyEvent(EmailGroupRmvMem, group, &u, oldGroupName); grpEvent != nil {
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
