// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"gorm.io/gorm"
)

func doUpdateKS(targetName string, r *http.Request) (code int, err error) {

	changes := map[string]interface{}{}
	// clog := hlog.FromRequest(r)
	code = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		// get the target
		kss, err := dbReadKS(map[string]interface{}{"name": targetName}, tx)
		if err != nil {
			return err
		}
		if len(kss) > 1 {
			return fmt.Errorf("multiple kickstarts returned for target name %s", targetName)
		}
		if len(kss) == 0 {
			return fmt.Errorf("no kickstarts returned for target name %s", targetName)
		}
		target := kss[0]
		oldFileName := target.Filename
		key := "kickstart"
		targetFile, handler, fileErr := r.FormFile(key)
		if fileErr == nil {
			defer targetFile.Close()
			if oldFileName != handler.Filename {
				_, sfErr := saveNewKickstartFile(targetFile, handler.Filename, false)
				if sfErr != nil {
					return sfErr
				}
			} else {
				_, rfErr := saveNewKickstartFile(targetFile, handler.Filename, true)
				if rfErr != nil {
					return rfErr
				}
			}

			changes["filename"] = handler.Filename
			// changes["name"] = strings.Split(handler.Filename, ".")[0]
		}

		name := r.FormValue("name")
		if name != "" {
			changes["name"] = name
		}

		if changes != nil {
			err := dbEditKS(&target, changes, tx)
			if err != nil {
				return err
			}
			if changes["filename"] != nil && oldFileName != changes["filename"] {
				// if we swapped the file and file name of new ks was different, delete old ks file
				filePath := filepath.Join(igor.TFTPPath, igor.KickstartDir, oldFileName)
				return os.Remove(filePath)
			}
			return nil
		} else {
			return fmt.Errorf("no proposed changes to kickstart detected")
		}

	}); err == nil {
		code = http.StatusOK
	}
	return
}
