// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"igor2/internal/pkg/common"
	"net/http"
	"net/url"
	"strconv"

	"github.com/rs/zerolog/hlog"
)

// destination for route POST /clusters
func handleCreateClusters(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	clog := hlog.FromRequest(r)
	actionPrefix := "create cluster(s)"
	rb := common.NewResponseBody()

	clusters, hostnames, status, err := doCreateClusters(r)

	if status >= http.StatusInternalServerError {
		clog.Error().Msgf("%s error - %v", actionPrefix, err)
		rb.Message = err.Error()
	} else if status >= http.StatusBadRequest {
		clog.Warn().Msgf("%s failed - %v", actionPrefix, err)
		rb.Message = err.Error()
	} else {
		rb.Data["clusters"] = clusters
		msg := fmt.Sprintf("'%s' created with following hosts %v", clusters[0].Name, hostnames)
		if err != nil {
			msg += " - " + err.Error()
		}
		clog.Info().Msgf("%s success - %s", actionPrefix, msg)
		rb.Message = msg
	}

	makeJsonResponse(w, status, rb)
}

// destination for route GET /clusters
func handleReadClusters(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	queryMap := r.URL.Query()
	clog := hlog.FromRequest(r)
	actionPrefix := "read cluster(s)"
	rb := common.NewResponseBody()
	var queryParams map[string]interface{}
	var doFileDump bool
	var getYamlFile bool
	var yDoc []byte
	var finalPath string

	queryParams, doFileDump, getYamlFile = parseClusterSearchParams(queryMap, r)
	clusters, status, err := doReadClusters(queryParams)
	if err != nil {
		rb.Message = err.Error()
		clog.Error().Msgf("%s error - %v", actionPrefix, err)
	} else if doFileDump || getYamlFile {

		yDoc, err = assembleYamlOutput(clusters)
		if err != nil {
			status = http.StatusInternalServerError
			rb.Message = err.Error()
			clog.Error().Msgf("%s error - %v", actionPrefix, err)
		} else {

			if doFileDump {
				finalPath, err = updateClusterConfigFile(yDoc, clog)

				if err != nil {
					rb.Message = err.Error()
				} else {
					rb.Message = fmt.Sprintf("dumped cluster config to %s", finalPath)
				}
			}
		}

	} else {
		clog.Info().Msgf("%s success", actionPrefix)
	}

	if status < http.StatusBadRequest {
		if getYamlFile {
			if doFileDump {
				rb.Message = fmt.Sprintf("dumped cluster config to %s", finalPath)
			}
			rb.Data["yaml"] = string(yDoc)
		} else {
			rb.Data["clusters"] = clusters
		}
	}
	makeJsonResponse(w, status, rb)
}

func validateClusterParams(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var validateErr error
		clog := hlog.FromRequest(r)

		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			clusterParams := getBodyFromContext(r)
			if len(clusterParams) > 0 {
				for key, val := range clusterParams {
					validateErr = NewUnknownParamError(key, val)
					break
				}
			}
		}

		if r.Method == http.MethodGet {
			queryParams := r.URL.Query()
			if queryParams != nil {
			queryParamLoop:
				for key, vals := range queryParams {
					switch key {
					case "dump", "getYaml":
						if len(vals) > 1 {
							validateErr = fmt.Errorf("invalid parameter: '%s' cannot have multiple values", key)
							break queryParamLoop
						}
						if _, err := strconv.ParseBool(vals[0]); err != nil {
							validateErr = fmt.Errorf("invalid parameter: '%s=%s' does not evaluate to boolean", key, vals[0])
							break queryParamLoop
						}
					case "name", "prefix":
						continue
					default:
						validateErr = NewUnknownParamError(key, vals)
						break queryParamLoop
					}
				}
			}
		}

		if validateErr != nil {
			reqUrl, _ := url.QueryUnescape(r.URL.RequestURI())
			clog.Warn().Msgf("validateHostParams - failed validation for %s:%s:%v - %v", getUserFromContext(r).Name, r.Method, reqUrl, validateErr)
			createValidationErrMessage(validateErr, w)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

func handleUpdateMotd(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	createParams := getBodyFromContext(r)
	clog := hlog.FromRequest(r)
	actionPrefix := "update motd"
	rb := common.NewResponseBody()

	status, err := doUpdateMotd(createParams)

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Info().Msgf("%s success", actionPrefix)
	}

	makeJsonResponse(w, status, rb)
}

func validateMotdParams(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var validateErr error
		clog := hlog.FromRequest(r)

		if r.Method == http.MethodPatch {
			clusterParams := getBodyFromContext(r)
			var ok bool

			if clusterParams != nil {
				if _, ok = clusterParams["motd"]; !ok {
					validateErr = NewMissingParamError("motd")
				}
				if _, ok = clusterParams["motdUrgent"]; !ok {
					validateErr = NewMissingParamError("motdUrgent")
				}

			patchParamLoop:
				for key, val := range clusterParams {
					switch key {
					case "motd":
						// we just check that name is a string
						if _, ok = val.(string); !ok {
							validateErr = NewBadParamTypeError(key, val, "string")
							break patchParamLoop
						}
					case "motdUrgent":
						// we just check that name is a string
						if _, ok = val.(bool); !ok {
							validateErr = NewBadParamTypeError(key, val, "bool")
							break patchParamLoop
						}
					default:
						validateErr = NewUnknownParamError(key, val)
						break patchParamLoop
					}
				}
			} else {
				validateErr = NewMissingParamError("")
			}
		}

		if validateErr != nil {
			clog.Warn().Msgf("validateMotdParams - %v", validateErr)
			createValidationErrMessage(validateErr, w)
			return
		}

		handler.ServeHTTP(w, r)
	})
}
