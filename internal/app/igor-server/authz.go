// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"igor2/internal/pkg/api"
	"net/http"
	"regexp"
	"strings"

	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"

	"igor2/internal/pkg/common"

	"github.com/julienschmidt/httprouter"
)

const (
	SysForbidDelete = "deletion not allowed"
)

var urlPartsMatcher = regexp.MustCompile(`^/(\w+)/?`)

func authzHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		rb := common.NewResponseBody()
		user := getUserFromContext(r)

		authInfo, err := user.getAuthzInfo()
		if err != nil {
			rb.Message = err.Error()
			makeJsonResponse(w, http.StatusInternalServerError, rb)
			return
		}

		// if the route is the base url ('igor show') the permission sought is 'view all reservations'
		var reqPermString string
		resource := PermReservations

		trimmedUrl := strings.TrimPrefix(r.URL.Path, api.BaseUrl)

		pathParts := urlPartsMatcher.FindStringSubmatch(trimmedUrl)
		if pathParts != nil {
			resource = pathParts[1]
		}

		// any authenticated user can reach this endpoint
		if r.URL.Path == api.PublicSettings {
			handler.ServeHTTP(w, r)
			return
		}

		// allowing a user to elevate is determined by handlers looking at the ElevateMap
		if r.URL.Path == api.Elevate {
			handler.ServeHTTP(w, r)
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == api.HostPolicy {
			handler.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == api.HostsBlock {
			// this perm won't match anything assigned to users so will fail, but will pass
			// the admin permission of '*'
			p, _ := NewPermission("host-block")
			if authInfo.IsPermitted(p) {
				handler.ServeHTTP(w, r)
			} else {
				rb.Message = "block/unblock hosts requires admin elevated privilege"
				makeJsonResponse(w, http.StatusForbidden, rb)
			}
			return
		}

		// allow view-restricted resources to pass if method is GET
		// these are filtered in the backend before results are returned
		if r.Method == http.MethodGet && (resource == PermDistros || resource == PermProfiles || resource == PermGroups) {
			handler.ServeHTTP(w, r)
			return
		}

		// power is a resource/action that we need to filter on the backend because
		// it can be invoked with different resource params (reservation name or hosts list)
		if r.Method == http.MethodPatch && r.URL.Path == api.HostsPower {
			handler.ServeHTTP(w, r)
			return
		}

		reqPermString += resource + PermDividerToken

		var resourceName string
		resourceType := strings.TrimRight(resource, "s")

		ps := httprouter.ParamsFromContext(r.Context())
		if ps == nil {
			reqPermString += PermWildcardToken + PermDividerToken
		} else if resource == PermReservations {
			resourceName = ps.ByName("resName")
			reqPermString += resourceName + PermDividerToken
		} else {
			resourceName = ps.ByName(resourceType + "Name")
			reqPermString += resourceName + PermDividerToken
		}

		switch r.Method {
		case http.MethodGet:
			reqPermString += PermViewAction
		case http.MethodPost:
			reqPermString += PermCreateAction
		case http.MethodPatch, http.MethodPut:
			reqPermString += PermEditAction + PermDividerToken
			editPart := getEditPart(r, resource)
			reqPermString += editPart
		case http.MethodDelete:
			reqPermString += PermDeleteAction
		}

		if igor.Auth.Scheme != "local" && ps.ByName("userName") != IgorAdmin {
			if strings.HasSuffix(reqPermString, "edit:password") || strings.HasSuffix(reqPermString, "edit:reset") {
				logger.Warn().Msgf("user '%s' requested igor password change but authentication scheme is %s, not local",
					user.Name, igor.Auth.Scheme)
				rb.Message = fmt.Sprintf("not allowed - igor authentication is being managed externally")
				makeJsonResponse(w, http.StatusForbidden, rb)
				return
			}
		}

		p, err := NewPermission(reqPermString)
		if err != nil {
			rb.Message = err.Error()
			makeJsonResponse(w, http.StatusInternalServerError, rb)
			return
		}

		if authInfo.IsPermitted(p) {
			handler.ServeHTTP(w, r)
		} else {

			// If the permission check failed it's possible the requested resource doesn't exist. Check for it and decide
			// how to reply.

			if resourceName != "" {

				var exists bool
				errStatus := http.StatusInternalServerError

				err = performDbTx(func(tx *gorm.DB) error {

					if groupSliceContains(user.Groups, GroupAdmins) {
						switch resource {
						case "images":
							exists, err = imageExists(resourceName, tx)
						case "hostpolicy":
							exists, err = hostPolicyExists(resourceName, tx, hlog.FromRequest(r))
							resourceType = "policy" // for name consistency on CLI
						}
					} else {
						if resource == "images" || resource == "hostpolicy" {
							errStatus = http.StatusForbidden
							return fmt.Errorf("access denied")
						}
					}

					switch resource {
					case PermReservations:
						exists, err = resvExists(resourceName, tx)
					case PermGroups:
						exists, err = groupExists(resourceName, tx)
					case PermDistros:
						exists, err = distroExists(resourceName, tx)
					case PermProfiles:
						exists, err = profileExists(resourceName, tx)
					case PermUsers:
						exists, err = userExists(resourceName, tx)
					case PermHosts:
						exists, err = hostExists(resourceName, tx)
					}

					if err != nil {
						return err
					}
					return nil
				})

				if err != nil {
					rb.Message = err.Error()
					makeJsonResponse(w, errStatus, rb)
					return
				} else if !exists {
					rb.Message = fmt.Sprintf("the %s '%s' does not exist", resourceType, resourceName)
					makeJsonResponse(w, http.StatusNotFound, rb)
					return
				}

				if groupSliceContains(user.Groups, GroupAdmins) {
					rb.Message = "elevated access is required before running this command"
				} else {
					rb.Message = fmt.Sprintf("you cannot access the %s '%s'", resourceType, resourceName)
				}
			} else {
				if groupSliceContains(user.Groups, GroupAdmins) {
					rb.Message = "elevated access is required before running this command"
				} else {
					rb.Message = "access denied"
				}
			}

			makeJsonResponse(w, http.StatusForbidden, rb)
		}
	})
}

// getEditPart generates the list of fields for a resource that a request is asking
// permission to edit. Note that some types of resources have fields that can be
// edited (call PATCH) by more than just the owner.
func getEditPart(r *http.Request, resource string) (editPart string) {

	body := getBodyFromContext(r)

	if resource == PermUsers {

		attrs := make([]string, 0, len(body))
		for k := range body {
			switch k {
			case "password", "email", "reset", "fullName":
				attrs = append(attrs, k)
			default:
				continue
			}
		}
		editPart = strings.Join(attrs, PermSubpartToken)

	} else if resource == PermReservations {

		attrs := make([]string, 0, len(body))
		for k := range body {
			switch k {
			case "group", "owner", "distro", "profile", "extend", "name", "description", "kernelArgs", "drop":
				attrs = append(attrs, k)
			case "extendMax":
				attrs = append(attrs, "extend")
			default:
				continue
			}
		}
		editPart = strings.Join(attrs, PermSubpartToken)

	} else {
		editPart = PermWildcardToken
	}

	return
}
