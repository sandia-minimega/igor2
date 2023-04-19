// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"net/http"
	"strings"

	"igor2/internal/pkg/common"

	"github.com/rs/zerolog/hlog"
)

func handleCbs(w http.ResponseWriter, r *http.Request) {
	clog := hlog.FromRequest(r)
	actionPrefix := "convert PXE.cfg to local boot"
	rb := common.NewResponseBody()

	ip := strings.Split(r.RemoteAddr, ":")[0]

	queryParams := map[string]interface{}{"ip": ip}
	hosts, status, err := doReadHosts(queryParams)
	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else if len(hosts) == 0 {
		clog.Warn().Msgf("%s failed - no hosts found matching IP address %s", actionPrefix, ip)
		status = http.StatusBadRequest
	} else {
		clog.Debug().Msgf("host search with IP %s returned %v results", ip, len(hosts))
		host := hosts[0]
		if err := setLocalConfig(&host); err != nil {
			clog.Warn().Msgf("%s failed to convert pxe.cfg file to local boot for host %s - %v", actionPrefix, host.Name, err)
		}
		status = http.StatusOK
	}

	w.WriteHeader(status)
	if _, err := w.Write([]byte{}); err != nil {
		panic(err)
	}
}

func getInfo(w http.ResponseWriter, r *http.Request) {
	clog := hlog.FromRequest(r)
	actionPrefix := "get user and hosts based on reservation related to calling host"
	rb := common.NewResponseBody()
	result := ""

	ip := strings.Split(r.RemoteAddr, ":")[0]
	queryParams := map[string]interface{}{"ip": ip}
	hosts, status, err := doReadHosts(queryParams)
	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else if len(hosts) == 0 {
		clog.Warn().Msgf("%s failed - no hosts found matching IP address %s", actionPrefix, ip)
		status = http.StatusBadRequest
	} else {
		clog.Debug().Msgf("host search with IP %s (as []byte) returned %v results", ip, len(hosts))
		host := hosts[0]
		query := map[string]interface{}{"hosts": []int{host.ID}}
		resvs, _, err := doReadReservations(query, nil)
		if err != nil {
			clog.Error().Msgf("%s: error returning reservations using host IP %v: %v", actionPrefix, ip, err.Error())
		} else if len(resvs) == 0 {
			clog.Error().Msgf("%s: error no reservations returned using host IP %v", actionPrefix, ip)
		} else {
			res := resvs[0]
			result = res.Name + " " + res.Owner.Name
			for _, host := range res.Hosts {
				result = result + " " + host.Name
			}
			result = result + "\n"
			status = http.StatusOK
		}
	}

	w.WriteHeader(status)
	if _, err := w.Write([]byte(result)); err != nil {
		panic(err)
	}
}
