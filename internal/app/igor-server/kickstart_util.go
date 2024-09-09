// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// saveKSFile pulls the kickstart file out of the request object
// and saves it to the ks path, then creates a kickstart obj
// with file info
func saveKSFile(r *http.Request) (*Kickstart, error) {
	var ks *Kickstart

	// try kickstart file
	key := "kickstart"
	targetFile, handler, err := r.FormFile(key)
	if err != nil {
		// no kickstart file found
		return nil, err
	}
	defer targetFile.Close()

	fileName := handler.Filename
	name := strings.Split(fileName, ".")[0]
	_, sfErr := saveNewKickstartFile(targetFile, fileName)
	if sfErr != nil {
		return nil, sfErr
	}
	// add file to the kickstart object
	ks = &Kickstart{
		Name:     name,
		Filename: fileName,
	}

	return ks, nil
}

// SaveFile takes a file object extracted from a multipart form
// and saves it to the staged folder using the given file name f
func saveNewKickstartFile(src multipart.File, f string) (target string, err error) {
	// get separate path and filename in case a full path was captured during upload
	_, fName := path.Split(f)
	filePath := filepath.Join(igor.TFTPPath, igor.KickstartDir, fName)

	// make sure we don't already have a file of the same name
	if _, err = os.Stat(filePath); err == nil {
		// file/path already exists
		return "", &FileAlreadyExistsError{msg: fmt.Sprintf("a kickstart file is already registered with file name: %s", fName)}
	} else if errors.Is(err, os.ErrNotExist) {
		// target file/path does *not* exist, write file out to kickstart directory
		var tempFile *os.File
		if tempFile, err = os.Create(filePath); err != nil {
			return "", err
		} else {
			defer tempFile.Close()
			// Copy the uploaded file to the created file on the filesystem
			if _, err = io.Copy(tempFile, src); err != nil {
				return "", err
			}
			return filePath, nil
		}
	} else {
		// Schrodinger: file may or may not exist. See err for details.
		if err == nil {
			err = fmt.Errorf("unknown error while attemting to determine file existence: %s", filePath)
		}
		return "", err
	}
}

// overwriteFile takes a file object extracted from a multipart form
// and saves it to the staged folder using the given file name fName
func replaceFile(src multipart.File, f string) (target string, err error) {
	// get separate path and filename in case a full path was captured during upload
	_, fName := path.Split(f)
	filePath := filepath.Join(igor.TFTPPath, igor.KickstartDir, fName)
	var tempFile *os.File
	if tempFile, err = os.Create(filePath); err != nil {
		return "", err
	} else {
		defer tempFile.Close()
		// Copy the uploaded file to the created file on the filesystem
		if _, err = io.Copy(tempFile, src); err != nil {
			return "", err
		}
		return filePath, nil
	}
}
