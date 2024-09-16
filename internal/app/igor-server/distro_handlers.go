// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"strings"

	"igor2/internal/pkg/common"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/hlog"
)

func handleCreateDistro(w http.ResponseWriter, r *http.Request) {
	clog := hlog.FromRequest(r)
	actionPrefix := "create distro"
	rb := common.NewResponseBody()
	var distro *Distro
	var status int
	var err error

	distro, status, err = doCreateDistro(r)

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		rb.Data["distro"] = filterDistroList([]Distro{*distro})
		clog.Info().Msgf("%s success - '%s' created", actionPrefix, distro.Name)
	}

	makeJsonResponse(w, status, rb)

}

func handleReadDistro(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	clog := hlog.FromRequest(r)
	actionPrefix := "read distro(s)"
	rb := common.NewResponseBody()
	var distroInfo []Distro

	searchParams, status, err := parseDistroReadParams(queryParams)
	if err == nil && status != http.StatusNotFound {
		distroInfo, status, err = doReadDistros(searchParams, r)
	} else if status == http.StatusNotFound {
		status = http.StatusOK
	}

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		if len(distroInfo) == 0 {
			rb.Message = "search returned no results"
		} else {
			rb.Data["distros"] = filterDistroList(distroInfo)
		}
	}

	makeJsonResponse(w, status, rb)
}

func handleUpdateDistro(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	clog := hlog.FromRequest(r)
	actionPrefix := "update distro"
	rb := common.NewResponseBody()
	ps := httprouter.ParamsFromContext(r.Context())
	distroName := ps.ByName("distroName")

	var status int
	var err error
	var dList []Distro

	dList, status, err = getDistrosTx([]string{distroName})
	if err == nil {
		distro := dList[0]
		// execute update process
		status, err = doUpdateDistro(&distro, r)
	}

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Info().Msgf("%s success - '%s' updated", actionPrefix, distroName)
	}

	makeJsonResponse(w, status, rb)
}

func handleDeleteDistro(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	ps := httprouter.ParamsFromContext(r.Context())
	distroName := ps.ByName("distroName")
	clog := hlog.FromRequest(r)
	actionPrefix := "delete distro"
	rb := common.NewResponseBody()

	status, err := doDeleteDistro(distroName, r)
	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Info().Msgf("%s success - '%s' deleted", actionPrefix, distroName)
	}

	makeJsonResponse(w, status, rb)
}

func validateDistroParams(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var validateErr error
		clog := hlog.FromRequest(r)

		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			// should only need to parse form once
			if validateErr = r.ParseMultipartForm(MaxMemory); validateErr != nil {
				clog.Warn().Msgf("validateDistroParams - %v", validateErr)
				createValidationErrMessage(validateErr, w)
				handler.ServeHTTP(w, r)
				return
			}
			distroParams := r.PostForm
			if len(distroParams) > 0 {
				name := r.FormValue("name")
				copyDistro := r.FormValue("copyDistro")
				useDistroImage := r.FormValue("useDistroImage")
				imageRef := r.FormValue("imageRef")
				if name == "" {
					validateErr = NewMissingParamError("name")
				} else if copyDistro == "" && useDistroImage == "" && imageRef == "" && (len(r.MultipartForm.File) < 1) {
					validateErr = fmt.Errorf("a new distro must have ONE of the following: existing distro, existing image, image ref, or kernel file AND initrd file")
				} else {

				postPutParamLoop:
					for key, val := range distroParams {
						switch key {
						case "name":
							if validateErr = checkDistroNameRules(val[0]); validateErr != nil {
								break postPutParamLoop
							}
						case "description":
							if validateErr = checkDesc(val[0]); validateErr != nil {
								break postPutParamLoop
							}
						case "distroGroups":
							for _, group := range val {
								if validateErr = checkGroupNameRules(group); validateErr != nil {
									break postPutParamLoop
								}
							}
						case "copyDistro":
							if validateErr = checkDistroNameRules(val[0]); validateErr != nil {
								break postPutParamLoop
							}
						case "useDistroImage":
							if validateErr = checkDistroNameRules(val[0]); validateErr != nil {
								break postPutParamLoop
							}
						case "imageRef":
							if validateErr = checkDistroImageRefRules(val[0]); validateErr != nil {
								break postPutParamLoop
							}
						case "public":
							public := strings.ToLower(val[0])
							if public != "true" {
								validateErr = fmt.Errorf("'%s' is not an acceptable value for public parameter (must be 'true')", val[0])
								break postPutParamLoop
							}
						case "default":
							makeDefault := strings.ToLower(val[0])
							if makeDefault != "true" {
								validateErr = fmt.Errorf("'%s' is not an acceptable value for parameter \"default\" (must be 'true')", val[0])
								break postPutParamLoop
							}
						case "kernelArgs":
							// already a valid string
							continue
						case "kickstart":
							if validateErr = checkFileRules(val[0]); validateErr != nil {
								break postPutParamLoop
							}
						case "boot":
							for _, v := range val {
								isValid := false
								for _, v2 := range AllowedBootModes {
									if strings.ToLower(v) == v2 {
										isValid = true
									}
								}
								if !isValid {
									validateErr = fmt.Errorf("invalid boot type given")
									break postPutParamLoop
								}
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
					case "name":
						for _, profileName := range vals {
							profileName = strings.TrimSpace(profileName)
							if validateErr = checkDistroNameRules(profileName); validateErr != nil {
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
					case "group":
						for _, groupName := range vals {
							groupName = strings.TrimSpace(groupName)
							if validateErr = checkGroupNameRules(groupName); validateErr != nil {
								break queryParamLoop
							}
						}
					case "imageID":
						for _, imageID := range vals {
							imageID = strings.TrimSpace(imageID)
							if validateErr = checkDistroImageIDRules(imageID); validateErr != nil {
								break queryParamLoop
							}
						}
					case "kernel", "initrd":
						for _, filenm := range vals {
							filenm = strings.TrimSpace(filenm)
							if validateErr = checkFileRules(filenm); validateErr != nil {
								break queryParamLoop
							}
						}
					case "default":
						if vals[0] != "true" {
							validateErr = fmt.Errorf("default flag must be true")
							break queryParamLoop
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
			// should only need to parse form once
			if validateErr = r.ParseMultipartForm(MaxMemory); validateErr != nil {
				clog.Warn().Msgf("validateDistroParams - %v", validateErr)
				createValidationErrMessage(validateErr, w)
				handler.ServeHTTP(w, r)
				return
			}
			distroParams := r.PostForm
			if len(distroParams) > 0 {
			patchParamLoop:
				for key, vals := range distroParams {
					switch key {
					case "name":
						for _, profileName := range vals {
							profileName = strings.TrimSpace(profileName)
							if validateErr = checkDistroNameRules(profileName); validateErr != nil {
								break patchParamLoop
							}
						}
					case "description":
						if validateErr = checkDesc(vals[0]); validateErr != nil {
							break patchParamLoop
						}
					case "owner":
						for _, ownerName := range vals {
							ownerName = strings.TrimSpace(ownerName)
							if validateErr = checkUsernameRules(ownerName); validateErr != nil {
								break patchParamLoop
							}
						}
					case "addGroup":
						for _, group := range vals {
							if validateErr = checkGroupNameRules(group); validateErr != nil {
								break patchParamLoop
							}
						}
					case "removeGroup":
						for _, group := range vals {
							if validateErr = checkGroupNameRules(group); validateErr != nil {
								break patchParamLoop
							}
						}
					case "public":
						public := strings.ToLower(vals[0])
						if public != "true" {
							validateErr = fmt.Errorf("'%s' is not an acceptable value for public parameter (must be 'true')", vals[0])
							break patchParamLoop
						}
					case "default":
						makeDefault := strings.ToLower(vals[0])
						if makeDefault != "true" {
							validateErr = fmt.Errorf("'%s' is not an acceptable value for parameter \"default\" (must be 'true')", vals[0])
							break patchParamLoop
						}
					case "default_remove":
						makeDefault := strings.ToLower(vals[0])
						if makeDefault != "true" {
							validateErr = fmt.Errorf("'%s' is not an acceptable value for parameter \"default_remove\" (must be 'true')", vals[0])
							break patchParamLoop
						}
					case "kernelArgs":
						// already a valid string
						continue
					case "kickstart":
						if validateErr = checkGenericNameRules(vals[0]); validateErr != nil {
							break patchParamLoop
						}
					default:
						validateErr = NewUnknownParamError(key, vals)
						break patchParamLoop
					}
				}
			} else {
				validateErr = NewMissingParamError("")
			}
		}

		if validateErr != nil {
			clog.Warn().Msgf("validateDistroParams - %v", validateErr)
			createValidationErrMessage(validateErr, w)
		}

		handler.ServeHTTP(w, r)
	})
}
