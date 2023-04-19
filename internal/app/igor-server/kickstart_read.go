// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"net/http"

	"github.com/rs/zerolog/hlog"
)

func doReadKickstart(queryParams map[string]interface{}) ([]Kickstart, int, error) {
	ksList, err := dbReadKickstartTx(queryParams)
	if err != nil {
		return ksList, http.StatusInternalServerError, err
	}

	return ksList, http.StatusOK, nil
}

func parseKSSearchParams(queryMap map[string][]string, r *http.Request) (map[string]interface{}, int, error) {
	// parse resParams and convert []string vals to proper corresponding types
	// template: db.Where(map[string]interface{}{"name": "jinzhu", "age": 20}).Find(&users)

	clog := hlog.FromRequest(r)
	status := http.StatusOK

	queryParams := map[string]interface{}{}
	// extract and convert each attribute, if present, and add to query

	for key, val := range queryMap {
		switch key {
		case "name":
			queryParams["file_name"] = val
		default:
			clog.Warn().Msgf("unrecognized search parameter '%s' with args '%v'", key, val)
		}
	}

	return queryParams, status, nil
}
