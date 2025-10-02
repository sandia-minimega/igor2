// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"igor2/internal/pkg/common"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/hlog"
)

// destination for route POST /groups
func handleCreateGroup(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	createParams := getBodyFromContext(r)
	clog := hlog.FromRequest(r)
	actionPrefix := "create group"
	rb := common.NewResponseBody()

	group, status, addMsg, err := doCreateGroup(createParams, r)

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		msg := fmt.Sprintf("igor group '%s' created %s", group.Name, addMsg)
		clog.Info().Msgf("%s success - %s by user %s", actionPrefix, msg, getUserFromContext(r).Name)
		rb.Message = msg
	}

	makeJsonResponse(w, status, rb)
}

// destination for route GET /groups(?params)
func handleReadGroups(w http.ResponseWriter, r *http.Request) {

	queryMap := r.URL.Query()
	clog := hlog.FromRequest(r)
	actionPrefix := "read group(s)"
	rb := common.NewResponseBodyGroups()
	var groupList []Group

	queryParams, status, err := parseGroupSearchParams(queryMap, r)
	if err == nil {
		groupList, status, err = doReadGroups(queryParams)
	}

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {

		actionUser := getUserFromContext(r)
		groupList = getViewAccessibleGroups(actionUser, groupList)

		if len(groupList) == 0 {
			rb.Message = "search returned no results"
		} else {
			for _, g := range groupList {
				groupType := "member"
				for _, owner := range g.Owners {
					if owner.Name == actionUser.Name {
						groupType = "owner"
					}
				}
				rb.Data[groupType] = append(rb.Data[groupType], *g.getGroupData())
			}
		}
	}
	makeJsonResponse(w, status, rb)
}

// destination for PATCH /groups/:groupName
func handleUpdateGroup(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	editParams := getBodyFromContext(r)
	clog := hlog.FromRequest(r)
	actionPrefix := "update group"
	ps := httprouter.ParamsFromContext(r.Context())
	name := ps.ByName("groupName")
	rb := common.NewResponseBody()

	status, err := doUpdateGroup(name, editParams, r)

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Info().Msgf("%s success - '%s' updated by user %s", actionPrefix, name, getUserFromContext(r).Name)
	}

	makeJsonResponse(w, status, rb)
}

// destination for DELETE /groups/:groupName
func handleDeleteGroup(w http.ResponseWriter, r *http.Request) {

	dbAccess.Lock()
	defer dbAccess.Unlock()

	ps := httprouter.ParamsFromContext(r.Context())
	name := ps.ByName("groupName")
	clog := hlog.FromRequest(r)
	actionPrefix := "delete group"
	rb := common.NewResponseBody()

	status, err := doDeleteGroup(name, r)

	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Info().Msgf("%s success - '%s' deleted by user %s", actionPrefix, name, getUserFromContext(r).Name)
	}
	makeJsonResponse(w, status, rb)
}

func validateGroupParams(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var validateErr error
		clog := hlog.FromRequest(r)

		// first check for user params in body
		if r.Method == http.MethodPost || r.Method == http.MethodPut {

			groupParams := getBodyFromContext(r)
			var ok bool

			if len(groupParams) > 0 {
				_, ldap := groupParams["isLDAP"]
				_, members := groupParams["members"]
				_, owners := groupParams["owners"]
				_, desc := groupParams["description"]
				if _, ok = groupParams["name"]; !ok {
					validateErr = NewMissingParamError("name")
				} else if ldap && groupParams["isLDAP"].(bool) && (members || owners || desc) {
					validateErr = fmt.Errorf("group creation includes disallowed params when marked as LDAP")
				} else {

				postPutParamLoop:
					for key, val := range groupParams {
						switch key {
						case "name":
							// we just check that name is a string
							if n, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else if validateErr = checkGenericNameRules(n); validateErr != nil {
								break postPutParamLoop
							} else if validateErr = checkReservedGroupNames(n); validateErr != nil {
								break postPutParamLoop
							}
						case "isLDAP":
							if _, ok := val.(bool); !ok {
								validateErr = NewBadParamTypeError(key, val, "bool")
								break postPutParamLoop
							}
						case "owners":
							for _, v := range val.([]interface{}) {
								if m, ok := v.(string); !ok {
									validateErr = NewBadParamTypeError(key, val, "[]string")
								} else if validateErr = checkUsernameRules(m); validateErr != nil {
									break postPutParamLoop
								}
							}
						case "members":
							for _, v := range val.([]interface{}) {
								if m, ok := v.(string); !ok {
									validateErr = NewBadParamTypeError(key, val, "[]string")
								} else if validateErr = checkUsernameRules(m); validateErr != nil {
									break postPutParamLoop
								}
							}
						case "description":
							if d, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break postPutParamLoop
							} else if validateErr = checkDesc(d); validateErr != nil {
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

		if r.Method == http.MethodPatch {
			groupParams := getBodyFromContext(r)

			if len(groupParams) > 0 {
				_, addOwners := groupParams["addOwners"]
				_, rmvOwners := groupParams["rmvOwners"]
				_, add := groupParams["add"]
				_, remove := groupParams["remove"]

				if (add || remove) && (addOwners || rmvOwners) {
					validateErr = fmt.Errorf("operations on owners and members must be separate commands")
				} else {

				patchParamLoop:
					for key, val := range groupParams {
						switch key {
						case "name":
							// we just check that name is a string
							if name, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break patchParamLoop
							} else if validateErr = checkGenericNameRules(name); validateErr != nil {
								break patchParamLoop
							} else if validateErr = checkReservedGroupNames(name); validateErr != nil {
								break patchParamLoop
							}
						case "description":
							if desc, ok := val.(string); !ok {
								validateErr = NewBadParamTypeError(key, val, "string")
								break patchParamLoop
							} else if validateErr = checkDesc(desc); validateErr != nil {
								break patchParamLoop
							}
						case "addOwners", "rmvOwners":
							for _, v := range val.([]interface{}) {
								if _, ok := v.(string); !ok {
									validateErr = NewBadParamTypeError(key, val, "[]string")
								} else if validateErr = checkUsernameRules(v.(string)); validateErr != nil {
									break patchParamLoop
								}
							}
						case "add", "remove":
							// members must be a string array
							for _, v := range val.([]interface{}) {
								if _, ok := v.(string); !ok {
									validateErr = NewBadParamTypeError(key, val, "[]string")
								} else if validateErr = checkUsernameRules(v.(string)); validateErr != nil {
									break patchParamLoop
								}
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

		if r.Method == http.MethodGet {
			queryParams := r.URL.Query()

		queryParamLoop:
			for key, vals := range queryParams {
				switch key {
				case "name":
					for _, name := range vals {
						name = strings.TrimSpace(name)
						if validateErr = checkGroupNameRules(name); validateErr != nil {
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
				case "showMembers":
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

		if r.Method == http.MethodDelete {
			ps := httprouter.ParamsFromContext(r.Context())
			name := ps.ByName("groupName")
			validateErr = checkReservedGroupNames(name)
		}

		if validateErr != nil {
			reqUrl, _ := url.QueryUnescape(r.URL.RequestURI())
			clog.Warn().Msgf("validateGroupParams - failed validation for %s:%s:%v - %v", getUserFromContext(r).Name, r.Method, reqUrl, validateErr)
			createValidationErrMessage(validateErr, w)
			return
		}

		handler.ServeHTTP(w, r)
	})
}
