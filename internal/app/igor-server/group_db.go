// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// dbCreateGroup creates a new group. If the group is not a pug it gathers additional
// information and creates the ownership permission as well. Pug permissions are handled
// separately in doCreateUser.
func dbCreateGroup(group *Group, isPug bool, tx *gorm.DB) (err error) {

	result := tx.Create(&group)
	if result.Error != nil {
		return result.Error
	}

	// change permissions here to get pug of each owner?

	if !isPug {
		//var pugs = make([]*Group, 0, len(group.Owner))
		for _, owner := range group.Owners {
			pug, gpErr := owner.getPug()
			if gpErr != nil {
				return gpErr
			}
			gPerm, cgopErr := createGroupOwnerPerms(group)
			if cgopErr != nil {
				return cgopErr
			}
			err = dbAppendPermissions(pug, gPerm, tx)
			if err != nil {
				return
			}
		}
		vPerm, _ := NewPermission(NewPermissionString(PermGroups, group.Name, PermViewAction))
		err = dbAppendPermissions(group, []Permission{*vPerm}, tx)
		if err != nil {
			return
		}
	}
	return
}

// dbReadGroupsTx performs dbReadGroups in a new transaction.
func dbReadGroupsTx(queryParams map[string]interface{}, excludePugs bool) (groupList []Group, err error) {

	err = performDbTx(func(tx *gorm.DB) error {
		groupList, err = dbReadGroups(queryParams, excludePugs, tx)
		return err
	})

	return groupList, err
}

// dbReadGroups returns a list of groups matching give queryParams, possibly matching none. If no queryParams are
// present then all groups are returned.  Set excludePugs to true if you don't want to include user-private groups
// in the results.
func dbReadGroups(queryParams map[string]interface{}, excludePugs bool, tx *gorm.DB) ([]Group, error) {

	var groups []Group

	tx = tx.Preload("Owners").Preload("Owners.Groups").Preload("Policies").
		Preload("Distros").Preload("Reservations").Order("name COLLATE NOCASE ASC")

	if excludePugs {
		tx = tx.Where("is_user_private = ?", false)
	}

	// if no params given, return all groups
	if len(queryParams) == 0 {
		result := tx.Find(&groups)
		return groups, result.Error
	}

	for key, val := range queryParams {
		switch val.(type) {
		case bool:
			if key == "showMembers" {
				tx = tx.Preload("Members")
			} else {
				tx = tx.Where(key, val)
			}
		case string:
			tx = tx.Where(key, val)
		case []int:
			switch strings.ToLower(key) {
			case "policies":
				tx = tx.Joins("JOIN groups_policies ON groups_policies.group_id = ID AND host_policy_id IN ?", val)
			case "distros":
				tx = tx.Joins("JOIN distros_groups ON distros_groups.group_id = ID AND distro_id IN ?", val)
			case "owners":
				tx = tx.Joins("JOIN groups_owners ON groups_owners.group_id = ID AND user_id IN ?", val)
			case "users":
				tx = tx.Joins("JOIN groups_users ON groups_users.group_id = ID AND user_id IN ?", val)
			default:
				tx = tx.Where(key+" IN ?", val)
			}
		case []string:
			tx = tx.Where(key+" IN ?", val)
		default:
			// we shouldn't reach this error because we already checked the param types
			logger.Error().Msgf("dbReadGroups: incorrect parameter type %T received for %s: %v", val, key, val)
		}
	}

	result := tx.Distinct().Find(&groups)

	return groups, result.Error
}

// dbEditGroup edits the properties of a Group.
func dbEditGroup(group *Group, changes map[string]interface{}, tx *gorm.DB) error {

	if _, ok := changes["ldapRemoveOwner"].(bool); ok {
		delete(changes, "ldapRemoveOwner")
		admin, _ := changes["Admin"].([]User)
		owner, _ := changes["Owner"].([]User)
		if err := tx.Model(&group).Clauses(clause.OnConflict{DoNothing: true}).Association("Owners").Append(admin); err != nil {
			return err
		}
		if err := tx.Model(&group).Association("Owners").Delete(owner); err != nil {
			return err
		}
		return nil
	}

	// Change the name of the group
	if name, ok := changes["name"].(string); ok {
		if perms, pResultErr := dbGetPermissionsByName(PermGroups, group.Name, tx); pResultErr != nil {
			return pResultErr
		} else {
			oldName := PermDividerToken + group.Name + PermDividerToken
			newName := PermDividerToken + name + PermDividerToken
			for _, p := range perms {
				newFact := strings.Replace(p.Fact, oldName, newName, 1)
				if result := tx.Model(&p).Update("Fact", newFact); result.Error != nil {
					return result.Error
				}
			}
			if result := tx.Model(&group).Update("Name", name); result.Error != nil {
				return result.Error
			}
		}
	}

	// Change the description of the group
	if desc, ok := changes["description"].(string); ok {
		if result := tx.Model(&group).Update("Description", desc); result.Error != nil {
			return result.Error
		}
	}

	// Add users to the group (this includes a new owner if they weren't already a member)
	if aUsers, ok := changes["add"].([]User); ok {
		if err := tx.Model(&group).Clauses(clause.OnConflict{DoNothing: true}).Association("Members").Append(aUsers); err != nil {
			return err
		}
	}

	if addOwners, ok := changes["addOwners"].([]User); ok {
		if err := tx.Model(&group).Clauses(clause.OnConflict{DoNothing: true}).Association("Owners").Append(addOwners); err != nil {
			return err
		}
		for _, owner := range addOwners {
			pug, gpErr := owner.getPug()
			if gpErr != nil {
				return gpErr
			}
			gPerm, cgopErr := createGroupOwnerPerms(group)
			if cgopErr != nil {
				return cgopErr
			}
			apErr := dbAppendPermissions(pug, gPerm, tx)
			if apErr != nil {
				return apErr
			}
		}
	}

	if rmvOwners, ok := changes["rmvOwners"].([]User); ok {
		if err := tx.Model(&group).Association("Owners").Delete(rmvOwners); err != nil {
			return err
		}
		for _, owner := range rmvOwners {
			if pChanges, gpErr := dbGetResourceOwnerPermissions(PermGroups, group.Name, &owner, tx); gpErr != nil {
				return gpErr
			} else {
				tx.Delete(pChanges)
			}
		}
	}

	// Remove users from the group
	if rmUsers, ok := changes["remove"].([]User); ok {
		if err := tx.Model(&group).Association("Members").Delete(rmUsers); err != nil {
			return err
		}
	}

	return nil
}

// dbDeleteGroup will delete the group and any extraneous permissions associated with it.
// Note that the perms argument should only be permissions granted to the group owner.
func dbDeleteGroup(group *Group, tx *gorm.DB) error {

	if err := tx.Model(&group).Association("Members").Clear(); err != nil {
		return err
	}

	if err := tx.Model(&group).Association("Owners").Clear(); err != nil {
		return err
	}

	if result := tx.Delete(&group); result.Error != nil {
		return result.Error
	}

	if err := dbDeletePermissionsByName(PermGroups, group.Name, tx); err != nil {
		return err
	}

	return nil
}

func createGroupOwnerPerms(group *Group) (ownerPerms []Permission, err error) {
	if !group.IsLDAP {
		ep := NewPermissionString(PermGroups, group.Name, PermEditAction, PermWildcardToken)
		ownerGroupEdit, err := NewPermission(ep)
		if err != nil {
			return nil, err
		}
		ownerPerms = append(ownerPerms, *ownerGroupEdit)
	}
	dp := NewPermissionString(PermGroups, group.Name, PermDeleteAction)
	ownerGroupDel, err := NewPermission(dp)
	if err != nil {
		return nil, err
	}
	ownerPerms = append(ownerPerms, *ownerGroupDel)
	return
}
