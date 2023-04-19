// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"igor2/internal/pkg/api"
	"net/http"
	"strings"
	"time"

	"igor2/internal/pkg/common"

	"github.com/rs/zerolog/hlog"
)

const (
	DefaultLocalUserPassword = "changeMe$1"
	DefaultTokenDuration     = 72
	MaxTokenDuration         = 720
)

// IAuth is an interface that provides an authentication process based on a strategy, e.g. basic local, ldap, token, etc.
type IAuth interface {
	authenticate(r *http.Request) (*User, error)
}

func initAuth() {
	igor.AuthBasic = NewBasicAuth()
	igor.AuthToken = NewTokenAuth()
	// attach igor.AuthSecondary to the implementation based
	// on config settings or nil if none present
	scheme := strings.ToLower(igor.Auth.Scheme)
	if strings.Contains(scheme, "ldap") {
		igor.AuthSecondary = NewLdapAuth()
	} else {
		igor.AuthSecondary = nil
	}
}

func authnHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		clog := hlog.FromRequest(r)
		actionPrefix := "user authentication"

		var user *User
		var err error

		// Not a BasicAuth request (or not a valid one), so try token verify

		user, err = igor.AuthToken.authenticate(r)

		if err != nil {
			rb := common.NewResponseBody()
			errLine := actionPrefix + " failed - " + err.Error()
			rb.Message = errLine
			switch err.(type) {
			case *BadCredentialsError:
				if _, _, ok := r.BasicAuth(); ok {
					break
				}
				// token verify failed and auth header was not set to basic,
				// local/secondary auth will fail
				// redirect to login if you are not igorweb
				if r.URL.Path != api.Login && strings.HasPrefix(r.UserAgent(), IgorCliPrefix) {
					http.Redirect(w, r, api.Login, http.StatusTemporaryRedirect)
					return
				}
				// otherwise bounce the user
				clog.Warn().Msgf(errLine)
				makeJsonResponse(w, http.StatusUnauthorized, rb)
				return

			default:
				clog.Error().Msgf(errLine)
				makeJsonResponse(w, http.StatusInternalServerError, rb)
				return
			}

			if user, err = doPasswordAuth(w, r); err != nil {
				// doPasswordAuth makes json responses for failures
				return
			}
		}

		rCopy := addUserToContext(r, user)
		handler.ServeHTTP(w, rCopy)
	})
}

// Wraps getUsersTx but returns BadCredentialsError if the user is not found.
func findUserForAuthN(username string) (*User, error) {
	users, status, err := getUsersTx([]string{username}, true)
	if err != nil {
		if status >= http.StatusInternalServerError {
			return nil, err
		} else if status == http.StatusNotFound {
			err = &BadCredentialsError{msg: err.Error()}
			return nil, err
		}
	}

	return &users[0], nil
}

func getTokenExpiration() time.Time {
	return time.Now().Add(time.Duration(igor.Auth.TokenDuration) * time.Hour)
}
