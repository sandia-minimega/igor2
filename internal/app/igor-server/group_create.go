// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"

	"gorm.io/gorm"
)

func doCreateGroup(groupParams map[string]interface{}, r *http.Request) (group *Group, status int, extraMsg string, err error) {

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

		// need to add additional owners if present and also mark whether the group is LDAP-synced.

		group = &Group{
			Name:   groupName,
			IsLDAP: false,
		}

		if isLdap, ok := groupParams["isLDAP"].(bool); ok && isLdap {
			group.IsLDAP = true
			group.Description = "( LDAP-sync group )"

			if members, owners, ldapErr := executeLdapGroupCreate(group); ldapErr != nil {
				// status is default
				return ldapErr
			} else if len(owners) == 1 && userSliceContains(owners, IgorAdmin) && userElevated(owner.Name) {
				extraMsg = "LDAP listed owner is not an igor user so sync-group is owned by igor-admin"
			} else if !(userSliceContains(owners, owner.Name) || userElevated(owner.Name)) {
				status = http.StatusForbidden
				return fmt.Errorf("you do not have persmission to add the LDAP group '%s' to igor - must be an owner/delegate of the group or an igor admin", groupName)
			} else {
				group.Members = members
				group.Owners = owners
			}

		} else {

			// non-LDAP group add
			var owners, members []string
			owners = append(owners, owner.Name)
			if requestedOwners, ok := groupParams["owners"].([]interface{}); ok {
				for _, v := range requestedOwners {
					coOwner := v.(string)
					if coOwner == owner.Name {
						continue // skip because included above
					}
					owners = append(owners, coOwner)
				}
			}

			if uList, guStatus, guErr := getUsers(owners, true, tx); guErr != nil {
				status = guStatus
				return guErr
			} else {
				group.Owners = uList
				group.Members = append(group.Members, uList...)
			}

			if requestedMembers, ok := groupParams["members"].([]interface{}); ok {
				for _, m := range requestedMembers {
					member := m.(string)
					for _, o := range owners {
						if member == o {
							continue // skip because included above
						}
					}
					members = append(members, member)
				}
			}

			if len(members) > 0 {
				if uList, guStatus, guErr := getUsers(members, true, tx); guErr != nil {
					status = guStatus
					return guErr
				} else {
					group.Members = append(group.Members, uList...)
				}
			}

			// set description, if included
			if desc, ok := groupParams["description"]; ok {
				group.Description = desc.(string)
			}
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
