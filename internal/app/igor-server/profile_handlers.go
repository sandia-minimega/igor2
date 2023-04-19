// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"net/http"
	"strings"

	"igor2/internal/pkg/common"

	"github.com/rs/zerolog/hlog"

	"github.com/julienschmidt/httprouter"
)

// destination for route POST /profiles
func handleCreateProfile(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	createParams := getBodyFromContext(r)
	clog := hlog.FromRequest(r)
	actionPrefix := "create profile"
	rb := common.NewResponseBody()

	profile, status, err := doCreateProfile(createParams, r)

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		rb.Data["profile"] = filterProfileList([]Profile{*profile})
		clog.Info().Msgf("%s success - '%s' created", actionPrefix, profile.Name)
	}

	makeJsonResponse(w, status, rb)
}

// destination for route GET /profiles
func handleReadProfiles(w http.ResponseWriter, r *http.Request) {
	queryMap := r.URL.Query()
	clog := hlog.FromRequest(r)
	actionPrefix := "read profile(s)"
	rb := common.NewResponseBody()
	var profiles []Profile

	queryParams, status, err := parseProfileSearchParams(queryMap, r)
	if err == nil {
		profiles, status, err = doReadProfiles(queryParams)
	}

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		rb.Data["profiles"] = filterProfileList(profiles)
		if len(profiles) == 0 {
			rb.Message = "search returned no results"
		}
	}

	makeJsonResponse(w, status, rb)
}

// destination for route PATCH /profiles/:profileName
func handleUpdateProfile(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	editParams := getBodyFromContext(r)
	clog := hlog.FromRequest(r)
	actionPrefix := "update profile"
	rb := common.NewResponseBody()

	ps := httprouter.ParamsFromContext(r.Context())
	profileName := ps.ByName("profileName")

	status, err := doUpdateProfile(profileName, editParams, r)

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Info().Msgf("%s success - '%s' updated", actionPrefix, profileName)
	}

	makeJsonResponse(w, status, rb)
}

// destination for route DELETE /profiles/:profileName
func handleDeleteProfile(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	ps := httprouter.ParamsFromContext(r.Context())
	profileName := ps.ByName("profileName")
	clog := hlog.FromRequest(r)
	actionPrefix := "delete profile"
	rb := common.NewResponseBody()

	status, err := doDeleteProfile(profileName)

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Info().Msgf("%s success - '%s' deleted", actionPrefix, profileName)
	}

	makeJsonResponse(w, status, rb)
}

func validateProfileParams(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var validateErr error
		clog := hlog.FromRequest(r)

		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			profileParams := getBodyFromContext(r)
			var ok bool

			if profileParams != nil {
				if _, ok = profileParams["name"]; !ok {
					validateErr = NewMissingParamError("name")
				} else if _, ok = profileParams["distro"]; !ok {
					validateErr = NewMissingParamError("distro")
				} else {

				postPutParamLoop:
					for key, val := range profileParams {
						switch key {
						case "kernelArgs":
							if _, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							}
						case "name":
							if profileName, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else if validateErr = checkProfileNameRules(profileName); validateErr != nil {
								break postPutParamLoop
							} else if validateErr = checkReservedProfileNames(profileName); validateErr != nil {
								break postPutParamLoop
							}
						case "description":
							if desc, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else if validateErr = checkDesc(desc); validateErr != nil {
								break postPutParamLoop
							}
						case "distro":
							if distro, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else if validateErr = checkDistroNameRules(distro); validateErr != nil {
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

		if r.Method == http.MethodGet {
			queryParams := r.URL.Query()
			if queryParams != nil {
			queryParamLoop:
				for key, vals := range queryParams {
					switch key {
					case "kernelArgs":
						continue
					case "name":
						for _, profileName := range vals {
							profileName = strings.TrimSpace(profileName)
							if validateErr = checkProfileNameRules(profileName); validateErr != nil {
								break queryParamLoop
							}
						}
					case "owner":
						for _, ownerName := range vals {
							ownerName = strings.TrimSpace(ownerName)
							if validateErr = checkUsernameRules(ownerName); validateErr != nil {
								break queryParamLoop
							}
						}
					case "distro":
						for _, distroName := range vals {
							distroName = strings.TrimSpace(distroName)
							if validateErr = checkDistroNameRules(distroName); validateErr != nil {
								break queryParamLoop
							}
						}
					default:
						validateErr = NewUnknownParamError(key, vals)
						break queryParamLoop
					}
				}
			} else {
				validateErr = NewMissingParamError("")
			}
		}

		if r.Method == http.MethodPatch {
			profileParams := getBodyFromContext(r)

		patchParamLoop:
			for key, val := range profileParams {
				switch key {
				case "kernelArgs":
					if _, ok := val.(string); !ok {
						validateErr = NewBadParamTypeError(key, val, "string")
						break patchParamLoop
					}
				case "description":
					if desc, ok := val.(string); !ok {
						validateErr = NewBadParamTypeError(key, val, "string")
						break patchParamLoop
					} else if validateErr = checkDesc(desc); validateErr != nil {
						break patchParamLoop
					}
				case "name":
					if name, ok := val.(string); !ok {
						validateErr = NewBadParamTypeError(key, val, "string")
						break patchParamLoop
					} else if validateErr = checkProfileNameRules(name); validateErr != nil {
						break patchParamLoop
					} else if validateErr = checkReservedProfileNames(name); validateErr != nil {
						break patchParamLoop
					}
				default:
					validateErr = NewUnknownParamError(key, val)
					break patchParamLoop
				}
			}
		}

		if validateErr != nil {
			clog.Warn().Msgf("validateProfileParams - %v", validateErr)
			createValidationErrMessage(validateErr, w)
			return
		}

		handler.ServeHTTP(w, r)
	})
}
