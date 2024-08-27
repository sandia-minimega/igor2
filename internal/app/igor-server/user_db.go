// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"gorm.io/gorm"
	"strings"
)

// dbCreateUser creates a new user within an existing transaction.
func dbCreateUser(user *User, tx *gorm.DB) error {
	result := tx.Create(&user)
	return result.Error
}

// dbReadUsers queries the DB for users matching queryParams within a new transaction.
func dbReadUsersTx(queryParams map[string]interface{}) (userList []User, err error) {
	err = performDbTx(func(tx *gorm.DB) error {
		userList, err = dbReadUsers(queryParams, tx)
		return err
	})

	return userList, err
}

// dbReadUsers queries the DB for users matching queryParams within an existing transaction.
func dbReadUsers(queryParams map[string]interface{}, tx *gorm.DB) (userList []User, err error) {

	tx = tx.Preload("Groups").Preload("Groups.Owners")

	// if no params given, return all users
	if len(queryParams) == 0 {
		result := tx.Find(&userList)
		return userList, result.Error
	}

	for key, val := range queryParams {
		switch val.(type) {
		case string, int:
			if strings.ToLower(key) == "exclude-admin" {
				tx = tx.Where("name != ?", IgorAdmin)
			} else {
				tx = tx.Where(key, val)
			}
		case []int:
			if strings.ToLower(key) == "groups" {
				tx = tx.Joins("JOIN groups_users ON groups_users.user_id = ID AND group_id IN ?", val)
			} else {
				tx = tx.Where(key+" IN ?", val)
			}
		case []string:
			tx = tx.Where(key+" IN ?", val)
		default:
			// we shouldn't reach this error because we already checked the param types
			logger.Error().Msgf("dbReadUsers: incorrect parameter type %T received for %s: %v", val, key, val)
		}
	}

	result := tx.Find(&userList)
	return userList, result.Error
}

// dbEditUser updates a user with values included in the changes map within an
// existing transaction.
func dbEditUser(user *User, changes map[string]interface{}, tx *gorm.DB) error {
	result := tx.Model(&user).Select("email", "pass_hash", "full_name").Updates(changes)
	return result.Error
}

// dbDeleteUser deletes a user from the User database table within an existing transaction. It also
// removes the membership association from all groups they currently belong to (including 'all').
func dbDeleteUser(user *User, tx *gorm.DB) error {

	if err := tx.Model(&user).Association("Groups").Clear(); err != nil {
		return err
	}

	result := tx.Delete(&user)
	return result.Error
}
