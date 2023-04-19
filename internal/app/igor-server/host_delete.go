// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"
)

// doDeleteHost removes a cluster host/node from igor's database.
//
// returns:
//
//	204,nil if delete was successful
//	404,error if host cannot be found
//	409,error if delete was not allowed due to its current state
//	500,error if an internal error occurred
func doDeleteHost(hostName string, r *http.Request) (status int, err error) {

	clog := hlog.FromRequest(r)
	status = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		hList, ghStatus, ghErr := getHosts([]string{hostName}, false, tx)
		if ghErr != nil {
			status = ghStatus
			return ghErr
		}
		host := hList[0]

		// make sure host isn't tied to current or future reservations
		if len(host.Reservations) > 0 {
			status = http.StatusConflict
			return fmt.Errorf("cannot delete a node with an active reservation")
		}

		// change the host's policy to default if it's something else
		if host.HostPolicy.Name != DefaultPolicyName {
			hpList, err := dbReadHostPoliciesTx(map[string]interface{}{"name": DefaultPolicyName}, clog)
			if err != nil {
				return err
			}
			err = dbEditHosts(hList, map[string]interface{}{"hostPolicy": hpList[0]}, tx)
			if err != nil {
				return err
			}
		}

		deleteErr := dbDeleteHosts(hList, tx)
		if deleteErr != nil {
			return deleteErr
		} else {
			var clusters []Cluster
			var yDoc []byte
			var cDumpErr error
			var finalPath string
			if clusters, cDumpErr = dbReadClusters(nil, tx); cDumpErr == nil {
				if yDoc, cDumpErr = assembleYamlOutput(clusters); cDumpErr == nil {
					finalPath, cDumpErr = updateClusterConfigFile(yDoc, clog)
				}
			}
			if cDumpErr == nil {
				clog.Info().Msgf("%s updated on host delete", finalPath)
			}
			return cDumpErr
		}
	}); err == nil {
		status = http.StatusOK
	}
	return

}
