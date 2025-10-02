// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"igor2/internal/pkg/common"
	"math"
	"net/http"

	"github.com/rs/zerolog/hlog"
)

var elevate = struct{}{}

// destination for PATCH /elevate
func handleElevateUser(w http.ResponseWriter, r *http.Request) {
	clog := hlog.FromRequest(r)
	actionPrefix := "elevate user"
	user := getUserFromContext(r)
	clog.Debug().Msgf("%s: %s", actionPrefix, user.Name)
	var status int
	rb := common.NewResponseBody()
	if user.Name == IgorAdmin {
		out := fmt.Sprintf("%s does not require the elevate privilege", IgorAdmin)
		rb.Message = out
		clog.Info().Msg(out)
		status = http.StatusAccepted
	} else if groupSliceContains(user.Groups, GroupAdmins) {
		igor.ElevateMap.Put(user.Name, elevate)
		out := fmt.Sprintf("elevate for user '%s' is active for next %v minutes", user.Name, igor.ElevateMap.TTL().Minutes())
		clog.Info().Msgf("%s success - %s", actionPrefix, out)
		rb.Message = out
		status = http.StatusOK
	} else {
		out := fmt.Sprintf("user '%s' is not an admin", user.Name)
		clog.Warn().Msgf("%s failed - %s", actionPrefix, out)
		rb.Message = out
		status = http.StatusForbidden
	}

	makeJsonResponse(w, status, rb)
}

// destination for GET /elevate
func handleElevateUserStatus(w http.ResponseWriter, r *http.Request) {
	clog := hlog.FromRequest(r)
	actionPrefix := "status elevate"
	user := getUserFromContext(r)
	clog.Debug().Msgf("%s: %s", actionPrefix, user.Name)
	var status int
	rb := common.NewResponseBody()
	if user.Name == IgorAdmin {
		out := fmt.Sprintf("%s has no elevate privilege", IgorAdmin)
		rb.Message = out
	} else if groupSliceContains(user.Groups, GroupAdmins) {
		remaining := igor.ElevateMap.Remaining(user.Name)
		var out string
		if remaining == 0 {
			out = fmt.Sprintf("elevate for user '%s' has expired", user.Name)
		} else {
			if remaining < 60 {
				out = fmt.Sprintf("elevate for user '%s' has %v seconds remaining", user.Name, remaining)
			} else {
				minRemaining := math.Round((float64(remaining)/60)*10) / 10
				out = fmt.Sprintf("elevate for user '%s' has %v minutes remaining", user.Name, minRemaining)
			}
		}
		clog.Info().Msgf("%s success - %s", actionPrefix, out)
		rb.Message = out
		status = http.StatusOK
	} else {
		out := fmt.Sprintf("user '%s' is not an admin", user.Name)
		clog.Warn().Msgf("%s failed - %s", actionPrefix, out)
		rb.Message = out
		status = http.StatusForbidden
	}

	makeJsonResponse(w, status, rb)
}

// destination for DELETE /elevate
func handleElevateUserCancel(w http.ResponseWriter, r *http.Request) {
	clog := hlog.FromRequest(r)
	actionPrefix := "cancel elevate"
	user := getUserFromContext(r)
	clog.Debug().Msgf("%s: %s", actionPrefix, user.Name)
	var status int
	rb := common.NewResponseBody()

	igor.ElevateMap.Remove(user.Name)
	if user.Name == IgorAdmin {
		out := fmt.Sprintf("%s has no elevate privilege", IgorAdmin)
		rb.Message = out
	} else if groupSliceContains(user.Groups, GroupAdmins) {
		msg := fmt.Sprintf("elevate for user '%s' is canceled", user.Name)
		clog.Info().Msgf("%s success - %s", actionPrefix, msg)
		rb.Message = msg
		status = http.StatusOK
	} else {
		msg := fmt.Sprintf("user '%s' is not an admin", user.Name)
		clog.Warn().Msgf("%s failed - %s", actionPrefix, msg)
		rb.Message = msg
		status = http.StatusForbidden
	}

	makeJsonResponse(w, status, rb)
}

// userElevated returns true if the named user is currently elevated or if they
// are logged in as igor-admin. Returns false otherwise.
func userElevated(username string) bool {
	return igor.ElevateMap.Contains(username) || username == IgorAdmin
}
