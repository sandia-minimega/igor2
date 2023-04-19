// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/hlog"

	"igor2/internal/pkg/common"

	"github.com/julienschmidt/httprouter"
)

// JSONMessage carries response metadata
type JSONMessage struct {
	Message string `json:"message"`
}

func handleCreateReservations(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	createParams := getBodyFromContext(r)
	clog := hlog.FromRequest(r)
	actionPrefix := "create reservation"
	rb := common.NewResponseBody()

	res, resIsNow, status, err := doCreateReservation(createParams, r)
	dbAccess.Unlock()

	if err == nil && resIsNow {
		now := time.Now()
		mrErr := manageReservations(&now, installReservations)
		if mrErr != nil {
			clog.Error().Msgf("%v", mrErr)
		}
	}

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		rb.Data["reservation"] = filterReservationList([]Reservation{*res}, getUserFromContext(r))
		clog.Info().Msgf("%s success - '%s' created", actionPrefix, res.Name)
	}

	makeJsonResponse(w, status, rb)
}

func handleReadReservations(w http.ResponseWriter, r *http.Request) {
	queryMap := r.URL.Query()
	clog := hlog.FromRequest(r)
	actionPrefix := "read reservation(s)"
	rb := common.NewResponseBody()
	var resvs []Reservation

	// parse queryMap and convert []string vals to proper corresponding types
	queryParams, timeParams, status, err := parseResSearchParams(queryMap, r)
	if err == nil {
		resvs, status, err = doReadReservations(queryParams, timeParams)
	}

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		rb.Data["reservations"] = filterReservationList(resvs, getUserFromContext(r))
		if len(resvs) == 0 {
			rb.Message = "search returned no results"
		}
	}

	makeJsonResponse(w, status, rb)
}

func handleUpdateReservation(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	editParams := getBodyFromContext(r)
	clog := hlog.FromRequest(r)
	actionPrefix := "update reservation"
	ps := httprouter.ParamsFromContext(r.Context())
	resName := ps.ByName("resName")
	rb := common.NewResponseBody()

	status, err := doUpdateReservation(resName, editParams, r)

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Info().Msgf("%s success - '%s' updated", actionPrefix, resName)
	}

	makeJsonResponse(w, status, rb)
}

func handleDeleteReservations(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	ps := httprouter.ParamsFromContext(r.Context())
	resName := ps.ByName("resName")
	clog := hlog.FromRequest(r)
	actionPrefix := "delete reservation"
	rb := common.NewResponseBody()

	status, err := doDeleteReservation(resName, r)

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Info().Msgf("%s success - '%s' deleted", actionPrefix, resName)
	}

	makeJsonResponse(w, status, rb)
}

func validateResvParams(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var validateErr error
		clog := hlog.FromRequest(r)

		if r.Method == http.MethodPost || r.Method == http.MethodPut {

			resParams := getBodyFromContext(r)

			if len(resParams) > 0 {
				_, nl := resParams["nodeList"]
				_, nc := resParams["nodeCount"]
				_, name := resParams["name"]
				_, profile := resParams["profile"]
				_, distro := resParams["distro"]
				if !name {
					validateErr = fmt.Errorf("missing reservation name (required)")
				} else if !nl && !nc {
					validateErr = fmt.Errorf("missing nodeList or nodeCount; one required to create reservation")
				} else if nl && nc {
					validateErr = fmt.Errorf("both nodeList and nodeCount found; only one allowed")
				} else if !distro && !profile {
					validateErr = fmt.Errorf("missing profile or distro; one required to create reservation")
				} else if distro && profile {
					validateErr = fmt.Errorf("both profile and distro found; only one allowed")
				} else {

				postPutParamLoop:
					for key, val := range resParams {
						switch strings.TrimSpace(key) {
						case "name":
							if resName, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else if validateErr = checkGenericNameRules(resName); validateErr != nil {
								break postPutParamLoop
							}
						case "description":
							if d, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else if validateErr = checkDesc(d); validateErr != nil {
								break postPutParamLoop
							}
						case "distro":
							if distroName, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else if validateErr = checkDistroNameRules(distroName); validateErr != nil {
								break postPutParamLoop
							}
						case "owner":
							if owner, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else if validateErr = checkUsernameRules(owner); validateErr != nil {
								break postPutParamLoop
							}
						case "profile":
							if profileName, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else if validateErr = checkProfileNameRules(profileName); validateErr != nil {
								break postPutParamLoop
							}
						case "group":
							if grName, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else if validateErr = checkGroupNameRules(grName); validateErr != nil {
								break postPutParamLoop
							} else if grName == GroupAll {
								validateErr = fmt.Errorf("reservations cannot be assigned to the 'all' group")
								break postPutParamLoop
							}
						case "noCycle":
							if _, ok := val.(bool); !ok {
								validateErr = NewBadParamTypeError(key, val, "bool")
								break postPutParamLoop
							}
						case "vlan":
							if _, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							}
						case "nodeList":
							if thisNodeList, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else {
								if strings.TrimSpace(thisNodeList) != "" {
									hostNames := igor.splitRange(thisNodeList)
									if len(hostNames) == 0 {
										validateErr = fmt.Errorf("couldn't parse node specification %v", thisNodeList)
										break postPutParamLoop
									}
								} else {
									validateErr = fmt.Errorf("at least 1 host name required to create reservation")
									break postPutParamLoop
								}
							}
						case "nodeCount":
							if _, ok := resParams["nodeCount"].(float64); !ok {
								validateErr = NewBadParamTypeError(key, val, "float64")
								break postPutParamLoop
							}
						case "duration":
							sDur, sOk := val.(string)
							_, fOk := val.(float64)
							if !sOk && !fOk {
								validateErr = NewBadParamTypeError(key, val, "string | float64")
								break postPutParamLoop
							} else if sOk {
								dur, err := common.ParseDuration(sDur)
								if err != nil {
									validateErr = fmt.Errorf("'%s' is not a recognized duration interval", sDur)
									break postPutParamLoop
								}
								if dur <= 0 {
									validateErr = fmt.Errorf("duration expression '%s' cannot be a negative value", sDur)
								}
							}
						case "start":
							if _, ok := val.(float64); !ok {
								validateErr = NewBadParamTypeError(key, val, "float64")
								break postPutParamLoop
							}
						case "kernelArgs":
							_, ok := val.(string)
							if !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
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
					for _, resvName := range vals {
						resvName = strings.TrimSpace(resvName)
						if validateErr = checkGenericNameRules(resvName); validateErr != nil {
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
				case "distro":
					for _, distroName := range vals {
						distroName = strings.TrimSpace(distroName)
						if validateErr = checkDistroNameRules(distroName); validateErr != nil {
							break queryParamLoop
						}
					}
				case "profile":
					for _, profileName := range vals {
						profileName = strings.TrimSpace(profileName)
						if validateErr = checkProfileNameRules(profileName); validateErr != nil {
							break queryParamLoop
						}
					}
				// case "from-start", "from-end", "to-start", "to-end":
				// 	if err := common.ValidateTimeFormat(vals[0]); err != nil {
				// 		validateErr = fmt.Errorf("parameter '%s' is not a recognized time format, found %s", key, vals[0])
				// 		break queryParamLoop
				// 	}
				// case "duration", "gte-duration", "lte-duration":
				// 	if _, err := time.ParseDuration(vals[0]); err != nil {
				// 		validateErr = fmt.Errorf("parameter '%s' is not a recognized duration format, found %s", key, vals[0])
				// 		break queryParamLoop
				// 	}
				// case "installed":
				// 	if vals[0] != "0" && vals[0] != "1" {
				// 		validateErr = fmt.Errorf("parameter 'installed' must be 0 or 1, found %s", vals[0])
				// 		break queryParamLoop
				// 	}
				// case "vlan", "gte-extendNum", "lte-extendNum", "eq-extendNum", "gte-nodeCount", "lte-nodeCount", "nodeCount":
				// 	for _, extNumStr := range vals {
				// 		if _, err := strconv.Atoi(extNumStr); err != nil {
				// 			validateErr = fmt.Errorf("parameter '%s' is not a number, found %s", key, extNumStr)
				// 			break queryParamLoop
				// 		}
				// 	}
				default:
					validateErr = NewUnknownParamError(key, vals)
					break queryParamLoop
				}
			}
		}

		if r.Method == http.MethodPatch {
			resParams := getBodyFromContext(r)

			if len(resParams) > 0 {
				_, doExtend := resParams["extend"]
				_, doExtendMax := resParams["extendMax"]
				_, doDistro := resParams["distro"]
				_, doProfile := resParams["profile"]
				_, doDrop := resParams["drop"]
				// if doing an extend command, it must be the only thing updating
				if doExtend || doExtendMax {
					if len(resParams) != 1 {
						validateErr = fmt.Errorf("extending a reservation can only be a singluar edit; found %v", resParams)
					} else if doExtend {
						sDur, sOk := resParams["extend"].(string)
						_, fOk := resParams["extend"].(float64)
						if !sOk && !fOk {
							validateErr = NewBadParamTypeError("extend", resParams["extend"], "string | float64")
						} else if sOk {
							dur, err := common.ParseDuration(sDur)
							if err != nil {
								validateErr = fmt.Errorf("'%s' is not a recognized duration interval", sDur)
							}
							if dur <= 0 {
								validateErr = fmt.Errorf("duration expression '%s' cannot be a negative value", sDur)
							}
						}
					}
				} else if doDrop {
					if len(resParams) != 1 {
						validateErr = fmt.Errorf("dropping nodes from a reservation can only be a singluar edit; found %v", resParams)
					} else {
						if thisNodeList, ok := resParams["drop"].(string); !ok {
							validateErr = NewBadParamTypeError("drop", resParams["drop"], "string")
						} else {
							if strings.TrimSpace(thisNodeList) != "" {
								hostNames := igor.splitRange(thisNodeList)
								if len(hostNames) == 0 {
									validateErr = fmt.Errorf("couldn't parse node specification %v", thisNodeList)
								}
							} else {
								validateErr = fmt.Errorf("at least 1 host name required to create reservation")
							}
						}
					}
				} else if doDistro || doProfile {
					if len(resParams) == 1 && (doDistro || doProfile) {
						for key, val := range resParams {
							switch key {
							case "distro":
								if distro, ok := val.(string); !ok {
									validateErr = NewBadParamTypeError(key, val, "string")
									break
								} else if validateErr = checkDistroNameRules(distro); validateErr != nil {
									break
								}
							case "profile":
								if profile, ok := val.(string); !ok {
									validateErr = NewBadParamTypeError(key, val, "string")
									break
								} else if validateErr = checkProfileNameRules(profile); validateErr != nil {
									break
								}
							default:
								validateErr = NewUnknownParamError(key, val)
								break
							}
						}
					} else if len(resParams) == 2 && doDistro && doProfile {
						validateErr = fmt.Errorf("both profile and distro params found; only one allowed")
					} else {
						validateErr = fmt.Errorf("distro and profile changes cannot be mixed with other reservation changes; found %v", resParams)
					}
				} else {
				patchParamLoop:
					for key, val := range resParams {
						switch key {
						case "name":
							if name, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break patchParamLoop
							} else if validateErr = checkGenericNameRules(name); validateErr != nil {
								break patchParamLoop
							}
						case "description":
							if desc, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break patchParamLoop
							} else if validateErr = checkDesc(desc); validateErr != nil {
								break patchParamLoop
							}
						case "owner":
							if owner, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break patchParamLoop
							} else if validateErr = checkUsernameRules(owner); validateErr != nil {
								break patchParamLoop
							}
						case "group":
							groupName, ok := val.(string)
							if !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break patchParamLoop
							} else if groupName == GroupNoneAlias {
								continue
							} else if groupName == GroupAll {
								validateErr = fmt.Errorf("reservations cannot be assigned to the 'all' group")
								break patchParamLoop
							} else if validateErr = checkGroupNameRules(groupName); validateErr != nil {
								break patchParamLoop
							}
						case "kernelArgs":
							_, ok := val.(string)
							if !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
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

		if validateErr != nil {
			clog.Warn().Msgf("validateResvParams - %v", validateErr)
			createValidationErrMessage(validateErr, w)
			return
		}

		handler.ServeHTTP(w, r)
	})
}
