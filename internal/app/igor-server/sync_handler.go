// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"igor2/internal/pkg/common"

	"github.com/rs/zerolog/hlog"
)

func syncHandler(w http.ResponseWriter, r *http.Request) {
	// runs a sync command on a given option
	// options currently include: arista
	clog := hlog.FromRequest(r)
	actionPrefix := "sync"
	rb := common.NewResponseBody()
	syncParams := r.URL.Query()

	result, status, err := runSync(syncParams)
	if err != nil {
		stdErrorResp(rb, status, actionPrefix, err, clog)
	} else {
		clog.Info().Msgf("%s success", actionPrefix)
	}
	rb.Data["sync"] = result

	makeJsonResponse(w, status, rb)
}

// Gather data integrity information, report, and fix
func runSync(params map[string][]string) (result map[string]interface{}, status int, err error) {
	// change based on outcome
	status = http.StatusInternalServerError

	force := false
	if f, ok := params["force"]; ok {
		force = strings.ToLower(f[0]) == "true"
	}
	quiet := false
	if q, ok := params["quiet"]; ok {
		quiet = strings.ToLower(q[0]) == "true"
	}
	// already check if present in validation
	cmd := strings.ToLower(params["cmd"][0])

	switch cmd {
	case "arista":
		if igor.Vlan.Network == "" {
			// they're not doing vlan segmentation
			err := fmt.Errorf("not doing vlan segmentation, nothing to sync")
			return nil, http.StatusBadRequest, err
		}
		return syncArista(force, quiet)
	default:
		status = http.StatusBadRequest
		err = fmt.Errorf("sync command %v not recognized", cmd)
		return
	}
}

// sync builds a map of all hosts, where each host has the following
// information captured about it:
// map["Host.Name"]{"powered":string, "res_vlan":string, "actual_vlan":string}
func syncArista(force, quiet bool) (result map[string]interface{}, status int, err error) {
	result = make(map[string]interface{})

	// we only need to care about hosts currently assigned to reservations
	hosts, err := getReservedHosts()
	if err != nil {
		return result, http.StatusInternalServerError, err
	}

	// get Arista vlan data
	logger.Debug().Msg("retrieving Arista data, this may take a few moments...")
	gt, err := networkVlan()
	if err != nil {
		logger.Error().Msg("Error gathering VLAN data from Arista")
		return result, http.StatusInternalServerError, err
	}

	// get all reservations
	reservations, err := dbReadReservationsTx(map[string]interface{}{}, map[string]time.Time{})
	if err != nil {
		logger.Error().Msgf("error retrieving reservations for sync: %v", err)
		return result, http.StatusInternalServerError, err
	}

	// determine what each host vlan should be from reservation
	withRes := map[string]map[string]interface{}{}
	for _, r := range reservations {
		vlan := strconv.Itoa(r.Vlan)

		for _, host := range r.Hosts {
			withRes[host.Name] = map[string]interface{}{"res_vlan": vlan}
		}
	}

	// report to construct
	report := make(map[string]map[string]string)
	// aggregate all to report and sync the node if force
	powerMapMU.Lock()
	for _, host := range hosts {
		hostName := host.HostName
		data := map[string]string{}
		if resInfo, ok := withRes[hostName]; ok {
			data["res_vlan"] = resInfo["res_vlan"].(string)
			if data["res_vlan"] == "0" || data["res_vlan"] == "" {
				data["res_vlan"] = "(none)"
			}
		} else {
			data["res_vlan"] = "(unknown)"
		}

		if powerInfo, ok := powerMap[hostName]; ok {
			if powerInfo == nil {
				data["powered"] = "unknown"
			} else if *powerInfo {
				data["powered"] = PowerOn
			} else {
				data["powered"] = PowerOff
			}
		} else {
			data["powered"] = "unknown"
		}

		data["switch_vlan"] = gt[hostName]
		// if arista had no vlan assigned, make explicit for readability
		if data["switch_vlan"] == "0" || data["switch_vlan"] == "" {
			data["switch_vlan"] = "(none)"
		}

		if force && data["res_vlan"] != data["switch_vlan"] {
			vlan, err := strconv.Atoi(data["res_vlan"])
			if err != nil {
				return result, http.StatusInternalServerError, err
			}
			if err := networkSet([]Host{host}, vlan); err != nil {
				logger.Error().Msgf("unable to set up network isolation for host %v", hostName)
				data["status"] = "VLAN correction failed!"
			} else {
				data["status"] = "VLAN correction succeeded"
			}
		}
		report[hostName] = data
	}
	powerMapMU.Unlock()

	logger.Debug().Msgf("report compiled by syncArista: %v", report)
	result["command"] = "arista"
	result["report"] = report
	result["force"] = strconv.FormatBool(force)
	result["quiet"] = strconv.FormatBool(quiet)

	return result, http.StatusOK, nil
}

func validateSyncParams(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var validateErr error
		clog := hlog.FromRequest(r)

		if r.Method == http.MethodGet {

			syncParams := r.URL.Query()

			if len(syncParams) > 0 {
				_, cmd := syncParams["cmd"]
				if !cmd {
					validateErr = fmt.Errorf("missing sync command value (required)")
				} else {

				getParamLoop:
					for key, val := range syncParams {
						switch strings.TrimSpace(key) {
						case "cmd":
							cmd := strings.TrimSpace(strings.ToLower(val[0]))
							if !stdNameCheckPattern.MatchString(cmd) {
								validateErr = fmt.Errorf("'%s' is not a legal sync command option", cmd)
								break getParamLoop
							}
						case "force":
							force := strings.TrimSpace(strings.ToLower(val[0]))
							if !(force == "true" || force == "false") {
								validateErr = fmt.Errorf("force value must be true or false")
								break getParamLoop
							}
						case "quiet":
							quiet := strings.TrimSpace(strings.ToLower(val[0]))
							if !(quiet == "true" || quiet == "false") {
								validateErr = fmt.Errorf("quiet value must be true or false")
								break getParamLoop
							}
						default:
							validateErr = NewUnknownParamError(key, val)
							break getParamLoop
						}
					}
				}
			} else {
				validateErr = NewMissingParamError("")
			}
		}

		if validateErr != nil {
			clog.Warn().Msgf("validateSyncParams - %v", validateErr)
			createValidationErrMessage(validateErr, w)
			return
		}

		handler.ServeHTTP(w, r)
	})
}
