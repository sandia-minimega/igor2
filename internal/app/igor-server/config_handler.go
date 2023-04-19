// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"net/http"

	"igor2/internal/pkg/common"
)

// configHandler returns the complete server configuration file as JSON.
func configHandler(w http.ResponseWriter, _ *http.Request) {
	rb := common.NewResponseBody()
	rb.Data["igor"] = igor.getServerConfig()
	makeJsonResponse(w, http.StatusOK, rb)
}

// settingsHandler returns useful server configuration settings as JSON.
func settingsHandler(w http.ResponseWriter, _ *http.Request) {
	rb := common.NewResponseBody()
	rb.Data["igor"] = igor.getServerSettings()
	makeJsonResponse(w, http.StatusOK, rb)
}
