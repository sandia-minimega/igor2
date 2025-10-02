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

	"github.com/google/uuid"
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

// registerImage processes the request to register an image.
func registerImage(r *http.Request, tx *gorm.DB) (image *DistroImage, status int, err error) {
	clog := hlog.FromRequest(r)
	clog.Debug().Msgf("Number of files attached: %v", len(r.MultipartForm.File))
	tempFiles := []string{}

	breed := strings.ToLower(r.FormValue("breed"))
	if breed != "" && !hasValidBreed(breed) {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid value for required image breed - %s", breed)
	}
	if breed == "" {
		breed = "generic-linux"
	}
	image = detectStagedFiles(r)
	if image == nil {
		image, tempFiles, err = stageUploadedFiles(r)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	} else {
		tempK := image.Kernel + ".kernel"
		tempI := image.Initrd + ".initrd"
		tempFiles = append(tempFiles, tempK, tempI)
	}

	if strings.ToLower(r.FormValue("localBoot")) == "true" {
		image.LocalBoot = true
	}

	image.Breed = breed
	image, err = processImage(image, tempFiles, tx)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return image, http.StatusOK, nil
}

// stageUploadedFiles extracts files from the multipart form and saves them to the staging directory.
func stageUploadedFiles(r *http.Request) (image *DistroImage, tempFiles []string, err error) {
	image = &DistroImage{Type: DistroKI}

	kernelFile, tempKFile, err := saveUploadedFile(r, "kernelFile")
	if err != nil {
		return nil, tempFiles, err
	}
	tempKFile = tempKFile + ".kernel"
	image.Kernel = kernelFile
	tempFiles = append(tempFiles, tempKFile)

	initrdFile, tempIFile, err := saveUploadedFile(r, "initrdFile")
	if err != nil {
		return nil, tempFiles, err
	}
	tempIFile = tempIFile + ".initrd"
	image.Initrd = initrdFile

	tempFiles = append(tempFiles, tempIFile)
	return image, tempFiles, nil
}

// saveUploadedFile saves an uploaded file to the staging directory.
func saveUploadedFile(r *http.Request, key string) (string, string, error) {
	file, handler, err := r.FormFile(key)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	return stageFile(file, handler.Filename)
}

// processImage processes the image files and stores them in the image directory.
func processImage(image *DistroImage, tempFiles []string, tx *gorm.DB) (*DistroImage, error) {
	tempK := ""
	tempI := ""
	switch image.Type {
	case DistroKI:
		tf := tempFiles[0]
		if strings.HasSuffix(tf, ".kernel") {
			tempK = strings.TrimSuffix(tf, ".kernel")
			tempI = strings.TrimSuffix(tempFiles[1], ".initrd")
		} else {
			tempK = strings.TrimSuffix(tempFiles[1], ".kernel")
			tempI = strings.TrimSuffix(tf, ".initrd")
		}
		stagedKernel := filepath.Join(igor.Server.ImageStagePath, tempK)
		stagedInitrd := filepath.Join(igor.Server.ImageStagePath, tempI)
		if err := checkFileExists(stagedKernel); err != nil {
			return nil, err
		}
		if err := checkFileExists(stagedInitrd); err != nil {
			return nil, err
		}
		hash, err := hashKIPair(stagedKernel, stagedInitrd)
		if err != nil {
			destroyStagedImages([]string{tempK, tempI})
			return nil, err
		}
		image.ImageID = hash
	default:
		return nil, fmt.Errorf("image type not recognized: %v", image.Type)
	}

	existingImages, err := dbReadImage(map[string]interface{}{"image_id": image.ImageID}, 0, tx)

	if err != nil {
		destroyStagedImages([]string{tempK, tempI})
		return nil, err
	}
	if len(existingImages) > 0 {
		destroyStagedImages([]string{tempK, tempI})
		return &existingImages[0], nil
	}

	image.Name = refFromHash(image.Type, image.ImageID)
	if image.Name == "" {
		destroyStagedImages([]string{tempK, tempI})
		return nil, fmt.Errorf("failed to create image ref from type: %v and hash: %v", image.Type, image.ImageID)
	}

	if err = processImageFiles(image, tempFiles); err != nil {
		destroyStagedImages([]string{tempK, tempI})
		return nil, err
	}

	dbAccess.Lock()
	defer dbAccess.Unlock()
	if err = dbCreateImage(image, tx); err != nil {
		destroyStagedImages([]string{tempK, tempI})
		return nil, err
	}

	enqueueInitrdJob(image)

	// on success, destroy staged image files
	destroyStagedImages([]string{tempK, tempI})
	return image, nil
}

// checkFileExists checks if a file exists at the given path.
func checkFileExists(path string) error {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return err
		}
		return fmt.Errorf("unknown error while checking file existence for path %s: %v", path, err)
	}
	return nil
}

// processImageFiles copies the image files from the staging directory to the image directory.
func processImageFiles(image *DistroImage, srcFiles []string) error {
	targetPath := getImageStorePath(image.ImageID)
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return err
	}

	switch image.Type {
	case DistroKI:
		tempK := ""
		tempI := ""
		tf := srcFiles[0]
		if strings.HasSuffix(tf, ".kernel") {
			tempK = strings.TrimRight(tf, ".kernel")
			tempI = strings.TrimRight(srcFiles[1], ".initrd")
		} else {
			tempK = strings.TrimRight(srcFiles[1], ".kernel")
			tempI = strings.TrimRight(tf, ".initrd")
		}
		kSrc := filepath.Join(igor.Server.ImageStagePath, tempK)
		iSrc := filepath.Join(igor.Server.ImageStagePath, tempI)
		kTarget := filepath.Join(targetPath, image.Kernel)
		iTarget := filepath.Join(targetPath, image.Initrd)

		if err := copyFile(kSrc, kTarget); err != nil {
			return err
		}
		if err := copyFile(iSrc, iTarget); err != nil {
			return err
		}
		image.KernelInfo, image.Breed = parseKernelInfo(image)

	default:
		if err := removeFolderAndContents(targetPath); err != nil {
			return err
		}
		return fmt.Errorf("unknown image type: %v, operation aborted", image.Type)
	}

	return nil
}

// stageFile saves a file to the staging directory.
func stageFile(src multipart.File, f string) (fName, tempFName string, err error) {
	_, fName = path.Split(f)
	tfn, err := uuid.NewRandom()
	tempFName = tfn.String()
	filePath := filepath.Join(igor.Server.ImageStagePath, tempFName)
	if _, err := os.Stat(filePath); err == nil {
		os.Remove(filePath)
	}
	tempFile, err := os.Create(filePath)
	if err != nil {
		return
	}
	defer tempFile.Close()
	if _, err = io.Copy(tempFile, src); err != nil {
		return
	}
	return
}

// hashKIPair hashes the kernel and initrd files and returns the hash.
func hashKIPair(kPath, iPath string) (string, error) {
	kernel, err := os.Open(kPath)
	if err != nil {
		return "", err
	}
	defer kernel.Close()

	initrd, err := os.Open(iPath)
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
	return hex.EncodeToString(kiHash.Sum(nil)), nil
}

// refFromHash generates a reference from the hash.
func refFromHash(prefix, hash string) string {
	if len(hash) < 8 {
		return ""
	}
	return prefix + hash[:8]
}

// copyFile copies a file from source to target.
func copyFile(srcPath, targetPath string) error {
	originalFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer originalFile.Close()

	newFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer newFile.Close()

	if _, err = io.Copy(newFile, originalFile); err != nil {
		return err
	}
	return newFile.Sync()
}

// detectStagedFiles checks for staged files in the request.
func detectStagedFiles(r *http.Request) *DistroImage {
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

// destroyStagedImages deletes the specified image files from the staging directory.
func destroyStagedImages(fnames []string) {
	var paths []string
	for _, path := range fnames {
		thisPath := filepath.Join(igor.Server.ImageStagePath, path)
		if err := checkFileExists(thisPath); err == nil {
			paths = append(paths, thisPath)
		}
	}
	deleteStagedFiles(paths)
}
