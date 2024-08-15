// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net"
	"net/http"
	"strconv"

	"igor2/internal/pkg/common"

	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/hlog"
)

// destination for route GET /hosts
func handleReadHosts(w http.ResponseWriter, r *http.Request) {
	// check to see if we're looking for a specific host
	queryMap := r.URL.Query()
	clog := hlog.FromRequest(r)
	actionPrefix := "read host(s)"
	rb := common.NewResponseBodyHosts()
	var hostList []Host
	var filterPowered *bool

	queryParams, status, err := parseHostSearchParams(queryMap, r)
	if err == nil {
		hostList, status, err = doReadHosts(queryParams)
		if len(hostList) > 0 {
			if powered, ok := queryMap["powered"]; ok {
				tmpPwrFilter, _ := strconv.ParseBool(powered[0])
				filterPowered = &tmpPwrFilter
			}
		}
	}

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		var hostDetails []common.HostData
		if len(hostList) == 0 {
			rb.Message = "search returned no results"
		} else {
			refreshPowerChan <- struct{}{}
			hostDetails = filterHostList(hostList, filterPowered, getUserFromContext(r))
		}
		rb.Data["hosts"] = hostDetails
	}

	makeJsonResponse(w, status, rb)
}

// destination for route PATCH /hosts/:hostName
func handleUpdateHost(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	editParams := getBodyFromContext(r)
	clog := hlog.FromRequest(r)
	actionPrefix := "update host"

	ps := httprouter.ParamsFromContext(r.Context())
	name := ps.ByName("hostName")
	rb := common.NewResponseBody()

	changes, status, err := parseHostEditParams(editParams, clog)
	if err == nil {
		status, err = doUpdateHost(name, changes, r)
	}

	if err != nil {
		if status < http.StatusBadRequest {
			msg := fmt.Sprintf("'%s' updated but problem writing new igor-clusters.yaml : %v", name, err)
			clog.Warn().Msgf("%s success - %s", actionPrefix, msg)
			rb.Message = msg
			status = http.StatusOK
		} else {
			rb.Message = err.Error()
			if status < http.StatusInternalServerError {
				clog.Warn().Msgf("%s failed - %v", actionPrefix, err)
			} else {
				clog.Error().Msgf("%s error - %v", actionPrefix, err)
			}
		}
	} else {
		clog.Info().Msgf("%s success - '%s' updated", actionPrefix, name)
	}
	makeJsonResponse(w, status, rb)
}

// destination for route DELETE /hosts/:hostName
func handleDeleteHosts(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	ps := httprouter.ParamsFromContext(r.Context())
	name := ps.ByName("hostName")
	clog := hlog.FromRequest(r)
	actionPrefix := "delete host"
	rb := common.NewResponseBody()

	status, err := doDeleteHost(name, r)

	if err != nil {
		if status < http.StatusBadRequest {
			msg := fmt.Sprintf("'%s' deleted but problem writing new igor-clusters.yaml : %v", name, err)
			clog.Warn().Msgf("%s success - %s", actionPrefix, msg)
			rb.Message = msg
			status = http.StatusOK
		} else {
			rb.Message = err.Error()
			if status < http.StatusInternalServerError {
				clog.Warn().Msgf("%s failed - %v", actionPrefix, err)
			} else {
				clog.Error().Msgf("%s error - %v", actionPrefix, err)
			}
		}
	} else {
		clog.Info().Msgf("%s success - '%s' deleted", actionPrefix, name)
	}
	makeJsonResponse(w, status, rb)
}

func validateHostParams(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var validateErr error
		clog := hlog.FromRequest(r)

		if r.Method == http.MethodGet {
			queryParams := r.URL.Query()
		queryParamLoop:
			for key, vals := range queryParams {
				switch key {
				case "eth":
					for _, val := range vals {
						if validateErr = checkEthRules(val); validateErr != nil {
							break queryParamLoop
						}
					}
				case "hostname":
					for _, val := range vals {
						if validateErr = checkGenericNameRules(val); validateErr != nil {
							break queryParamLoop
						}
					}
				case "name":
					for _, val := range vals {
						if strings.TrimSpace(val) != "" {
							names := igor.splitRange(val)
							if len(names) == 0 {
								validateErr = fmt.Errorf("couldn't parse node specification %v", val)
								break queryParamLoop
							}
						}
					}
				case "hostPolicy":
					for _, val := range vals {
						if validateErr = checkHostpolicyNameRules(val); validateErr != nil {
							break queryParamLoop
						}
					}
				case "ip":
					for _, val := range vals {
						if ip := net.ParseIP(val); ip == nil {
							validateErr = NewBadParamTypeError(key, val, "valid IPv4/6 string")
							break queryParamLoop
						}
					}
				case "mac":
					for _, val := range vals {
						if _, err := net.ParseMAC(val); err != nil {
							validateErr = NewBadParamTypeError(key, val, "invalid MAC address")
							break queryParamLoop
						}
					}
				case "reservation":
					for _, val := range vals {
						if validateErr = checkGenericNameRules(val); validateErr != nil {
							break queryParamLoop
						}
					}
				case "state":
					for _, val := range vals {
						if resolveHostState(val) == HostInvalid {
							validateErr = fmt.Errorf("host state term '%s' not accepted", val)
							break queryParamLoop
						}
					}
				case "powered":
					if len(vals) > 1 {
						validateErr = fmt.Errorf("invalid parameter: '%s' cannot have multiple values", key)
						break queryParamLoop
					}
					if _, err := strconv.ParseBool(vals[0]); err != nil {
						validateErr = fmt.Errorf("invalid parameter: '%s=%s' does not evaluate to boolean", key, vals[0])
						break queryParamLoop
					}
				default:
					validateErr = NewUnknownParamError(key, vals)
					break queryParamLoop
				}
			}
		}

		if r.Method == http.MethodPatch {

			hostParams := getBodyFromContext(r)

			if hostParams != nil {
			patchParamLoop:
				for key, val := range hostParams {
					switch key {
					case "eth":
						if _, ok := val.(string); !ok {
							validateErr = NewBadParamTypeError(key, val, "string")
							break patchParamLoop
						} else if validateErr = checkEthRules(val.(string)); validateErr != nil {
							break patchParamLoop
						}
					case "hostPolicy":
						if _, ok := val.(string); !ok {
							validateErr = NewBadParamTypeError(key, val, "string")
							break patchParamLoop
						} else if validateErr = checkHostpolicyNameRules(val.(string)); validateErr != nil {
							break patchParamLoop
						}
					case "hostname":
						if _, ok := val.(string); !ok {
							validateErr = NewBadParamTypeError(key, val, "string")
							break patchParamLoop
						} else if validateErr = checkGenericNameRules(val.(string)); validateErr != nil {
							break patchParamLoop
						}
					case "ip":
						if ipStr, ok := val.(string); !ok {
							validateErr = NewBadParamTypeError(key, val, "string")
							break patchParamLoop
						} else if ip := net.ParseIP(ipStr); ip == nil {
							validateErr = NewBadParamTypeError(key, val, "valid IPv4/6 string")
							break patchParamLoop
						}
					case "boot":
						if _, ok := val.(string); !ok {
							validateErr = NewBadParamTypeError(key, val, "string")
							break patchParamLoop
						}
						is_valid := false
						for _, v := range AllowedBootModes {
							if strings.ToLower(val.(string)) == v {
								is_valid = true
							}
						}
						if !is_valid {
							validateErr = fmt.Errorf("invalid boot type given")
							break patchParamLoop
						}
					case "mac":
						if mac, ok := val.(string); !ok {
							validateErr = NewBadParamTypeError(key, val, "string")
							break patchParamLoop
						} else {
							_, err := net.ParseMAC(mac)
							if err != nil {
								validateErr = fmt.Errorf("invalid mac address: %v given as search parameter", mac)
								break patchParamLoop
							}
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
			clog.Warn().Msgf("validateHostParams - %v", validateErr)
			createValidationErrMessage(validateErr, w)
			return
		}

		handler.ServeHTTP(w, r)

	})
}

func handlePowerHosts(w http.ResponseWriter, r *http.Request) {

	powerParams := getBodyFromContext(r)
	clog := hlog.FromRequest(r)
	cmd, hostList, status, err := checkPowerParams(powerParams, r)
	actionPrefix := "power " + cmd + " host(s)"
	if err == nil {
		status, err = doPowerHosts(cmd, hostList, clog)
	}

	rb := common.NewResponseBody()
	rb.Data["hosts"] = hostList
	if err != nil {
		clog.Error().Msgf("%s error - %v", actionPrefix, err)
		rb.Message = err.Error()
	} else {
		clog.Info().Msgf("%s success", actionPrefix)
	}

	makeJsonResponse(w, status, rb)
}

func validatePowerParams(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var validateErr error
		clog := hlog.FromRequest(r)

		hostParams := getBodyFromContext(r)

		if len(hostParams) > 0 {
			_, h := hostParams["hosts"]
			_, r := hostParams["resName"]
			_, c := hostParams["cmd"]
			if !h && !r {
				validateErr = fmt.Errorf("missing required param (hosts or resName) to issue power command")
			} else if h && r {
				validateErr = fmt.Errorf("both hosts and resName found (only 1 allowed)")
			} else if !c {
				validateErr = fmt.Errorf("missing power command")
			} else {

			patchParamLoop:
				for key, val := range hostParams {
					switch key {
					case "hosts":
						if _, ok := val.(string); !ok {
							validateErr = NewBadParamTypeError(key, val, "string")
							break patchParamLoop
						}
					case "resName":
						if rn, ok := val.(string); !ok {
							validateErr = NewBadParamTypeError(key, val, "string")
							break patchParamLoop
						} else if validateErr = checkGenericNameRules(rn); validateErr != nil {
							break patchParamLoop
						}
					case "cmd":
						if c, ok := val.(string); !ok {
							validateErr = NewBadParamTypeError(key, val, "string")
							break patchParamLoop
						} else if validateErr = checkPowerCmdSyntax(c); validateErr != nil {
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

		if validateErr != nil {
			clog.Warn().Msgf("validatePowerParams - %v", validateErr)
			createValidationErrMessage(validateErr, w)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

func handleBlockHosts(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	powerParams := getBodyFromContext(r)
	clog := hlog.FromRequest(r)
	actionPrefix := "block host(s)"
	block, hostList, status, err := checkBlockParams(powerParams)
	if !block {
		actionPrefix = "unblock host(s)"
	}
	if err == nil {
		status, err = doUpdateBlockHosts(block, hostList, r)
	}

	rb := common.NewResponseBody()
	rb.Data["hosts"] = hostList
	if err != nil {
		clog.Error().Msgf("%s error - %v", actionPrefix, err)
		rb.Message = err.Error()
	} else {
		clog.Info().Msgf("%s success [%v]", actionPrefix, strings.Join(hostList, ","))
	}

	makeJsonResponse(w, status, rb)
}

func validateBlockParams(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var validateErr error
		clog := hlog.FromRequest(r)

		hostParams := getBodyFromContext(r)

		if len(hostParams) > 0 {
			_, h := hostParams["hosts"]
			_, b := hostParams["block"]
			if !h {
				validateErr = fmt.Errorf("missing required hosts parameter")
			} else if !b {
				validateErr = fmt.Errorf("missing required block parameter")
			} else {

			patchParamLoop:
				for key, val := range hostParams {
					switch key {
					case "hosts":
						if _, ok := val.(string); !ok {
							validateErr = NewBadParamTypeError(key, val, "string")
							break patchParamLoop
						}
					case "block":
						if _, ok := val.(bool); !ok {
							validateErr = NewBadParamTypeError(key, val, "bool")
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

		if validateErr != nil {
			clog.Warn().Msgf("validatePowerParams - %v", validateErr)
			createValidationErrMessage(validateErr, w)
			return
		}
		handler.ServeHTTP(w, r)
	})
}
