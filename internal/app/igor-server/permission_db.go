// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"sort"

	"gorm.io/gorm"
)

// dbGetPermissions returns a list of permissions that match the params in the given map.
func dbGetPermissions(query map[string]interface{}, tx *gorm.DB) (perms []Permission, err error) {
	if result := tx.Where(query).Find(&perms); result.Error != nil {
		return nil, result.Error
	}
	return
}

// dbGetPermissionsByName returns a list of permissions of the given resource type and
// unique resource name. For example:
//
//	dbGetPermissionsByName(PermDistros, "foo", tx)
//
// would return all permissions for the distro named "foo".
func dbGetPermissionsByName(resourceType string, resourceName string, tx *gorm.DB) (perms []Permission, err error) {
	searchParam := resourceType + PermDividerToken + resourceName + PermDividerToken + "%"
	if result := tx.Where("fact LIKE ?", searchParam).Find(&perms); result.Error != nil {
		return nil, result.Error
	}
	return
}

// dbDeletePermissionsByName deletes all permissions where resource is the type and
// name is the unique reference. For example:
//
//	dbDeletePermissionsByName(PermDistros, "foo", tx)
//
// would delete all permissions for the distro named "foo".
//
// It is useful for complete removal of a resource from igor.
func dbDeletePermissionsByName(resourceType string, resourceName string, tx *gorm.DB) (err error) {
	searchParam := resourceType + PermDividerToken + resourceName + PermDividerToken + "%"
	result := tx.Where("fact LIKE ?", searchParam).Delete(&Permission{})
	return result.Error
}

// dbGetPermissionsByGroupTx returns a list of all permissions assigned to the given Group.
func dbGetPermissionsByGroupTx(groups []Group) (perms []Permission, err error) {
	err = performDbTx(func(tx *gorm.DB) error {
		return tx.Model(groups).Association("Permissions").Find(&perms)
	})
	return
}

// dbGetResourceOwnerPermissions returns a list of permissions for the named resource owned by the given user.
func dbGetResourceOwnerPermissions(resourceType string, resourceName string, owner *User, tx *gorm.DB) (perms []Permission, err error) {

	if ownerPrivateGroup, pugErr := owner.getPug(); pugErr != nil {
		return nil, pugErr
	} else {
		return dbGetResourceGroupPermissions(resourceType, resourceName, ownerPrivateGroup, tx)
	}
}

// dbGetResourceGroupPermissions returns a list of permissions for the named resource associated with the given group.
func dbGetResourceGroupPermissions(resourceType string, resourceName string, group *Group, tx *gorm.DB) (perms []Permission, err error) {

	permFind := resourceType + PermDividerToken + resourceName + PermDividerToken + "%"
	err = tx.Model(group).Where("fact LIKE ?", permFind).Association("Permissions").Find(&perms)
	return
}

// dbGetHostPowerPermissions retrieves any power permission that matches input group and list of hosts
func dbGetHostPowerPermissions(group *Group, resHosts []Host, tx *gorm.DB) (perms []Permission, err error) {

	sort.Slice(resHosts, func(i, j int) bool {
		return resHosts[i].Name < resHosts[j].Name
	})

	var hList string
	for i := 0; i < len(resHosts)-1; i++ {
		hList += resHosts[i].Name + PermSubpartToken
	}
	hList += resHosts[len(resHosts)-1].Name

	permFind := PermPowerAction + PermDividerToken + hList + "%"
	err = tx.Model(group).Where("fact LIKE ?", permFind).Association("Permissions").Find(&perms)
	return
}

// dbAppendPermissions performs the steps to add list of Permission to the given Group in the database.
func dbAppendPermissions(group *Group, perms []Permission, tx *gorm.DB) error {
	return tx.Model(&group).Association("Permissions").Append(perms)
}

// dbDeletePermissionsByGroup deletes all permissions associated with the provided Group. That is,
// given Group g, delete every permission granted to g.
func dbDeletePermissionsByGroup(group *Group, tx *gorm.DB) error {
	if group.Name == GroupAll {
		return fmt.Errorf("delete all permissions of group '%s' not allowed", GroupAll)
	}
	result := tx.Where("group_id = ?", group.ID).Delete(&Permission{})
	return result.Error
}
