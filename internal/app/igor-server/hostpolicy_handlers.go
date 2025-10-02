// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"igor2/internal/pkg/common"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/hlog"
)

// destination for route POST /hosts
func handleCreateHostPolicy(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	createParams := getBodyFromContext(r)
	clog := hlog.FromRequest(r)
	actionPrefix := "create host policy"
	rb := common.NewResponseBody()

	hostPolicy, status, err := doCreateHostPolicy(createParams, r)

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		rb.Data["hostPolicy"] = filterHostPoliciesList([]HostPolicy{*hostPolicy})
		clog.Info().Msgf("%s success - '%s' created", actionPrefix, hostPolicy.Name)
	}

	makeJsonResponse(w, status, rb)

}

// destination for route GET /hosts
func handleReadHostPolicies(w http.ResponseWriter, r *http.Request) {
	// check to see if we're looking for a specific host
	queryMap := r.URL.Query()
	clog := hlog.FromRequest(r)
	actionPrefix := "read host policies"
	rb := common.NewResponseBody()
	var hostPolicies []HostPolicy

	clog.Debug().Msgf("handleReadHostPolicies params received: %v", queryMap)
	// parse queryMap and convert []string vals to proper corresponding types
	queryParams, status, err := parseHostPolicySearchParams(queryMap, r)
	if err == nil {
		hostPolicies, status, err = doReadHostPolicies(queryParams, r)
	}

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		if len(hostPolicies) == 0 {
			rb.Message = "search returned no results"
		} else {
			rb.Data["hostPolicies"] = filterHostPoliciesList(hostPolicies)
		}
	}

	makeJsonResponse(w, status, rb)
}

// destination for route PATCH /hosts/:hostName
func handleUpdateHostPolicy(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	editParams := getBodyFromContext(r)
	clog := hlog.FromRequest(r)
	actionPrefix := "update host policy"

	ps := httprouter.ParamsFromContext(r.Context())
	name := ps.ByName("hostpolicyName")

	changes, status, err := parseHostPolicyEditParams(editParams, clog)
	if err == nil {
		status, err = doUpdateHostPolicy(name, changes, r)
	}
	rb := common.NewResponseBody()

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Info().Msgf("%s success - '%s' updated", actionPrefix, name)
	}
	makeJsonResponse(w, status, rb)
}

// destination for route DELETE /hosts/:hostName
func handleDeleteHostPolicy(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	ps := httprouter.ParamsFromContext(r.Context())
	name := ps.ByName("hostpolicyName")
	clog := hlog.FromRequest(r)
	actionPrefix := "delete host policy"
	rb := common.NewResponseBody()

	status, err := doDeleteHostPolicy(name, r)

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Info().Msgf("%s success - '%s' deleted", actionPrefix, name)
	}

	makeJsonResponse(w, status, rb)
}

func validateHostPolicyParams(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var validateErr error
		clog := hlog.FromRequest(r)

		if r.Method == http.MethodPost || r.Method == http.MethodPut {

			hostPolicyParams := getBodyFromContext(r)

			if hostPolicyParams != nil {

				_, name := hostPolicyParams["name"]
				if !name {
					validateErr = fmt.Errorf("missing host policy name (required)")
				} else {
				postPutParamLoop:
					for key, val := range hostPolicyParams {
						switch key {
						case "name":
							if _, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else if validateErr = checkHostPolicyNameRules(val.(string)); validateErr != nil {
								break postPutParamLoop
							}
						case "maxResTime":
							if dur, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else {
								duration, err := common.ParseDuration(dur)
								if err != nil {
									validateErr = err
								} else if duration <= 0 {
									validateErr = fmt.Errorf("duration expression '%s' cannot be a negative value", dur)
									break postPutParamLoop
								}
							}
						case "accessGroups":
							grNames, ok := val.([]interface{})
							if !ok {
								// return internal error instead?
								validateErr = NewBadParamTypeError(key, val, "[string]interface")
								break postPutParamLoop
							}
							for _, val := range grNames {
								if name, ok := val.(string); !ok {
									validateErr = NewBadParamTypeError(key, val, "[string]interface")
									break postPutParamLoop
								} else {
									if validateErr = checkGroupNameRules(name); validateErr != nil {
										break postPutParamLoop
									}
								}
							}
						case "notAvailable":
							validateErr = validateScheduleBlockParams(key, val)
							if validateErr != nil {
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
		queryParamLoop:
			for key, vals := range queryParams {
				switch key {
				case "name":
					for _, val := range vals {
						if validateErr = checkHostPolicyNameRules(val); validateErr != nil {
							break queryParamLoop
						}
					}
				case "accessGroups":
					for _, val := range vals {
						if validateErr = checkGroupNameRules(val); validateErr != nil {
							break queryParamLoop
						}

					}
				case "hosts":
					// TODO: we don't currently have a way to check a host name
					continue
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
					case "name":
						if name, ok := val.(string); !ok {
							validateErr = NewBadParamTypeError(key, val, "string")
							break patchParamLoop
						} else if validateErr = checkGenericNameRules(name); validateErr != nil {
							break patchParamLoop
						}
					case "maxResTime":
						if dur, ok := val.(string); !ok {
							validateErr = NewBadParamTypeError(key, val, "string")
							break patchParamLoop
						} else {
							duration, err := common.ParseDuration(dur)
							if err != nil {
								validateErr = err
							} else if duration <= 0 {
								validateErr = fmt.Errorf("duration expression '%s' cannot be a negative value", dur)
								break patchParamLoop
							}
						}
					case "addGroups", "removeGroups":
						grNames, ok := val.([]interface{})
						if !ok {
							// return internal error instead?
							validateErr = NewBadParamTypeError(key, val, "string array")
							break patchParamLoop
						}
						for _, val := range grNames {
							if name, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string array")
								break patchParamLoop
							} else {
								if validateErr = checkGroupNameRules(name); validateErr != nil {
									break patchParamLoop
								}
							}
						}
					case "addNotAvailable", "removeNotAvailable":
						validateErr = validateScheduleBlockParams(key, val)
						if validateErr != nil {
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
			reqUrl, _ := url.QueryUnescape(r.URL.RequestURI())
			clog.Warn().Msgf("validateHostPolicyParams - failed validation for %s:%s:%v - %v", getUserFromContext(r).Name, r.Method, reqUrl, validateErr)
			createValidationErrMessage(validateErr, w)
			return
		}

		handler.ServeHTTP(w, r)

	})
}

func validateScheduleBlockParams(key string, val interface{}) error {
	sbList, ok := val.([]interface{})
	if !ok {
		return NewBadParamTypeError(key, val, "[]interface{}")
	}
	for _, val := range sbList {
		if sbInstance, ok := val.(map[string]interface{}); !ok {
			return NewBadParamTypeError(key, val, "[string]interface{}")
		} else {
			if val, ok := sbInstance["start"].(string); !ok {
				return NewBadParamTypeError(key, val, "string")
			} else if _, err := parseSBInstance(val); err != nil {
				return err
			}
			if val, ok := sbInstance["duration"].(string); !ok {
				return NewBadParamTypeError(key, val, "string")
			} else {
				duration, err := common.ParseDuration(val)
				if err != nil {
					return err
				} else if duration <= 0 {
					return fmt.Errorf("duration expression '%s' cannot be a negative value", val)
				}
			}
		}
	}
	return nil
}

func handleApplyPolicy(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	applyParams := getBodyFromContext(r)
	clog := hlog.FromRequest(r)
	actionPrefix := "apply policy"
	policy, hosts, status, err := checkApplyPolicyParams(applyParams, clog)
	if err == nil {
		status, err = doApplyPolicy(policy, hosts)
	}

	rb := common.NewResponseBody()
	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Info().Msgf("%s success", actionPrefix)
	}

	makeJsonResponse(w, status, rb)
}

func validateApplyPolicyParams(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var validateErr error
		clog := hlog.FromRequest(r)

		hostParams := getBodyFromContext(r)

		if len(hostParams) > 0 {
			_, h := hostParams["nodeList"]
			_, b := hostParams["policy"]
			if !h {
				validateErr = fmt.Errorf("missing required hosts parameter")
			} else if !b {
				validateErr = fmt.Errorf("missing required policy name parameter")
			} else {

			patchParamLoop:
				for key, val := range hostParams {
					switch key {
					case "nodeList":
						if thisNodeList, ok := val.(string); !ok {
							validateErr = NewBadParamTypeError(key, val, "string")
							break patchParamLoop
						} else {
							if strings.TrimSpace(thisNodeList) != "" {
								hostNames := igor.splitRange(thisNodeList)
								if len(hostNames) == 0 {
									validateErr = fmt.Errorf("couldn't parse node specification %v", thisNodeList)
									break patchParamLoop
								}
							} else {
								validateErr = fmt.Errorf("at least 1 host name required to create reservation")
								break patchParamLoop
							}
						}
					case "policy":
						if _, ok := val.(string); !ok {
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
