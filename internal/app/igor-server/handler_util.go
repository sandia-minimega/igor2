// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"

	"igor2/internal/pkg/api"
	"igor2/internal/pkg/common"
)

// Regex for simple names. Includes letters, numbers, underscore, dash, and dot. Must be 3-24 characters in
// length. No whitespace allowed.
var stdNameCheckPattern = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9._-]{2,23}$`)

// Regex for Eth names. Includes letters, numbers, slash. Must be 3-24 characters in
// length. No whitespace allowed.
// Interface naming convention: https://www.cisco.com/assets/sol/sb/Switches_Emulators_v2_3_5_xx/help/350_550/index.html#page/tesla_350_550_olh/ts_getting_started_01_22.html
var stdEthCheckPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9\/]{2,23}$`)

// Regex for distro Image ref (Image name). Consists of a prefix (image type), followed by 8 characters which can be a combination
// of letters and numbers. Must be 10 characters in length total. No whitespace allowed. Example: kid942c59b
var stdImageRefCheckPattern = regexp.MustCompile(`^(ki|iso)[a-zA-Z0-9]{8}$`)

// Regex for distro Image Hash. Consists of characters which can be a combination
// of letters and numbers. Must be 80 characters in length total. No whitespace allowed.
// Example: d942c59b7cd38145330b421d11a52ac6936a0bab019cc8b26ab37834ee48272285f8bff5ede05f8a
var stdImageIDCheckPattern = regexp.MustCompile(`^[a-zA-Z0-9]{40}$`)

// Regex for description fields. Includes letters, numbers, limited punctuation and space character. Max 256
// characters in length.
var descCheckPattern = regexp.MustCompile(`^[a-zA-Z0-9 :,)(.?!_-]{0,256}$`)

// Regex for file names. Cannot start or end with spaces. May have a .ext included at the end, or not.
var fileNameCheckPattern = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9 ._-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9_-])?$`)

// checkGenericNameRules looks for unwanted patterns in names for various resources. Allows only characters specified
// in the stdNameCheckPattern but cannot be all digits or all punctuation.
func checkGenericNameRules(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("name cannot be empty")
	}

	isNotDigitOrPunc := func(c rune) bool { return (c < '0' || c > '9') && c != '.' && c != '_' && c != '-' }
	digitsPuncOnly := strings.IndexFunc(name, isNotDigitOrPunc) == -1
	if !stdNameCheckPattern.MatchString(name) || digitsPuncOnly {
		return fmt.Errorf("'%s' is not a legal name. (Must be 3-24 chars, start with letter, number or underscore. Can contain dash and dot. Cannot be all digits or all punctuation.)", name)
	}
	return isResourceNameMatch(name)
}

// isResourceNameMatch throws an error if the input value matches a word that could cause a problem with
// a permissions check or operation. These words should not be used as the name of a given resource.
func isResourceNameMatch(value string) error {
	switch value {
	case PermGroups, PermUsers, PermClusters, PermDistros, PermHosts, PermProfiles, PermReservations,
		"hostPolicy", "group", "user", "cluster", "distro", "host", "profile", "reservation":
		return fmt.Errorf("name cannot be restricted word '%s'", value)
	default:
		return nil
	}
}

// A very simple panic handler
func panicHandler(w http.ResponseWriter, r *http.Request, err interface{}) {
	logger.Panic().Stack().Msgf("panic intercepted - %v: %v\n%v", r.URL.Path, err, string(debug.Stack()))
	rb := common.NewResponseBody()
	rb.Message = fmt.Sprintf("server error; please notify admins : %v", err)
	makeJsonResponse(w, http.StatusInternalServerError, rb)
}

// Standard description field checker. Returns error if string has illegal characters or is longer than max length (256).
// Includes letters, numbers, space and limited punctuation:
//
//	.,_-():?!
//
// Whitespace is trimmed from ends. Can be empty.
func checkDesc(desc string) error {
	if !descCheckPattern.MatchString(strings.TrimSpace(desc)) {
		return fmt.Errorf("description field invalid, must be 0-256 characters and may only contain letters, numbers, space and .,_-():?! characters")
	}
	return nil
}

func createValidationErrMessage(validateErr error, w http.ResponseWriter) {
	rb := common.NewResponseBody()
	rb.Message = validateErr.Error()
	makeJsonResponse(w, http.StatusBadRequest, rb)
}

// marshalJSONBody turns the intended body response into a JSON string. It will panic if
// it is unable to do so, indicating a programming issue with how the body response has been
// constructed.
func marshalJSONBody(v interface{}) []byte {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return jsonBytes
}

// Handler that parses the MIME type of the request body and either return a 400
// if the type can't be determined or a 415 if the type is not 'application/json'. The check
// is only applied to request types that are allowed to have content (PUT,POST, and PATCH). Other
// request types pass though.
func checkContentType(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut || r.Method == http.MethodPost || r.Method == http.MethodPatch {
			// won't have content - jump to next if
			ct := r.Header.Get(common.ContentType)
			cl := r.Header.Get(common.ContentLength)
			if ct != "" || cl != "0" {
				mt, _, err := mime.ParseMediaType(ct)
				if err != nil {
					logger.Error().Msgf("malformed content-type: %v", err)
					rb := common.NewResponseBody()
					rb.Message = err.Error()
					makeJsonResponse(w, http.StatusBadRequest, rb)
					return
				}
				if strings.HasPrefix(r.URL.Path, api.Distros) || strings.HasPrefix(r.URL.Path, api.Images) || strings.HasPrefix(r.URL.Path, api.Kickstarts) {
					if (r.Method == http.MethodPost || r.Method == http.MethodPatch) && mt != common.MFormData {
						errMsg := fmt.Sprintf("need content-type '%s', but got '%s'", common.MFormData, ct)
						logger.Error().Msg(errMsg)
						rb := common.NewResponseBody()
						rb.Message = errMsg
						makeJsonResponse(w, http.StatusUnsupportedMediaType, rb)
						return
					}
				} else {
					if mt != common.MAppJson {
						if !strings.HasPrefix(r.URL.Path, api.Login) {
							errMsg := fmt.Sprintf("need content-type '%s', but got '%s'", common.MAppJson, ct)
							logger.Error().Msg(errMsg)
							rb := common.NewResponseBody()
							rb.Message = errMsg
							makeJsonResponse(w, http.StatusUnsupportedMediaType, rb)
							return
						}

					}
				}
			}
		}
		if handler != nil {
			handler.ServeHTTP(w, r)
		}
	})
}

// storeJSONBodyHandler extracts the body of an incoming request, unmarshals it from
// JSON into a map[string]interface{} and stores the map in the context that is forwarded
// to each subsequent handler. It can be accessed by calling:
//
//	myParamMap := getBodyFromContext(r)
//
// It will panic if the body encounters a read error (which shouldn't happen).
//
// If an InvalidUnmarshalError is encountered it will log an error and return 400 Bad Request.
func storeJSONBodyHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var mapBody map[string]interface{}
		if body, readErr := io.ReadAll(r.Body); readErr == nil {
			if len(body) > 0 {
				err := json.Unmarshal(body, &mapBody)
				if err != nil {
					errMsg := fmt.Sprintf("JSON unmarshal error: %v", err)
					logger.Error().Msg(errMsg)
					rb := common.NewResponseBody()
					rb.Message = err.Error()
					makeJsonResponse(w, http.StatusBadRequest, rb)
					return
				}
			}
			rCopy := addBodyToContext(r, mapBody)
			handler.ServeHTTP(w, rCopy)
		} else {
			panic(readErr)
		}
	})
}

// makeJsonResponse performs the proper order of actions to set fields in the
// ResponseWriter when attaching JSON content.
func makeJsonResponse(w http.ResponseWriter, status int, rb common.ResponseBody) {
	rb.SetStatus(status)
	w.Header().Set(common.ContentType, common.MAppJson)
	w.WriteHeader(status)
	jsonBytes := marshalJSONBody(rb)
	if _, err := w.Write(jsonBytes); err != nil {
		panic(err)
	}
}

type jsonBodyKey struct{}

func addBodyToContext(r *http.Request, mapBody map[string]interface{}) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, jsonBodyKey{}, mapBody)
	return r.WithContext(ctx)
}

func getBodyFromContext(r *http.Request) map[string]interface{} {
	mapBody, _ := r.Context().Value(jsonBodyKey{}).(map[string]interface{})
	return mapBody
}

type userContextKey struct{}

func addUserToContext(r *http.Request, user *User) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, userContextKey{}, user)
	return r.WithContext(ctx)
}

func getUserFromContext(r *http.Request) *User {
	user, _ := r.Context().Value(userContextKey{}).(*User)
	return user

}
