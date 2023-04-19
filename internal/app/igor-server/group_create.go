// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"

	"gorm.io/gorm"
)

func doCreateGroup(groupParams map[string]interface{}, r *http.Request) (group *Group, status int, err error) {

	groupName := groupParams["name"].(string)
	owner := getUserFromContext(r)
	status = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		exists, geErr := groupExists(groupName, tx)
		if geErr != nil {
			return geErr // uses default err status
		}
		if exists {
			status = http.StatusConflict
			return fmt.Errorf("group '%s' already exists", groupName)
		}

		group = &Group{
			Name:  groupName,
			Owner: *owner,
		}

		var members []string
		members = append(members, owner.Name)
		if requestedMembers, ok := groupParams["members"].([]interface{}); ok {
			for _, v := range requestedMembers {
				member := v.(string)
				if member == owner.Name {
					continue // skip because included above
				}
				members = append(members, member)
			}
		}

		if uList, guStatus, guErr := getUsers(members, true, tx); guErr != nil {
			status = guStatus
			return guErr
		} else {
			group.Members = uList
		}

		// set description, if included
		if desc, ok := groupParams["description"]; ok {
			group.Description = desc.(string)
		}

		return dbCreateGroup(group, false, tx) // uses default err status

	}); err == nil {
		status = http.StatusCreated

		// only send this email if the group has members other than the owner
		if len(group.Members) > 1 {

			groupCreatedMsg := makeGroupNotifyEvent(EmailGroupCreated, group, nil, "")
			if groupCreatedMsg != nil {
				groupNotifyChan <- *groupCreatedMsg
			}
		}
	}

	return
}
