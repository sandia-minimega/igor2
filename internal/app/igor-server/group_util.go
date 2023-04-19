// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"strings"
)

// checkGroupNameRules determines if the input string meets the criteria for a valid group name.
func checkGroupNameRules(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("group name cannot be empty")
	}
	if !stdNameCheckPattern.MatchString(name) {
		return fmt.Errorf("'%s' is not a legal group name", name)
	}

	return isResourceNameMatch(name)
}

// Apply this to create/edit/del operations to prevent use of names that are reserved or might be confusing.
func checkReservedGroupNames(name string) error {
	lname := strings.ToLower(name)
	if lname == GroupAll || lname == GroupNoneAlias || strings.HasPrefix(lname, GroupUserPrefix) ||
		strings.HasPrefix(lname, "admin") {
		return fmt.Errorf("group name '%s' is not allowed", name)
	}
	return nil
}

// groupSliceContains returns true if one of the groups in the slice has the given name, false otherwise.
func groupSliceContains(groups []Group, name string) bool {
	for _, g := range groups {
		if g.Name == name {
			return true
		}
	}
	return false
}

// removeGroup removes the given target Group from the given slice of groups.
func removeGroup(gSlice []Group, target *Group) []Group {
	for idx, v := range gSlice {
		if v.Name == target.Name {
			return append(gSlice[0:idx], gSlice[idx+1:]...)
		}
	}
	return gSlice
}

// getGroupIDsFromNames returns the database ID field of each named group.
func getGroupIDsFromNames(groupNames []string) ([]int, int, error) {
	if groupList, status, err := getGroupsTx(groupNames, false); err != nil {
		return nil, status, err
	} else if len(groupNames) != len(groupList) {
		return nil, http.StatusBadRequest, fmt.Errorf("number of groups retrieved from DB does not equal number of group names given")
	} else {
		return groupIDsOfGroups(groupList), http.StatusOK, nil
	}
}

// groupNamesOfGroups returns a list of Group names from
// the provided list of groups.
func groupNamesOfGroups(groups []Group) []string {
	groupNames := make([]string, len(groups))
	for i, g := range groups {
		groupNames[i] = g.Name
	}
	return groupNames
}

// groupIDsOfGroups returns a list of Group IDs from
// the provided list of groups.
func groupIDsOfGroups(groups []Group) []int {
	groupIDs := make([]int, len(groups))
	for i, g := range groups {
		groupIDs[i] = g.ID
	}
	return groupIDs
}

// getViewAccessibleGroups will filter the groupList to the ones that are
// visible to the given user based on their granted permissions.
func getViewAccessibleGroups(user *User, groupList []Group) (accessGroups []Group) {
	authzInfo, _ := user.getAuthzInfo()
	for _, g := range groupList {
		groupPerm, _ := NewPermission(NewPermissionString(PermGroups, g.Name, PermViewAction))
		if authzInfo.IsPermitted(groupPerm) {
			accessGroups = append(accessGroups, g)
		}
	}
	return
}
