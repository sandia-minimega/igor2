// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func checkDistroNameRules(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("distro name cannot be empty")
	}
	if !stdNameCheckPattern.MatchString(name) {
		return fmt.Errorf("'%s' is not a legal distro name", name)
	}
	return isResourceNameMatch(name)
}

func checkDistroImageRefRules(ref string) error {
	if len(ref) == 0 {
		return fmt.Errorf("distro imageRef cannot be empty")
	}
	if !stdImageRefCheckPattern.MatchString(ref) {
		return fmt.Errorf("'%s' is not a legal distro imageRef", ref)
	}
	return nil
}

func checkDistroImageIDRules(ref string) error {
	if len(ref) == 0 {
		return fmt.Errorf("distro Image ID cannot be empty")
	}
	if !stdImageIDCheckPattern.MatchString(ref) {
		return fmt.Errorf("'%s' is not a legal distro image ID value", ref)
	}
	return nil
}

func checkFileRules(ref string) error {
	if len(ref) == 0 {
		return fmt.Errorf("file name cannot be empty")
	}
	if !fileNameCheckPattern.MatchString(ref) {
		return fmt.Errorf("'%s' is not a legal file name", ref)
	}
	return nil
}

func removeFolderAndContents(dir string) error {
	// open folder path
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	// find and delete all files in the folder path
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	// finally, delete the folder path
	err = os.Remove(dir)
	if err != nil {
		return err
	}
	return nil
}

// deleteStagedFiles requires the full path of each file to be deleted
func deleteStagedFiles(paths []string) error {
	for _, p := range paths {
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}
	return nil
}

func getAllowedDistros(user *User) ([]Distro, error) {
	return dbReadDistrosTx(map[string]interface{}{"groups": groupIDsOfGroups(user.Groups)})
}

func scopeDistrosToUser(distros []Distro, user *User) []Distro {
	// if the user isn't admin, add user's groups to the search params
	// this will ensure distro query will only include distros that
	// have at least one of the user's groups in it.
	if userElevated(user.Name) {
		return distros
	}
	var results []Distro
	for _, distro := range distros {
		allowed := false
		for _, group := range distro.Groups {
			if groupSliceContains(user.Groups, group.Name) {
				allowed = true
				break
			}
		}
		if allowed {
			results = append(results, distro)
		}
	}

	return results
}

// distroNamesOfDistros returns a list of Distro names from
// the provided list of distros.
func distroNamesOfDistros(distros []Distro) []string {
	distroNames := make([]string, len(distros))
	for i, d := range distros {
		distroNames[i] = d.Name
	}
	return distroNames
}

// distroIDsOfDistros returns a list of Distro IDs from
// the provided list of distros.
func distroIDsOfDistros(distros []Distro) []int {
	distroIDs := make([]int, len(distros))
	for i, d := range distros {
		distroIDs[i] = d.ID
	}
	return distroIDs
}

func getImageStorePath(imageID string) string {
	return filepath.Join(igor.TFTPPath, igor.ImageStoreDir, imageID)
}

func parseKernelInfo(image *DistroImage) (kInfo string, breed string) {

	kInfo = "not found"
	breed = ""

	targetPath := filepath.Join(getImageStorePath(image.ImageID), image.Kernel)

	fileCmdOut, fileCmdErr := processWrapper("file", targetPath)
	if fileCmdErr != nil {
		logger.Warn().Msgf("file command error on '%s': %v", targetPath, fileCmdErr)
	}
	if len(fileCmdOut) > 0 {

		kernVerInfo := strings.ToLower(strings.Split(fileCmdOut, ": ")[1])
		logger.Debug().Msgf("file command output: %s", kernVerInfo)
		for _, b := range DistroBreed {
			if strings.Contains(kernVerInfo, b) {
				breed = b
				break
			}
		}

		verPattern := `version (\S+)`
		verRegex := regexp.MustCompile(verPattern)
		match := verRegex.FindStringSubmatch(kernVerInfo)
		if len(match) > 1 {
			kInfo = match[1]
		}
	}

	logger.Debug().Msgf("kernel version: %s, breed: %s", kInfo, breed)
	return
}
