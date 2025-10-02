// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"strings"

	"gorm.io/gorm"
)

// dbCreateDistro is the create operation for creating a new distro.
func dbCreateDistro(distro *Distro, tx *gorm.DB) error {

	// set owner permissions
	operms, err := createDistroOwnerPerms(distro.Name)
	if err != nil {
		return err
	}
	pug, err := distro.Owner.getPug()
	if err != nil {
		return err
	}
	if err = dbAppendPermissions(pug, operms, tx); err != nil {
		return err
	}

	// set group permissions
	for _, group := range distro.Groups {
		gperms, _ := createDistroGroupPerms(distro.Name)
		if err = dbAppendPermissions(&group, gperms, tx); err != nil {
			return err
		}
	}
	// create distro
	result := tx.Create(&distro)
	return result.Error

}

// dbReadDistrosTx performs dbReadDistros in a new transaction.
func dbReadDistrosTx(queryParams map[string]interface{}) (distroList []Distro, err error) {

	err = performDbTx(func(tx *gorm.DB) error {
		distroList, err = dbReadDistros(queryParams, tx)
		return err
	})

	return distroList, err
}

// dbReadDistros returns a list of distros matching the given queryParams, possibly returning none.
// If no queryParams are provided, all distros are returned.
func dbReadDistros(queryParams map[string]interface{}, tx *gorm.DB) (distroList []Distro, err error) {

	tx = tx.Preload("DistroImage").Preload("Owner").Preload("Groups").Preload("Owner.Groups").Preload("Kickstart")

	// if no params given, return all distros
	if len(queryParams) == 0 {
		result := tx.Find(&distroList)
		return distroList, result.Error
	}

	for key, val := range queryParams {
		switch val.(type) {
		case string, int, bool:
			tx = tx.Where(key, val)
		case []int:
			if strings.ToLower(key) == "groups" {
				tx = tx.Joins("JOIN distros_groups ON distros_groups.distro_id = ID AND group_id IN ?", val)
			} else {
				tx = tx.Where(key+" IN ?", val)
			}
		case []string:
			tx = tx.Where(key+" IN ?", val)
		default:
			// we shouldn't reach this error because we already checked the param types
			logger.Error().Msgf("dbReadDistros: incorrect parameter type %T received for %s: %v", val, key, val)
		}
	}

	result := tx.Group("Name").Find(&distroList)
	return distroList, result.Error
}

// dbEditDistro updates the target user in the Distro database table with information from
// the changes map.
func dbEditDistro(distro *Distro, changes map[string]interface{}, tx *gorm.DB) error {

	// tx = tx.Preload("DistroImage").Preload("Owner").Preload("Groups").Preload("Owner.Groups").Preload("Kickstart")

	// first find and deal with ownership/group changes, if any
	modifyOwner := false
	modifyGroups := false
	targetOwner := &distro.Owner
	originalOwner := distro.Owner
	var removeGroups []Group
	var addGroups []Group

	// Change the name of the distro
	if name, ok := changes["Name"].(string); ok {
		if perms, pResultErr := dbGetPermissionsByName(PermDistros, distro.Name, tx); pResultErr != nil {
			return pResultErr
		} else {
			oldName := PermDividerToken + distro.Name + PermDividerToken
			newName := PermDividerToken + name + PermDividerToken
			for _, p := range perms {
				newFact := strings.Replace(p.Fact, oldName, newName, 1)
				if result := tx.Model(&p).Update("Fact", newFact); result.Error != nil {
					return result.Error
				}
			}
			if result := tx.Model(&distro).Update("Name", name); result.Error != nil {
				return result.Error
			}
			delete(changes, "Name")
		}
	}
	if rGroups, ok := changes["removeGroup"]; ok {
		modifyGroups = true
		removeGroups = rGroups.([]Group)
		delete(changes, "removeGroup")
	}
	if aGroups, ok := changes["addGroup"]; ok {
		modifyGroups = true
		addGroups = aGroups.([]Group)
		delete(changes, "addGroup")
	}
	if newOwner, ok := changes["owner"].(*User); ok {
		targetOwner = newOwner
		modifyOwner = true
		delete(changes, "owner")
	}
	if _, ok := changes["isPublic"].(bool); ok {
		modifyGroups = true
		modifyOwner = true
		admin, _, err := getIgorAdmin(tx)
		if err != nil {
			return err
		}
		targetOwner = admin
		removeGroups = distro.Groups
		// remove old user's pug from this list, we'll deal with it separately
		pug, err := distro.Owner.getPug()
		if err != nil {
			return err
		}
		removeGroups = removeGroup(removeGroups, pug)
		if allGroup, _, err := getAllGroup(tx); err != nil {
			return err
		} else {
			addGroups = []Group{*allGroup}
		}
		delete(changes, "isPublic")
	}
	// if we're changing owner and/or groups, handle permission changes
	// and set new owner/groups to the distro object
	if modifyGroups || modifyOwner {
		// if the owner is changing, first remove their distro permissions
		if modifyOwner {
			poChanges, err := dbGetResourceOwnerPermissions(PermDistros, distro.Name, &originalOwner, tx)
			if err != nil {
				return err
			}
			// update owner permissions to new owner ID
			newOwnerPug, err := targetOwner.getPug()
			if err != nil {
				return err
			}
			if result := tx.Model(poChanges).Update("GroupID", newOwnerPug.ID); result.Error != nil {
				return result.Error
			}
			// save new owner to distro
			distro.Owner = *targetOwner
			tx.Save(&distro)
		}
		if len(removeGroups) > 0 {
			for _, group := range removeGroups {
				// remove group's permissions
				pgChanges, err := dbGetResourceGroupPermissions(PermDistros, distro.Name, &group, tx)
				if err != nil {
					return err
				}
				if result := tx.Delete(pgChanges); result.Error != nil {
					return result.Error
				}
				// remove group from distro
				if err = tx.Model(&distro).Association("Groups").Delete(group); err != nil {
					return err
				}
			}
		}
		if len(addGroups) > 0 {
			for _, group := range addGroups {
				// add new group to distro groups
				distro.Groups = append(distro.Groups, group)
				// add distro permissions for new group
				gPerms, err := createDistroGroupPerms(distro.Name)
				if err != nil {
					return err
				}
				pErr := dbAppendPermissions(&group, gPerms, tx)
				if pErr != nil {
					return pErr
				}
			}
		}
		// save any owner and/or group changes made
		if result := tx.Save(&distro); result.Error != nil {
			return result.Error
		}
	}

	if modifyOwner {
		// swap owner pugs in the distro if changing owners but not public
		// owner permissions changes were already handled separately above
		oldOwnerPug, err := originalOwner.getPug()
		if err != nil {
			return err
		}
		distro.Groups = removeGroup(distro.Groups, oldOwnerPug)
		if err = tx.Model(&distro).Association("Groups").Delete(oldOwnerPug); err != nil {
			return err
		}

		// add the new owner's pug to the group if owner isn't igor-admin
		if distro.Owner.Name != IgorAdmin {
			newOwnerPug, err := distro.Owner.getPug()
			if err != nil {
				return err
			}
			distro.Groups = append(distro.Groups, *newOwnerPug)
		}
		tx.Save(&distro)
	}

	if _, ok := changes["autoRemoveOwner"].(bool); ok {
		oldOwnerPug, err := originalOwner.getPug()
		if err != nil {
			return err
		}
		distro.Groups = removeGroup(distro.Groups, oldOwnerPug)
		if err = tx.Model(&distro).Association("Groups").Delete(oldOwnerPug); err != nil {
			return err
		}

		adminPug := changes["adminPug"].(*Group)
		distro.OwnerID = adminPug.ID
		if result := tx.Model(&distro).Update("OwnerID", adminPug.ID); result.Error != nil {
			return result.Error
		}
		return nil
	}

	if newKs, ok := changes["kickstart"].(Kickstart); ok {
		distro.Kickstart = newKs
		distro.KickstartID = newKs.ID
		delete(changes, "kickstart")
	}

	if newInitrdInfo, ok := changes["initrdInfo"].(string); ok {
		if result := tx.Model(&distro.DistroImage).Update("InitrdInfo", newInitrdInfo); result.Error != nil {
			return result.Error
		}
		delete(changes, "initrdInfo")
	}

	// update distro with any remaining changes
	if result := tx.Model(&distro).Updates(changes); result.Error != nil {
		return result.Error
	}
	return nil

}

// dbDeleteDistro deletes a distro from the Distro database table
func dbDeleteDistro(distro *Distro, tx *gorm.DB) error {

	// delete the distro's permissions
	err := dbDeletePermissionsByName(PermDistros, distro.Name, tx)
	if err != nil {
		return err
	}

	// clear out references to the distro in the distros_groups join table
	if err := tx.Model(&distro).Association("Groups").Clear(); err != nil {
		return err
	}

	// delete the distro
	result := tx.Delete(&distro)
	return result.Error
}

func createDistroGroupPerms(distroName string) ([]Permission, error) {
	pstr := NewPermissionString(PermDistros, distroName, PermViewAction)
	distroView, err := NewPermission(pstr)
	if err != nil {
		return nil, err
	}
	return []Permission{*distroView}, nil
}

func createDistroOwnerPerms(distroName string) ([]Permission, error) {
	pstr := NewPermissionString(PermDistros, distroName, PermEditAction, PermWildcardToken)
	ownerDistroEdit, err := NewPermission(pstr)
	if err != nil {
		return nil, err
	}
	pstr = NewPermissionString(PermDistros, distroName, PermDeleteAction+PermSubpartToken+PermViewAction)
	ownerDistroDelView, err := NewPermission(pstr)
	if err != nil {
		return nil, err
	}
	return []Permission{*ownerDistroEdit, *ownerDistroDelView}, nil
}
