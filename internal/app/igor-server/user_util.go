// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// This matches most cases but can be more robust. All email strings should be forced to use lower case.
var emailCheckPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var nameCheckPattern = regexp.MustCompile(`^[a-z_]([a-z0-9_\-]){0,31}$`)
var fullNameCheckPattern = regexp.MustCompile(`^[a-zA-Z. -]{0,32}$`)

// checkUsernameRules determines if the input string meets the criteria for
// a valid user name on igor. Igor follows general username rules for Linux: can
// only begin with a letter or underscore and can include numbers
// and hyphen afterwards. It has a limit of 32 characters.
func checkFullnameRules(name string) error {
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

// checkLocalPasswordRules determines if the input string meets the criteria for
// a user password that is locally maintained by igor. These rules require the password
// to be 8-16 characters in length and must be composed of letters, numbers and punctuation/symbols with
// at least one from each category. All other types including whitespace are not allowed.
func checkLocalPasswordRules(password string) error {
	// Go's regex support doesn't include lookahead ... making this a more manual process
	var letterFound bool
	var numberFound bool
	var specialFound bool
	minLength := 8
	maxLength := 16
	legalChars := 0

	for _, c := range password {
		switch {
		case unicode.IsNumber(c):
			numberFound = true
			legalChars++
		case unicode.IsLetter(c):
			letterFound = true
			legalChars++
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			specialFound = true
			legalChars++
		}
	}

	if legalChars != len(password) {
		return fmt.Errorf("password contains characters that are not allowed")
	}

	if legalChars < minLength {
		return fmt.Errorf("password must have minimum length of 8 characters")
	} else if legalChars > maxLength {
		return fmt.Errorf("password cannot exceed maximum length of 16 characters")
	}

	var missingStuff string
	if !numberFound {
		missingStuff += "at least 1 number"
	}

	if !letterFound {
		if !numberFound {
			missingStuff += ", "
		}
		missingStuff += "at least 1 letter"
	}

	if !specialFound {
		if !numberFound || !letterFound {
			missingStuff += ", "
		}
		missingStuff += "at least 1 special character"
	}

	if missingStuff != "" {
		return fmt.Errorf("new password is missing %s", missingStuff)
	}

	return nil
}

func getPasswordHash(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), 13)
}

func userSliceContains(users []User, name string) bool {
	for _, u := range users {
		if u.Name == name {
			return true
		}
	}
	return false
}

// userNamesOfUsers returns a list of User names from
// the provided list of users.
func userNamesOfUsers(users []User) []string {
	userNames := make([]string, len(users))
	for i, u := range users {
		userNames[i] = u.Name
	}
	return userNames
}

// userIDsOfUsers returns a list of User IDs from
// the provided list of users.
func userIDsOfUsers(users []User) []int {
	userIDs := make([]int, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}
	return userIDs
}

// filterNonMembers take s a slice of User objects and a slice of user names
// any names that are not among the slice of User objects are collected as
// a slice of names and returned
func filterNonMembers(users []User, names []string) []string {
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
