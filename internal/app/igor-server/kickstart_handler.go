// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"

	"igor2/internal/pkg/common"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/hlog"
)

func handleRegisterKickstart(w http.ResponseWriter, r *http.Request) {
	clog := hlog.FromRequest(r)
	actionPrefix := "register kickstart"
	rb := common.NewResponseBody()

	ks, status, err := doRegisterKickstart(r)
	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		msg := fmt.Sprintf("kickstart file registered successfully as: %s", ks.Name)
		clog.Info().Msgf("%s success -%s", actionPrefix, msg)
		rb.Message = msg
	}

	makeJsonResponse(w, status, rb)
}

func handleReadKickstart(w http.ResponseWriter, r *http.Request) {
	queryMap := r.URL.Query()
	clog := hlog.FromRequest(r)
	actionPrefix := "read kickstart"
	rb := common.NewResponseBody()
	status := http.StatusInternalServerError
	kickstarts := []Kickstart{}

	searchParams, code, err := parseKSSearchParams(queryMap, r)
	if err != nil {
		status = code
	} else {
		kickstarts, status, err = doReadKickstart(searchParams)
		if status == http.StatusNotFound {
			status = http.StatusOK
		}
	}

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		if len(kickstarts) == 0 {
			rb.Message = "no kickstart file found"
		} else {
			rb.Data["kickstarts"] = filterKickstartList(kickstarts)
		}
	}

	makeJsonResponse(w, status, rb)
}

func handleUpdateKickstart(w http.ResponseWriter, r *http.Request) {
	clog := hlog.FromRequest(r)
	actionPrefix := "update kickstart"
	rb := common.NewResponseBody()

	ps := httprouter.ParamsFromContext(r.Context())
	ksName := ps.ByName("kickstartName")

	status, err := doUpdateKS(ksName, r)
	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		msg := fmt.Sprintf("kickstart file updated successfully: %s", ksName)
		clog.Info().Msgf("%s success -%s", actionPrefix, msg)
		rb.Message = msg
	}

	makeJsonResponse(w, status, rb)
}

func handleDeleteKickstart(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	ps := httprouter.ParamsFromContext(r.Context())
	ksName := ps.ByName("kickstartName")
	clog := hlog.FromRequest(r)
	actionPrefix := "delete kickstart file"
	rb := common.NewResponseBody()

	status, err := doDeleteKS(ksName, r)
	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Info().Msgf("%s success - '%s' deleted", actionPrefix, ksName)
	}

	makeJsonResponse(w, status, rb)
}

func validateKSParams(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var validateErr error
		clog := hlog.FromRequest(r)
		// should only need to parse form once
		if validateErr = r.ParseMultipartForm(MaxMemory); validateErr != nil {
			clog.Warn().Msgf("validateKickstartParams - %v", validateErr)
			createValidationErrMessage(validateErr, w)
			return
		}

		if r.Method == http.MethodGet {
			queryParams := r.URL.Query()
		queryParamLoop:
			for key, vals := range queryParams {
				switch key {
				case "name":
					for _, val := range vals {
						if validateErr = checkGenericNameRules(val); validateErr != nil {
							break queryParamLoop
						}
					}
				default:
					validateErr = NewUnknownParamError(key, vals)
					break queryParamLoop
				}
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
