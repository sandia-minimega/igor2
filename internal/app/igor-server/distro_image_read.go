// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

// doReadDistroImages performs a DB lookup of Distro records that match the provided queryParams. It will
// return these as a list which can also be empty/nil if no matches were found. It will also pass back any
// encountered GORM errors with status code 500.
func doReadDistroImages() (images []DistroImage, status int, err error) {

	err = performDbTx(func(tx *gorm.DB) error {
		images, err = dbReadImage(map[string]interface{}{}, 0, tx)
		return err
	})
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return images, http.StatusOK, nil
}

// getImages is a convenience method to perform a lookup of images based on list of provided names.
// It will be successful as long as at least one image is found, otherwise it will return a NotFound error.
//
//	list,200,nil if any named image found
//	nil,404,err if no named image found
//	nil,500,err if db error
func getImages(imageNames []string, tx *gorm.DB) ([]DistroImage, int, error) {
	// profile search by name must always be in the context of their owner
	images, err := dbReadImage(map[string]interface{}{"name": imageNames}, 0, tx)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	} else if len(images) == 0 {
		return nil, http.StatusNotFound, fmt.Errorf("image '%s' not found", strings.Join(imageNames, ","))
	}

	return images, http.StatusOK, nil
}

// imageExists will perform a simple query to see if an image exists in the
// database. It will pass back any encountered GORM errors.
func imageExists(name string, tx *gorm.DB) (found bool, err error) {
	queryParams := map[string]interface{}{"name": name}
	iList, findErr := dbReadImage(queryParams, 0, tx)
	if findErr != nil {
		return false, findErr
	}
	if len(iList) > 0 {
		return true, nil
	}
	return false, nil
}
