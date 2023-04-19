// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// dbCreateProfile creates a new profile along with its permissions.
func dbCreateProfile(profile *Profile, tx *gorm.DB) error {
	oPerms, err := createProfileOwnerPerms(profile.Name)
	if err != nil {
		return err
	}
	pug, pugErr := profile.Owner.getPug()
	if pugErr != nil {
		return pugErr
	}
	if err = dbAppendPermissions(pug, oPerms, tx); err != nil {
		return err
	}
	result := tx.Create(&profile)
	return result.Error
}

// dbReadProfilesTx performs dbReadProfiles in a new transaction.
func dbReadProfilesTx(queryParams map[string]interface{}) (profileList []Profile, err error) {

	err = performDbTx(func(tx *gorm.DB) error {
		profileList, err = dbReadProfiles(queryParams, tx)
		return err
	})

	return profileList, err
}

// dbReadProfiles returns a list of profiles matching the given queryParams. If no queryParams are
// specified then all profiles are returned.
func dbReadProfiles(queryParams map[string]interface{}, tx *gorm.DB) (profileList []Profile, err error) {

	tx = tx.Preload("Owner").Preload("Distro").Preload("Owner.Groups").Preload("Distro.Groups").Preload("Distro.Kickstart")

	// if no params given, return all reservations
	if len(queryParams) == 0 {
		result := tx.Find(&profileList)
		return profileList, result.Error
	}

	for key, val := range queryParams {
		switch val.(type) {
		case string, int:
			tx = tx.Where(key, val)
		case []string, []int:
			tx = tx.Where(key+" IN ?", val)
		default:
			logger.Error().Msgf("dbReadProfiles: incorrect parameter type %T received for %s: %v", val, key, val)
		}
	}

	result := tx.Find(&profileList)
	return profileList, result.Error
}

func dbEditProfile(p *Profile, changes map[string]interface{}, tx *gorm.DB) error {
	// Ideally, target has already been found in the db and
	// changes have already been screened by the handler

	// Change the name of the distro
	if name, ok := changes["Name"].(string); ok {
		if perms, pResultErr := dbGetPermissionsByName(PermProfiles, p.Name, tx); pResultErr != nil {
			return pResultErr
		} else {
			oldName := PermDividerToken + p.Name + PermDividerToken
			newName := PermDividerToken + name + PermDividerToken
			for _, perm := range perms {
				newFact := strings.Replace(perm.Fact, oldName, newName, 1)
				if result := tx.Model(&perm).Update("Fact", newFact); result.Error != nil {
					return result.Error
				}
			}
			if result := tx.Model(&p).Update("Name", name); result.Error != nil {
				return result.Error
			}
			delete(changes, "Name")
		}
	}

	result := tx.Model(&p).Updates(changes)
	return result.Error
}

func dbDeleteProfile(profile *Profile, tx *gorm.DB) error {
	perms, err := dbGetResourceOwnerPermissions(PermProfiles, profile.Name, &profile.Owner, tx)
	if err != nil {
		return err
	}
	if len(perms) > 0 {
		if result := tx.Delete(perms); result.Error != nil {
			return result.Error
		}
	} else {
		return fmt.Errorf("no permissions found for profile %v and owner %v", profile.Name, profile.Owner.Name)
	}

	result := tx.Delete(&profile)
	return result.Error
}

// func createProfileGroupPerms(profileName string) ([]Permission, error) {
// 	pstr := NewPermissionString(PermProfiles, profileName, PermViewAction)
// 	profileView, err := NewPermission(pstr)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return []Permission{profileView}, nil
// }

func createProfileOwnerPerms(profileName string) ([]Permission, error) {
	pstr := NewPermissionString(PermProfiles, profileName, PermEditAction, PermWildcardToken)
	ownerProfileEdit, err := NewPermission(pstr)
	if err != nil {
		return nil, err
	}
	pstr = NewPermissionString(PermProfiles, profileName, PermDeleteAction)
	ownerProfileDel, err := NewPermission(pstr)
	if err != nil {
		return nil, err
	}
	pstr = NewPermissionString(PermProfiles, profileName, PermViewAction)
	profileView, err := NewPermission(pstr)
	if err != nil {
		return nil, err
	}
	return []Permission{*ownerProfileEdit, *ownerProfileDel, *profileView}, nil
}
