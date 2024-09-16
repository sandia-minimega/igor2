// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"igor2/internal/pkg/common"

	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog/hlog"
)

// TokenAuth implements IAuth interface
type TokenAuth struct{}

// NewTokenAuth instantiates the jwt.Token implementation of IAuth
func NewTokenAuth() IAuth {
	igor.AuthTokenKeypath = filepath.Join(igor.IgorHome, ".jwt", "jkey")
	if err := verifyJwtSecret(); err != nil {
		logger.Error().Msgf("%v", err)
	}
	return &TokenAuth{}
}

// MyClaims carries custom configs for jwt.token
type MyClaims struct {
	Username string `json:"username"`
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
		clog.Warn().Msgf(errLine)
		return nil, &BadCredentialsError{msg: errLine}
	}

	token, parseErr := jwt.ParseWithClaims(tokenString, &MyClaims{}, acquireTokenSecret)

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
		clog.Warn().Msgf(errLine)
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

func acquireTokenSecret(token *jwt.Token) (interface{}, error) {
	// make sure the signing method is the same as when we first generated the token
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		err := fmt.Errorf("validate token failed - Unexpected signing method: %v", token.Header["alg"])
		logger.Warn().Msgf(err.Error())
		return nil, err
	}
	return getJwtToken()
}

func generateToken(username string, exprTime time.Time) (tokenString string, err error) {

	// set token expiration
	claims := &MyClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exprTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Create the JWT string
	key, err := getJwtToken()
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

// ensure storage exists or create
func verifyJwtSecret() error {
	// ensure path exists
	storePath, _ := filepath.Split(igor.AuthTokenKeypath)
	// check key file exists
	if _, err := os.Stat(igor.AuthTokenKeypath); errors.Is(err, os.ErrNotExist) {
		// make sure the path exists
		createErr := os.MkdirAll(storePath, 0755)
		if createErr != nil {
			return createErr
		}
		// keep it secret, keep it safe
		var file, err = os.OpenFile(igor.AuthTokenKeypath, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			return err
		}
		defer file.Close()
		secret, _ := generateSecret()
		_, err = file.Write(secret)
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteJwtSecret() error {
	// check key file exists
	if _, err := os.Stat(igor.AuthTokenKeypath); !errors.Is(err, os.ErrNotExist) {
		// file appears to exist, let's fix that
		if err := os.Remove(igor.AuthTokenKeypath); err != nil {
			return err
		}
	}
	return nil
}

func getJwtToken() ([]byte, error) {
	return os.ReadFile(igor.AuthTokenKeypath)
}

func generateSecret() ([]byte, error) {
	// read 64 cryptographically secure pseudorandom numbers
	// from rand.Reader and write them to a byte slice
	c := 64
	b := make([]byte, c)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("error:", err)
		return nil, err
	}
	return b, nil
}

// handleResetToken allows admin to generate a new jwt secret key
// will invalidate all existing authn jwt Tokens
func handleResetToken(w http.ResponseWriter, r *http.Request) {
	clog := hlog.FromRequest(r)
	actionPrefix := "reset JWT secret"
	rb := common.NewResponseBody()
	var status int
	var err error

	// delete the secret file
	err = deleteJwtSecret()

	if err == nil {
		// generate a new secret
		err = verifyJwtSecret()
	}

	if err != nil {
		rb.Message = err.Error()
		status = http.StatusInternalServerError
		clog.Error().Msgf("%s error - %v", actionPrefix, err)
	} else {
		status = http.StatusOK
		rb.Data["result"] = "token secret refreshed successfully"
		clog.Info().Msgf("%s success", actionPrefix)
	}

	makeJsonResponse(w, status, rb)

}
