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

func doDeleteKS(ksName string, r *http.Request) (code int, err error) {
	clog := hlog.FromRequest(r)
	code = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		kss, err := dbReadKS(map[string]interface{}{"name": ksName}, tx)
		if err != nil {
			return err
		}
		if len(kss) > 1 {
			return fmt.Errorf("multiple kickstarts returned for target name %s", ksName)
		}
		if len(kss) == 0 {
			return fmt.Errorf("no kickstarts returned for target name %s", ksName)
		}
		ks := kss[0]

		// are there Distros which reference this?
		distros, err := dbReadDistros(map[string]interface{}{"kickstart_id": ks.ID}, tx)
		if err != nil {
			return err
		}
		if len(distros) > 0 {
			var dNames []string
			for _, distro := range distros {
				dNames = append(dNames, distro.Name)
			}
			return fmt.Errorf("cannot delete kickstart while linked to existing distros. Delete the following distros first: %v", dNames)
		}

		return deleteKS(&ks, tx, clog)
	}); err != nil {
		return
	}
	return http.StatusOK, nil
}

func deleteKS(ks *Kickstart, tx *gorm.DB, clog *zl.Logger) (err error) {

	// delete the kickstart from the db first in case rollback happens
	clog.Info().Msgf("removing kickstart %s from db", ks.Filename)
	if err = dbDeleteKS(ks, tx); err != nil {
		return err
	}

	// get the path to the image folder
	targetPath := filepath.Join(igor.TFTPPath, igor.KickstartDir, ks.Filename)
	// destroy the file
	clog.Info().Msgf("removing kickstart %s file from %v", ks.Filename, targetPath)
	return deleteStagedFiles([]string{targetPath})
}
