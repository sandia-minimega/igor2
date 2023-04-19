// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"gorm.io/gorm"
)

// dbCreateImage registers a new image (K/I pair) to the db.
func dbCreateImage(image *DistroImage, tx *gorm.DB) error {
	result := tx.Create(&image)
	return result.Error
}

// dbReadImage returns images matching the given parameters.
func dbReadImage(queryParams map[string]interface{}, tx *gorm.DB) (images []DistroImage, err error) {

	tx = tx.Preload("Distros")

	// if no params given, return all users
	if len(queryParams) == 0 {
		result := tx.Find(&images)
		return images, result.Error
	}

	for key, val := range queryParams {
		switch val.(type) {
		case string:
			tx = tx.Where(key, val)
		case []string:
			querystmt := key + " IN ?"
			tx = tx.Where(querystmt, val)
		default:
			// we shouldn't reach this error because we already checked the param types
			logger.Error().Msgf("Incorrect parameter type received for %s: %v", key, val)
		}
	}

	result := tx.Find(&images)
	return images, result.Error
}

// dbDeleteImage deletes an image from the Image database table
func dbDeleteImage(image *DistroImage, tx *gorm.DB) error {
	// Ideally, target has already been found in the db
	result := tx.Delete(&image)
	return result.Error
}
