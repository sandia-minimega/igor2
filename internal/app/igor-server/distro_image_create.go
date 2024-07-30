// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"
)

// doRegisterImage calls registerImage in a new transaction.
func doRegisterImage(r *http.Request) (image *DistroImage, status int, err error) {
	status = http.StatusInternalServerError
	err = performDbTx(func(tx *gorm.DB) error {
		image, status, err = registerImage(r, tx)
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
func registerImage(r *http.Request, tx *gorm.DB) (image *DistroImage, status int, err error) {
	clog := hlog.FromRequest(r)
	// potential way of determining whether files were included and type based on count?
	clog.Debug().Msgf("Number of files attached: %v", len(r.MultipartForm.File))

	// net-boot only: we're getting a KI pair
	// check for included staged file names, admin may have manually placed files in staging folder for us
	image = detectStagedFiles(r)
	if image == nil {
		// we need to pull files from the multiform and stage them
		image, err = stageUploadedFiles(r)
		if err != nil {
			return image, http.StatusInternalServerError, err
		}
	}

	// is image intended for local installation/booting?
	if strings.ToLower(r.FormValue("localBoot")) == "true" {
		image.LocalBoot = true
	}

	// get boot type(s) of image
	if boots, ok := r.Form["boot"]; ok {
		for _, boot := range boots {
			if strings.ToLower(boot) == "bios" {
				image.BiosBoot = true
			}
			if strings.ToLower(boot) == "uefi" {
				image.UefiBoot = true
			}
		}
	} else {
		return image, http.StatusBadRequest, fmt.Errorf("at least one value required image boot type")
	}

	// set image OS breed value, if given, otherwise put generic as default
	breed := strings.ToLower(r.FormValue("breed"))
	validBreed := hasValidBreed(breed)
	if breed != "" && !validBreed {
		return image, http.StatusBadRequest, fmt.Errorf("invalid value for required image breed - %s", breed)
	}
	if breed == "" {
		breed = "generic"
	}
	image.Breed = breed

	// ensure image file(s) exist in the image store
	image, err = processImage(image, tx)
	if err != nil {
		return image, http.StatusInternalServerError, err
	}
	return image, http.StatusOK, nil
}

// stageUploadedFiles extracts files inside the multipart form and saves them to the
// igor_staged_images directory to be processed into the igor_images directory later
func stageUploadedFiles(r *http.Request) (*DistroImage, error) {
	// Will expand eventually to detect and accomodate different file type (ex. iso)
	// ex. try different file keys until success, if no successes, return error
	var image *DistroImage

	// try kernel file
	key := "kernelFile"
	targetFile, handler, err := r.FormFile(key)
	if err != nil {
		// no kernel file found (may not return at this point)
		return nil, err
	}
	defer targetFile.Close()

	tempPath, sfErr := stageFile(targetFile, handler.Filename)
	if sfErr != nil {
		return nil, sfErr
	}
	// add file.kernel to the image
	image = &DistroImage{
		Type:   DistroKI,
		Kernel: filepath.Base(tempPath),
	}

	// upload initrd file
	key = "initrdFile"
	targetFile, handler, err = r.FormFile(key)
	if err != nil {
		return nil, err
	}
	defer targetFile.Close()

	tempPath, err = stageFile(targetFile, handler.Filename)
	if err != nil {
		return nil, err
	}
	// add file.initrd to the return slice
	image.Initrd = filepath.Base(tempPath)

	return image, nil
}

// processImage locates the image files within the igor_staged_images directory, hashes them
// into a unique ID, checks for duplicates using the hash. If unique, it will generate
// a refID from the hash, then send the files on to be moved into the igor_images/hashID
// directory
// TODO: currently hardcoded to handle KI pairs only, need to add ISO support later
func processImage(image *DistroImage, tx *gorm.DB) (*DistroImage, error) {
	switch image.Type {
	case DistroKI:
		// setup paths
		kPath := filepath.Join(igor.Server.ImageStagePath, image.Kernel)
		iPath := filepath.Join(igor.Server.ImageStagePath, image.Initrd)
		// make sure both files exist
		if err := pathExists(kPath); err != nil {
			return image, err
		}
		if err := pathExists(iPath); err != nil {
			return image, err
		}
		// hash the KI pair
		hash, err := hashKIPair(kPath, iPath)
		if err != nil {
			return image, err
		}
		image.ImageID = hash
	default:
		return image, fmt.Errorf("image type not recognized: %v", image.Type)
	}

	// is this image already in our collection?
	images, err := dbReadImage(map[string]interface{}{"image_id": image.ImageID}, tx)
	if err != nil {
		return image, err
	}
	if len(images) > 0 {
		// this image already exists, stop and return the existing image
		// but first, destroy the staged image files
		destroyStagedImages(image)
		return &images[0], nil
	}

	// generate ref from hash
	image.Name = refFromHash(image.Type, image.ImageID)
	if image.Name == "" {
		return image, fmt.Errorf("error: failed to create image ref from type: %v and hash: %v", image.Type, image.ImageID)
	}

	// store files to image folder
	if err = processImageFiles(image); err != nil {
		return image, err
	}

	dbAccess.Lock()
	defer dbAccess.Unlock()
	// create db entry of the image
	if err = dbCreateImage(image, tx); err != nil {
		return image, err
	}

	// on success, destroy staged image files
	destroyStagedImages(image)

	return image, nil
}

// returns nil if path exists, error otherwise
func pathExists(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if errors.Is(err, os.ErrNotExist) {
		// target file/path does *not* exist
		return err
	} else {
		// Schrodinger: file may or may not exist. See err for details.
		err = fmt.Errorf("unknown error while attemting to determine file existence for path %s: %v", path, err.Error())
		return err
	}
}

// processImageFiles copies the image files from the igor_staged_images folder to
// igor_images/hashID folder
func processImageFiles(image *DistroImage) (err error) {
	// make sure the image dir path exists in the image store directory
	targetPath := filepath.Join(igor.TFTPPath, igor.ImageStoreDir, image.ImageID)
	// check whether target path already exists
	if _, err = os.Stat(targetPath); err == nil {
		// image path already exists, but no DB entry represents it? Fail!
		return fmt.Errorf("error: path %v already exists. Remove this folder and its contents and try again", targetPath)
	} else if errors.Is(err, os.ErrNotExist) {
		// create the image folder in the image store
		err = os.Mkdir(targetPath, 0755)
		if err != nil {
			return err
		}
	} else {
		return err
	}

	// move files based on type
	switch image.Type {
	case DistroKI:
		kPath := filepath.Join(igor.Server.ImageStagePath, image.Kernel)
		iPath := filepath.Join(igor.Server.ImageStagePath, image.Initrd)
		err = copyFile(kPath, filepath.Join(targetPath, image.Kernel))
		if err != nil {
			return err
		}
		err = copyFile(iPath, filepath.Join(targetPath, image.Initrd))
		if err != nil {
			return err
		}
	default:
		// unknown or unlisted file type, roll back the operation and return error
		if rmErr := removeFolderAndContents(targetPath); rmErr != nil {
			return rmErr
		}
		return fmt.Errorf("unknown image type: %v, operation aborted", image.Type)
	}

	return nil
}

// stageFile takes a file object extracted from a multipart form
// and saves it to the staged folder using the given file name fName
func stageFile(src multipart.File, f string) (target string, err error) {
	// get spearate path and filename in case a full path was captured during upload
	_, fName := path.Split(f)
	filePath := filepath.Join(igor.Server.ImageStagePath, fName)

	// make sure we don't already have a file of the same name
	if _, err = os.Stat(filePath); err == nil {
		// file/path already exists
		return "", &FileAlreadyExistsError{msg: fmt.Sprintf("File already exists: %s", filePath)}
	} else if errors.Is(err, os.ErrNotExist) {
		// target file/path does *not* exist, write file out to staging directory
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

// hashKIPair takes a kernel and initrd files, creates a hash of each,
// appends them together, then encodes the resulting hash to a hex value
func hashKIPair(kPath, iPath string) (ref string, err error) {
	var kernel, initrd *os.File
	kernel, err = os.Open(kPath)
	if err != nil {
		return "", err
	}
	defer kernel.Close()
	initrd, err = os.Open(iPath)
	if err != nil {
		return "", err
	}
	defer initrd.Close()
	kiHash := sha1.New()
	if _, err = io.Copy(kiHash, kernel); err != nil {
		return "", fmt.Errorf("unable to hash file %v: %v", kPath, err)
	}
	if _, err = io.Copy(kiHash, initrd); err != nil {
		return "", fmt.Errorf("unable to hash file %v: %v", iPath, err)
	}
	ref = hex.EncodeToString(kiHash.Sum(nil))

	return ref, nil
}

// refFromHash builds a value of form <prefix> followed by the first
// 8 characters from the image's hash ID
func refFromHash(prefix, hash string) string {
	if len(hash) < 8 {
		return ""
	}
	return prefix + hash[:8]
}

// copyFile copies a file from source path to target path
func copyFile(srcPath, targetPath string) error {
	// Open original file
	originalFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer originalFile.Close()

	// Create new file
	var newFile *os.File
	newFile, err = os.Create(targetPath)
	if err != nil {
		logger.Error().Msgf("%v", err)
		return err
	}
	defer newFile.Close()

	// Copy the bytes to destination from source
	_, err = io.Copy(newFile, originalFile)
	if err != nil {
		logger.Error().Msgf("%v", err)
		return err
	}
	// Commit the file contents
	// Flushes memory to disk
	err = newFile.Sync()
	if err != nil {
		return err
	}
	return nil
}

// // determine if references to staged file(s) were given to us
// // and what types of files they are. Returns an Image obj
// // containing the image type, or nil otherwise
func detectStagedFiles(r *http.Request) *DistroImage {
	// expand in the future to accomodate different image type
	kFile := r.FormValue("kstaged")
	iFile := r.FormValue("istaged")
	if kFile != "" && iFile != "" {
		return &DistroImage{
			Type:   DistroKI,
			Kernel: kFile,
			Initrd: iFile,
		}
	}
	return nil
}

// destroyStagedImages deletes the specified image files from
// the igor_staged_images directory
func destroyStagedImages(image *DistroImage) {
	var paths []string
	switch image.Type {
	case DistroKI:
		// setup paths
		kPath := filepath.Join(igor.Server.ImageStagePath, image.Kernel)
		iPath := filepath.Join(igor.Server.ImageStagePath, image.Initrd)
		// make sure both files exist
		if err := pathExists(kPath); err == nil {
			paths = append(paths, kPath)
		}
		if err := pathExists(iPath); err == nil {
			paths = append(paths, iPath)
		}
		deleteStagedFiles(paths)
	}
}
