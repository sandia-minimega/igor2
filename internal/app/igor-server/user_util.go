// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"igor2/internal/pkg/common"
	"regexp"
	"strings"
)

// This matches most cases but can be more robust. All email strings should be forced to use lower case.
var emailCheckPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var nameCheckPattern = regexp.MustCompile(`^[a-z_]([a-z0-9_\-]){0,31}$`)
var fullNameCheckPattern = regexp.MustCompile(`^[a-zA-Z. -]{0,32}$`)

// checkUsernameRules determines if the input string meets the criteria for
// a valid username. Igor follows general username rules for Linux: can
// only begin with a letter or underscore and can include numbers
// and hyphen afterward. It has a limit of 32 characters.
func checkFullNameRules(name string) error {
	if !fullNameCheckPattern.MatchString(name) {
		return fmt.Errorf("%s is not allowed for full name field", name)
	}
	return nil
}

func checkUsernameRules(name string) error {
	if !nameCheckPattern.MatchString(name) {
		return fmt.Errorf("%s is not a legal username", name)
	}
	return nil
}

// checkEmailRules determines if the input string meets the criteria for a valid
// email address
func checkEmailRules(email string) error {
	if strings.Contains(email, "..") {
		return fmt.Errorf("%s is not a legal username", email)
	}
	if !emailCheckPattern.MatchString(email) {
		return fmt.Errorf("%s is not a legal email address", email)
	}
	return nil
}

func userSliceContains(users []User, name string) bool {
	for _, u := range users {
		if u.Name == name {
			return true
		}
	}
	return false
}

// userNamesOfUsers returns a list of usernames from the provided User list.
func userNamesOfUsers(users []User) []string {
	userNames := make([]string, len(users))
	for i, u := range users {
		userNames[i] = u.Name
	}
	return userNames
}

// usersFromNames returns a subset of User objects from the supplied User slice
// whose usernames match those in the provided string slice. It will silently
// ignore names in the string slice that have no match.
func usersFromNames(users []User, names []string) []User {

	var foundUsers []User
	for _, v := range names {
		for _, u := range users {
			if v == u.Name {
				foundUsers = append(foundUsers, u)
				continue
			}
		}
	}
	return foundUsers
}

// usernamesFromNames returns a subset of username strings from the supplied
// User slice whose usernames match those in the provided string slice. It will
// silently ignore names in the string slice that have no match.
func usernamesFromNames(users []User, names []string) []string {

	var foundUsers []string
	for _, v := range names {
		for _, u := range users {
			if v == u.Name {
				foundUsers = append(foundUsers, u.Name)
				continue
			}
		}
	}
	return foundUsers
}

// userIDsOfUsers returns a list of User IDs from the provided list of users.
func userIDsOfUsers(users []User) []int {
	userIDs := make([]int, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}
	return userIDs
}

// filterNonUsers takes a slice of User objects and a slice of usernames
// any names that are not among the slice of User objects are collected as
// a slice of names and returned
func filterNonUsers(users []User, names []string) []string {
	var inList bool
	var notFound []string
	for _, v := range names {
		inList = false
		for _, u := range users {
			if v == u.Name {
				inList = true
				continue
			}
		}
		if !inList {
			notFound = append(notFound, v)
		}
	}
	return notFound
}

// usernameDiff compares the contents of slice a to b, producing a list
// of names in b that were not found in a. Note that b should not contain
// the empty string or pure whitespace as the underlying data structure
// used to compare against a will ignore those string values.
func usernameDiff(a, b []string) []string {
	userMap := common.NewSet()
	userMap.Add(a...)
	var diff []string
	for _, x := range b {
		if !userMap.Contains(x) {
			diff = append(diff, x)
		}
	}
	return diff
}

func removeUserByName(users []User, username string) []User {
	for i, u := range users {
		if u.Name == username {
			return append(users[:i], users[i+1:]...)
		}
	}
	return users
}
