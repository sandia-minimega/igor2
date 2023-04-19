// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"github.com/rs/zerolog/hlog"
	"net/http"
	"strconv"
)

// doReadClusters performs a DB lookup of Cluster records that match the provided queryParams. It will
// return these as a list which can also be empty/nil if no matches were found. It will also pass back any
// encountered GORM errors with status code 500.
func doReadClusters(queryParams map[string]interface{}) ([]Cluster, int, error) {
	cList, err := dbReadClustersTx(queryParams)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return cList, http.StatusOK, nil
}

func parseClusterSearchParams(queryMap map[string][]string, r *http.Request) (queryParams map[string]interface{}, doDump bool, getYaml bool) {

	clog := hlog.FromRequest(r)

	for key, val := range queryMap {
		switch key {
		case "dump":
			doDump, _ = strconv.ParseBool(val[0])
		case "getYaml":
			getYaml, _ = strconv.ParseBool(val[0])
		case "name":
			queryParams["name"] = val
		case "prefix":
			queryParams["prefix"] = val
		default:
			clog.Warn().Msgf("unrecognized search parameter '%s' with args '%v'", key, val)
		}
	}
	return
}
