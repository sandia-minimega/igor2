// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"strings"

	"igor2/internal/pkg/common"
)

var tempProfilePrefix = "tpf_"

// checkProfileNameRules determines if the input string meets the criteria for
// a valid profile name.
func checkProfileNameRules(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("profile name cannot be empty")
	}
	if !stdNameCheckPattern.MatchString(name) {
		return fmt.Errorf("'%s' is not a legal profile name", name)
	}
	return isResourceNameMatch(name)
}

func checkReservedProfileNames(name string) error {
	if strings.HasPrefix(name, tempProfilePrefix) {
		return fmt.Errorf("profile name with prefix %v not allowed", tempProfilePrefix)
	}
	return nil
}

func generateDefaultProfileName(user *User) string {
	unique := false
	name := ""
	// make sure the generated name is unique for that user
	for !unique {
		name = tempProfilePrefix + common.RandSeq(4)
		if profiles, _ := dbReadProfilesTx(map[string]interface{}{"name": name, "owner_id": user.ID}); len(profiles) == 0 {
			unique = true
		}
	}
	return name
}

// profileNamesOfProfiles returns a list of Profile names from the provided list of profiles.
func profileNamesOfProfiles(profiles []Profile) []string {
	profileNames := make([]string, len(profiles))
	for i, p := range profiles {
		profileNames[i] = p.Name
	}
	return profileNames
}

// profileIDsOfProfiles returns a list of Profile IDs from
// the provided list of profiles.
func profileIDsOfProfiles(profiles []Profile) []int {
	profileIDs := make([]int, len(profiles))
	for i, p := range profiles {
		profileIDs[i] = p.ID
	}
	return profileIDs
}
