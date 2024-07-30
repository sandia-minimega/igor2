// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// BasicAuth implements IAuth interface
type BasicAuth struct{}

// NewBasicAuth instantiates the Basic implementation of IAuth
func NewBasicAuth() IAuth {
	return &BasicAuth{}
}

func (l *BasicAuth) authenticate(r *http.Request) (*User, error) {

	actionPrefix := "local login"

	username, password, ok := r.BasicAuth()
	if !ok {
		errLine := actionPrefix + " failed - problem reading basic auth header"
		return nil, fmt.Errorf(errLine)
	}

	// verify Igor knows the user
	user, fuErr := findUserForAuthN(username)
	if fuErr != nil {
		return nil, fuErr
	}

	authErr := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password))
	if authErr != nil {
		errLine := actionPrefix + " failed - incorrect password"
		bcErr := BadCredentialsError{msg: errLine}
		return nil, &bcErr
	}

	return user, nil
}

func createPasswordHash(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), 13)
}

// checkLocalPasswordRules determines if the input string meets the criteria for
// a user password that is locally maintained. These rules require the password
// to be 8-16 characters in length and must be composed of letters, numbers and
// punctuation/symbols with at least one from each category. All other types
// including whitespace are not allowed.
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
