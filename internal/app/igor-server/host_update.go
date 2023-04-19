// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net"
	"net/http"

	zl "github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"
)

func doUpdateHost(hostName string, changes map[string]interface{}, r *http.Request) (status int, err error) {

	clog := hlog.FromRequest(r)

	status = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		hList, ghStatus, ghErr := getHosts([]string{hostName}, false, tx)
		if err != nil {
			status = ghStatus
			return ghErr
		}

		err = dbEditHosts(hList, changes, tx)
		if err != nil {
			return err // uses default err status
		} else {
			var doDump bool
			var cDumpErr error
			var finalPath string

			for k := range changes {
				if k == "HostPolicy" || k == "ip" || k == "eth" {
					if k == "HostPolicy" {
						k = "hostPolicy"
					}
					clog.Debug().Msgf("config field %s changed when editing host %s", k, hostName)
					doDump = true
				}
			}

			if doDump {
				clog.Info().Msg("writing new version of cluster config file")
				var clusters []Cluster
				var yDoc []byte

				if clusters, cDumpErr = dbReadClusters(nil, tx); cDumpErr == nil {
					if yDoc, cDumpErr = assembleYamlOutput(clusters); cDumpErr == nil {
						finalPath, cDumpErr = updateClusterConfigFile(yDoc, clog)
					}
				}
			}
			clog.Info().Msgf("%s updated on host update", finalPath)
			return cDumpErr
		}

	}); err == nil {
		status = http.StatusOK
	}
	return
}

func parseHostEditParams(editParams map[string]interface{}, clog *zl.Logger) (map[string]interface{}, int, error) {

	changes := map[string]interface{}{}

	// check for IP change
	if val, ok := editParams["ip"].(string); ok {
		hostIP := net.ParseIP(val)
		changes["ip"] = hostIP.String()
	}
	// check for hostname change
	if val, ok := editParams["hostname"].(string); ok {
		changes["host_name"] = val
	}
	// check for mac address chnage
	if val, ok := editParams["mac"].(string); ok {
		if _, err := net.ParseMAC(val); err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("invalid mac address value %s given for host update", val)
		} else {
			changes["mac"] = val
		}
	}
	// check for eth change
	if val, ok := editParams["eth"].(string); ok {
		changes["eth"] = val
	}
	// determine if new host policy
	if val, ok := editParams["hostPolicy"].(string); ok {
		if val == "" {
			// if host policy value handed in was an empty string, assume user wanted to
			// simply remove the current policy and replace with the default
			defaultPolicy, _ := dbReadHostPoliciesTx(map[string]interface{}{"name": DefaultPolicyName}, clog)
			changes["HostPolicy"] = defaultPolicy[0]
		} else {
			if hpList, err := dbReadHostPoliciesTx(map[string]interface{}{"name": val}, clog); err != nil {
				return nil, http.StatusInternalServerError, err
			} else if len(hpList) == 0 {
				return nil, http.StatusBadRequest, fmt.Errorf("no host policy found with name %s", val)
			} else {
				changes["HostPolicy"] = hpList[0]
			}
		}

	}

	return changes, http.StatusOK, nil
}
