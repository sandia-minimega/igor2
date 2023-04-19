// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"path/filepath"

	zl "github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"
)

func doDeleteDistroImage(distroImageName string, r *http.Request) (code int, err error) {
	clog := hlog.FromRequest(r)
	code = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		var images []DistroImage
		images, code, err = getImages([]string{distroImageName}, tx)
		if err != nil {
			return err
		}
		targetImage := images[0]
		if len(targetImage.Distros) > 0 {
			var dNames []string
			for _, distro := range targetImage.Distros {
				dNames = append(dNames, distro.Name)
			}
			return fmt.Errorf("cannot delete image while linked to existing distros. Delete the following distros first: %v", dNames)
		}

		return deleteDistroImage(&targetImage, tx, clog)
	}); err != nil {
		return
	}
	return http.StatusOK, nil
}

func deleteDistroImage(image *DistroImage, tx *gorm.DB, clog *zl.Logger) (err error) {

	// delete the image from the db first in case rollback happens
	clog.Info().Msgf("removing image %s from db", image.Name)
	if err = dbDeleteImage(image, tx); err != nil {
		return err
	}

	// get the path to the image folder
	targetPath := filepath.Join(igor.TFTPPath, igor.ImageStoreDir, image.ImageID)
	// destroy the folder and contents
	clog.Info().Msgf("removing image %s files from %v", image.Name, targetPath)
	return removeFolderAndContents(targetPath)
}
