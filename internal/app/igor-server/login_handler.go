// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"igor2/internal/pkg/common"

	"github.com/rs/zerolog/hlog"
)

func loginGetHandler(w http.ResponseWriter, r *http.Request) {

	if _, err := doPasswordAuth(w, r); err != nil {
		return
	}

	makeJsonResponse(w, http.StatusOK, common.NewResponseBody())
}

// loginPostHandler processes a POST /login request. See doPasswordAuth for possible
// error return codes.
func loginPostHandler(w http.ResponseWriter, r *http.Request) {

	if _, err := doPasswordAuth(w, r); err != nil {
		// return because all the errors have been dealt with,
		// so now it's back up the handler chain
		return
	}

	// if this is the CLI client (no login page) then this
	// was a redirect. Now we have to redirect again so
	// we can re-try the original command
	if strings.HasPrefix(r.UserAgent(), IgorCliPrefix) {
		rUrl := r.Header.Get(common.Referer)
		oUrl := r.Header.Get(common.Origin)
		if rUrl != "" && rUrl != oUrl+"/" {
			http.Redirect(w, r, rUrl, http.StatusTemporaryRedirect)
			return
		}
	}

	makeJsonResponse(w, http.StatusOK, common.NewResponseBody())
}

// doPasswordAuth authenticates the request using Basic authorization. If successful
// it will generate a new auth token cookie attached to the response writer.
//
// On error will return:
//
//	400/BadRequest if the Basic auth header couldn't be parsed or the username or password were blank.
//	401/Unauthorized if authentication failed.
//	500/ServerError if something unexpected happened.
func doPasswordAuth(w http.ResponseWriter, r *http.Request) (user *User, err error) {

	clog := hlog.FromRequest(r)
	actionPrefix := "user authentication"
	rb := common.NewResponseBody()

	username, password, baOK := r.BasicAuth()
	if !baOK {
		errLine := actionPrefix + " " + username + " " + password + " failed - problem reading basic auth header"
		clog.Warn().Msgf(errLine)
		rb.Message = errLine
		makeJsonResponse(w, http.StatusBadRequest, rb)
		return
	}

	// enforce basic rules of these params
	username = strings.TrimSpace(strings.ToLower(username))
	password = strings.TrimSpace(password)

	// also check if either username or password are empty strings
	// if so, redirect to login or Bad request
	if username == "" || password == "" {
		// "<script>alert('Please login')</script>"
		errLine := actionPrefix + " failed - username or password are blank"
		clog.Warn().Msgf(errLine)
		rb.Message = errLine
		makeJsonResponse(w, http.StatusBadRequest, rb)
		return
	}

	// If the user is elevated at this time, remove them.
	igor.ElevateMap.Remove(username)

	// Local or Secondary Auth

	// first run secondary auth if present and user is NOT original admin
	if igor.AuthSecondary != nil && username != IgorAdmin {
		user, err = igor.AuthSecondary.authenticate(r)
	} else {
		user, err = igor.AuthBasic.authenticate(r)
	}

	if err != nil {
		errLine := err.Error()
		rb.Message = errLine
		var badCredentialsError *BadCredentialsError
		switch {
		case errors.As(err, &badCredentialsError):
			// authentication failed
			// at this point igor CLI came from the /login handler and the
			// user must have entered their username/password wrong. For igorweb
			// they will have been on the login page already. So both fail here.
			clog.Warn().Msgf(errLine)
			makeJsonResponse(w, http.StatusUnauthorized, rb)
			return
		default:
			clog.Error().Msgf(errLine)
			makeJsonResponse(w, http.StatusInternalServerError, rb)
			return
		}
	}

	if user == nil {
		// user.Name is empty (this should never happen)
		err = fmt.Errorf("%s failed - no user object returned", actionPrefix)
		panic(err)
	}

	// we have successfully logged in, token generation time!
	exprTime := getTokenExpiration()

	tokenString, gtErr := generateToken(user.Name, exprTime)
	if gtErr != nil {
		errLine := fmt.Sprintf("%s failed - %v", actionPrefix, gtErr)
		clog.Error().Msgf(errLine)
		makeJsonResponse(w, http.StatusInternalServerError, rb)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	})

	return
}
