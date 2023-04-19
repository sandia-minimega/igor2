// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"net/http"

	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"
)

// doRegisterImage calls registerImage in a new transaction.
func doRegisterKickstart(r *http.Request) (ks *Kickstart, status int, err error) {
	status = http.StatusInternalServerError
	err = performDbTx(func(tx *gorm.DB) error {
		ks, status, err = registerKickstart(r, tx)
		return err
	})
	return
}

// registerImage looks in the request for either files attached to the multiform,
// or file name references if an admin placed files into the local staged folder manually.
// It will then locate and hash the files, check if the hash already exists in the db,
// if the image is new, store the files to the images folder and create a new image entry (KIref, hash, filenames)
// then return the new/existing image object
// NOTE: For now, we assume we're only dealing with KI pairs
func registerKickstart(r *http.Request, tx *gorm.DB) (ks *Kickstart, status int, err error) {

	// user := getUserFromContext(r)

	clog := hlog.FromRequest(r)
	// potential way of determining whether files were included and type based on count?
	clog.Debug().Msgf("Number of files attached: %v", len(r.MultipartForm.File))

	// we need to pull files from the multiform and stage them
	ks, err = saveKSFile(r)
	if err != nil {
		return ks, http.StatusInternalServerError, err
	}

	// Set user as owner
	user := getUserFromContext(r)
	ks.Owner = *user
	ks.OwnerID = user.ID

	dbAccess.Lock()
	defer dbAccess.Unlock()
	// create db entry of the image
	if err = dbCreateKS(ks, tx); err != nil {
		return ks, http.StatusInternalServerError, err
	}

	return ks, http.StatusCreated, nil
}
