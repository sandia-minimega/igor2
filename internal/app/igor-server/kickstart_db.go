// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"gorm.io/gorm"
)

// dbCreateKS registers a new Kickstart to the db.
func dbCreateKS(ks *Kickstart, tx *gorm.DB) error {
	result := tx.Create(&ks)
	return result.Error
}

func dbReadKickstartTx(queryParams map[string]interface{}) (ks []Kickstart, err error) {
	err = performDbTx(func(tx *gorm.DB) error {
		ks, err = dbReadKS(queryParams, tx)
		return err
	})

	return ks, err
}

// dbReadKS returns ks objects matching the given parameters.
func dbReadKS(queryParams map[string]interface{}, tx *gorm.DB) (ks []Kickstart, err error) {

	tx.Preload("Users")

	// if no params given, return all kickstarts
	if len(queryParams) == 0 {
		result := tx.Find(&ks)
		return ks, result.Error
	}

	for key, val := range queryParams {
		switch val.(type) {
		case string:
			tx = tx.Where(key, val)
		case []string:
			queryStmt := key + " IN ?"
			tx = tx.Where(queryStmt, val)
		default:
			// we shouldn't reach this error because we already checked the param types
			logger.Error().Msgf("Incorrect parameter type received for %s: %v", key, val)
		}
	}

	result := tx.Find(&ks)
	return ks, result.Error
}

// dbEditKS applies changes to the target kickstart.
func dbEditKS(ks *Kickstart, changes map[string]interface{}, tx *gorm.DB) error {
	if len(changes) > 0 {
		result := tx.Model(&ks).Updates(changes)
		return result.Error
	}
	return nil
}

// dbDeleteKS deletes a kickstart from the Kickstart database table
func dbDeleteKS(ks *Kickstart, tx *gorm.DB) error {
	// Ideally, target has already been found in the db
	result := tx.Delete(&ks)
	return result.Error
}
