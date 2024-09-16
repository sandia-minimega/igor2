// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"strings"

	"igor2/internal/pkg/common"

	"github.com/rs/zerolog/hlog"

	"github.com/julienschmidt/httprouter"
)

// destination for route POST /users
func handleCreateUser(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	createParams := getBodyFromContext(r)
	clog := hlog.FromRequest(r)
	actionPrefix := "create user"
	rb := common.NewResponseBody()
	var status int

	if igor.Auth.Ldap.Sync.EnableUserSync {
		status = http.StatusBadRequest
		err := fmt.Errorf("cannot create local user when LDAP manages account creation")
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		if user, ucStatus, err := doCreateUser(createParams, r); err != nil {
			stdErrorResp(rb, ucStatus, actionPrefix, err, clog)
		} else {
			status = ucStatus
			msg := fmt.Sprintf("igor user '%s' created", user.Name)
			clog.Info().Msgf("%s success - %s", actionPrefix, msg)
			rb.Message = msg
		}
	}

	makeJsonResponse(w, status, rb)
}

// destination for route GET /users(?params)
func handleReadUsers(w http.ResponseWriter, r *http.Request) {
	queryMap := r.URL.Query()
	clog := hlog.FromRequest(r)
	actionUser := getUserFromContext(r)
	actionPrefix := "read user(s)"
	var users []User
	rb := common.NewResponseBodyUsers()

	queryParams, status, err := parseUserSearchParams(queryMap, r)
	if err == nil {
		users, status, err = doReadUsers(queryParams)
	}

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		var userDetails []common.UserData
		if len(users) == 0 {
			rb.Message = "search returned no results"
		} else {
			for _, u := range users {
				// don't return igor-admin as part of the normal user query list
				if !userElevated(actionUser.Name) && u.Name == IgorAdmin {
					continue
				}
				ud := u.getUserData(actionUser)
				userDetails = append(userDetails, *ud)
			}
		}
		rb.Data["users"] = userDetails
	}

	makeJsonResponse(w, status, rb)
}

// destination for PATCH /users/:username
func handleUpdateUser(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	editParams := getBodyFromContext(r)
	clog := hlog.FromRequest(r)
	actionPrefix := "update user"
	ps := httprouter.ParamsFromContext(r.Context())
	username := ps.ByName("userName") // the user we area altering
	rb := common.NewResponseBody()

	updateMsg, status, err := doUpdateUser(username, editParams, r)

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		msg := fmt.Sprintf("user '%s' %s", username, updateMsg)
		clog.Info().Msgf("%s success - %s", actionPrefix, msg)
		rb.Message = msg
	}
	makeJsonResponse(w, status, rb)
}

// destination for DELETE /users/:username
func handleDeleteUser(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	ps := httprouter.ParamsFromContext(r.Context())
	name := ps.ByName("userName")
	clog := hlog.FromRequest(r)
	actionPrefix := "delete user"
	rb := common.NewResponseBody()

	status, err := doDeleteUser(name, r)

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		msg := fmt.Sprintf("user '%s' deleted", name)
		clog.Info().Msgf("%s success - %s", actionPrefix, msg)
		rb.Message = msg
	}
	makeJsonResponse(w, status, rb)
}

// validateUserParams is a handler that performs syntax checking on either body or
// query parameters
func validateUserParams(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var validateErr error
		clog := hlog.FromRequest(r)

		// first check for user params in body
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			userParams := getBodyFromContext(r)
			var ok bool

			if userParams != nil {
				if _, ok = userParams["name"]; !ok {
					validateErr = NewMissingParamError("name")
				} else if _, ok = userParams["email"]; !ok {
					validateErr = NewMissingParamError("email")
				} else {

				postPutParamLoop:
					for key, val := range userParams {
						switch key {
						case "name":
							if user, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else if validateErr = checkUsernameRules(strings.ToLower(user)); validateErr != nil {
								break postPutParamLoop
							}
						case "fullName":
							if user, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else if validateErr = checkFullNameRules(strings.ToLower(user)); validateErr != nil {
								break postPutParamLoop
							}
						case "email":
							if email, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else if validateErr = checkEmailRules(email); validateErr != nil {
								break postPutParamLoop
							}
						default:
							validateErr = NewUnknownParamError(key, val)
							break postPutParamLoop
						}
					}
				}
			} else {
				validateErr = NewMissingParamError("")
			}
		}

		// PATCH only allows updating of email address or password
		if r.Method == http.MethodPatch {
			userParams := getBodyFromContext(r)

			if userParams != nil {
				_, npw := userParams["password"]
				_, opw := userParams["oldPassword"]
				_, bReset := userParams["reset"]
				_, bEmail := userParams["email"]
				_, bFullName := userParams["fullName"]
				if bReset && (npw || opw || bEmail || bFullName) {
					validateErr = fmt.Errorf("reset password cannot be executed with other user edits")
				} else if (bEmail || bFullName) && (opw || npw) {
					validateErr = fmt.Errorf("password changes must be done separately from other edits")
				} else if npw && !opw {
					validateErr = NewMissingParamError("oldPassword")
				} else if !npw && opw {
					validateErr = NewMissingParamError("password")
				} else {

				patchParamLoop:
					for key, val := range userParams {
						switch key {
						case "email":
							if email, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break patchParamLoop
							} else if validateErr = checkEmailRules(email); validateErr != nil {
								break patchParamLoop
							}
						case "fullName":
							if fullName, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break patchParamLoop
							} else if validateErr = checkFullNameRules(fullName); validateErr != nil {
								break patchParamLoop
							}
						case "password":
							if passwd, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break patchParamLoop
							} else if validateErr = checkLocalPasswordRules(passwd); validateErr != nil {
								break patchParamLoop
							}
						case "oldPassword":
							if _, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break patchParamLoop
							}
						case "reset":
							if reset, ok := val.(bool); !ok {
								validateErr = NewBadParamTypeError(key, val, "bool")
								break patchParamLoop
							} else if !reset {
								validateErr = fmt.Errorf("invalid parameter '%s': must be boolean=true to have effect", key)
								break patchParamLoop
							}
						default:
							validateErr = NewUnknownParamError(key, val)
							break patchParamLoop
						}
					}
				}
			} else {
				validateErr = NewMissingParamError("")
			}
		}

		// Now check for user params in query
		if r.Method == http.MethodGet {
			queryParams := r.URL.Query()

		queryParamLoop:
			for key, val := range queryParams {
				switch key {
				case "name":
					for _, name := range val {
						name = strings.TrimSpace(strings.ToLower(name))
						if validateErr = checkUsernameRules(name); validateErr != nil {
							break queryParamLoop
						}
					}
				default:
					validateErr = NewUnknownParamError(key, val)
					break queryParamLoop
				}
			}
		}

		if validateErr != nil {
			clog.Warn().Msgf("validateUserParams - %v", validateErr)
			createValidationErrMessage(validateErr, w)
			return
		}

		if handler != nil {
			handler.ServeHTTP(w, r)
		}
	})
}
