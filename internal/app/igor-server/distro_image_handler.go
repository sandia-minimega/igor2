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

func handleRegisterDistroImage(w http.ResponseWriter, r *http.Request) {
	clog := hlog.FromRequest(r)
	actionPrefix := "register boot image"
	rb := common.NewResponseBody()

	image, status, err := doRegisterImage(r)
	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		rb.Data["image"] = image
		msg := fmt.Sprintf("igor boot image files registered to refID: %s", image.Name)
		clog.Info().Msgf("%s success -%s", actionPrefix, msg)
		rb.Message = msg
	}

	makeJsonResponse(w, status, rb)
}

func handleReadDistroImage(w http.ResponseWriter, r *http.Request) {
	clog := hlog.FromRequest(r)
	actionPrefix := "read distro images"
	rb := common.NewResponseBody()

	distroImages, status, err := doReadDistroImages()
	if status == http.StatusNotFound {
		status = http.StatusOK
	}

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		if len(distroImages) == 0 {
			rb.Message = "search returned no results"
		} else {
			rb.Data["distroImages"] = filterDistroImagesList(distroImages)
		}
	}

	makeJsonResponse(w, status, rb)
}

func handleDeleteDistroImage(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	ps := httprouter.ParamsFromContext(r.Context())
	distroImageName := ps.ByName("imageName")
	clog := hlog.FromRequest(r)
	actionPrefix := "delete distro image"
	rb := common.NewResponseBody()

	status, err := doDeleteDistroImage(distroImageName, r)
	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Info().Msgf("%s success - '%s' deleted", actionPrefix, distroImageName)
	}

	makeJsonResponse(w, status, rb)
}

func validateDistroImageParams(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var validateErr error
		clog := hlog.FromRequest(r)

		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			// should only need to parse form once
			if validateErr = r.ParseMultipartForm(MaxMemory); validateErr != nil {
				clog.Warn().Msgf("validateDistroImageParams - %v", validateErr)
				createValidationErrMessage(validateErr, w)
				return
			}
			diParams := r.PostForm
			if len(diParams) > 0 {
			postPutParamLoop:
				for key, val := range diParams {
					switch key {
					case "kstaged", "istaged":
						if validateErr = checkFileRules(val[0]); validateErr != nil {
							break postPutParamLoop
						}
					case "localBoot":
						if len(val) > 0 && strings.ToLower(val[0]) != "true" {
							validateErr = fmt.Errorf("invalid value for localBoot, must be 'true'")
							break postPutParamLoop
						}
					case "breed":
						if validateErr = checkGenericNameRules(val[0]); validateErr != nil {
							break postPutParamLoop
						}
					case "boot":
						for _, v := range val {
							is_valid := false
							for _, v2 := range AllowedBootModes {
								if strings.ToLower(v) == v2 {
									is_valid = true
								}
							}
							if !is_valid {
								validateErr = fmt.Errorf("invalid boot type given")
								break postPutParamLoop
							}
						}
					default:
						validateErr = NewUnknownParamError(key, val)
						break postPutParamLoop
					}
				}
			} else {
				validateErr = NewMissingParamError("")
			}
		}

		if validateErr != nil {
			clog.Warn().Msgf("validateDistroImageParams - %v", validateErr)
			createValidationErrMessage(validateErr, w)
			return
		}

		handler.ServeHTTP(w, r)
	})
}
