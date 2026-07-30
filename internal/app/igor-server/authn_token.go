// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"igor2/internal/pkg/common"
	"igor2/internal/pkg/jwtkeys"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/hlog"
)

// TokenAuth implements IAuth interface
type TokenAuth struct{}

// NewTokenAuth instantiates the jwt.Token implementation of IAuth
func NewTokenAuth() IAuth {
	igor.AuthTokenKeypath = filepath.Join(igor.IgorHome, ".jwt", "private.pem")
	if err := verifyJwtKeyPair(); err != nil {
		logger.Error().Msgf("%v", err)
	}
	return &TokenAuth{}
}

// MyClaims carries custom configs for jwt.token
type MyClaims struct {
	Username string `json:"username"`
	// Igor groups the user is a part of for use by downstream applications
	Groups []string `json:"groups"`
	jwt.RegisteredClaims
}

func (l *TokenAuth) authenticate(r *http.Request) (*User, error) {
	clog := hlog.FromRequest(r)
	actionPrefix := "verify token"

	// extract token
	tokenString, err := extractToken(r)
	if err != nil {
		errLine := actionPrefix + " failed - error extracting token: " + err.Error()
		clog.Warn().Msgf("%v", errLine)
		return nil, err
	}
	if tokenString == "" {
		errLine := actionPrefix + " failed - no token"
		clog.Warn().Msg(errLine)
		return nil, &BadCredentialsError{msg: errLine}
	}

	token, parseErr := jwt.ParseWithClaims(
		tokenString,
		&MyClaims{},
		acquireTokenPublicKey,
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
	)

	if token == nil || parseErr != nil {
		// no token was recovered from header, user never included one
		errLine := actionPrefix + " failed - "
		if parseErr != nil {
			clog.Warn().Msgf("%v", parseErr)
			errLine += parseErr.Error()
		} else {
			errLine += "no token found"
		}
		return nil, &BadCredentialsError{msg: errLine}
	}

	claims, ok := token.Claims.(*MyClaims)
	if !ok {
		// expired or invalid token
		errLine := actionPrefix + " failed - expired or invalid token"
		clog.Warn().Msg(errLine)
		return nil, &BadCredentialsError{msg: errLine}
	}

	// verify Igor knows the user
	user, err := findUserForAuthN(claims.Username)
	if err != nil {
		clog.Warn().Msgf("%s failed - %v", actionPrefix, err)
		return nil, err
	}

	return user, nil
}

func acquireTokenPublicKey(token *jwt.Token) (interface{}, error) {
	// Reject algorithms other than the one used by Igor to sign tokens.
	method, ok := token.Method.(*jwt.SigningMethodRSA)
	if !ok || method.Alg() != jwt.SigningMethodRS256.Alg() {
		err := fmt.Errorf("validate token failed - unexpected signing method: %v", token.Header["alg"])
		logger.Warn().Msgf("%v", err)
		return nil, err
	}
	return jwtkeys.LoadPublic(jwtPublicKeyPath())
}

func generateToken(username string, groups []string, exprTime time.Time) (tokenString string, err error) {

	// set token expiration
	claims := &MyClaims{
		Username: username,
		Groups:   groups,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "igor",
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Minute)),
			ExpiresAt: jwt.NewNumericDate(exprTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// Create the JWT string
	key, err := jwtkeys.LoadPrivate(igor.AuthTokenKeypath)
	if err != nil {
		return "", err
	}
	tokenString, err = token.SignedString(key)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ExtractToken extracts a jwt.token string from Bearer Header
func extractToken(r *http.Request) (string, error) {
	token := ""
	bearToken := r.Header.Get(common.Authorization)
	// token may be in the auth header: bearer
	strArr := strings.Split(bearToken, " ")
	if len(strArr) == 2 {
		if strings.ToLower(strArr[0]) == "bearer" {
			token = strArr[1]
		}
	}
	if token != "" && token != "undefined" {
		return token, nil
	}

	// token may be in the cookie
	c, err := r.Cookie("auth_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			// no cookie present
			return "", nil
		}
		// something went wrong, return a bad request status
		return "", err
	}
	token = c.Value
	return token, nil
}

// return the path of the public key
func jwtPublicKeyPath() string {
	return filepath.Join(filepath.Dir(igor.AuthTokenKeypath), "public.pem")
}

// ensure storage exists or create a new asymmetric signing key pair
func verifyJwtKeyPair() error {
	return jwtkeys.Ensure(igor.AuthTokenKeypath, jwtPublicKeyPath())
}

// delete the keypair during reset token
func deleteJwtKeyPair() error {
	return jwtkeys.Delete(igor.AuthTokenKeypath, jwtPublicKeyPath())
}

// handleResetToken allows admin to generate a new JWT signing key pair
// and invalidates all existing authentication tokens.
func handleResetToken(w http.ResponseWriter, r *http.Request) {
	clog := hlog.FromRequest(r)
	actionPrefix := "reset JWT signing key pair"
	rb := common.NewResponseBody()
	var status int
	var err error

	// delete the signing key pair
	err = deleteJwtKeyPair()

	if err == nil {
		// generate a new signing key pair
		err = verifyJwtKeyPair()
	}

	if err != nil {
		rb.Message = err.Error()
		status = http.StatusInternalServerError
		clog.Error().Msgf("%s error - %v", actionPrefix, err)
	} else {
		status = http.StatusOK
		rb.Data["result"] = "token signing key pair refreshed successfully"
		clog.Info().Msgf("%s success", actionPrefix)
	}

	makeJsonResponse(w, status, rb)

}
