// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"

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
