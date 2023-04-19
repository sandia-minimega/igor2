// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"github.com/rs/zerolog"
	"gorm.io/gorm"
	"net/http"
)

// Maps the power command parameters to a list of hosts and checks permissions to ensure the user
// can actually issue a power command for those hosts.
func checkApplyPolicyParams(applyParams map[string]interface{}, clog *zerolog.Logger) (policy *HostPolicy, hosts *[]Host, status int, err error) {

	hostPolicyName := applyParams["policy"].(string)
	val := applyParams["nodeList"].(string)
	status = http.StatusInternalServerError

	hostList := igor.splitRange(val)

	if err = performDbTx(func(tx *gorm.DB) error {

		hpList, ghpStatus, ghpErr := getHostPolicies([]string{hostPolicyName}, tx, clog)
		if ghpErr != nil {
			status = ghpStatus
			return ghpErr
		}
		policy = &hpList[0]

		hList, ghStatus, ghErr := getHosts(hostList, true, tx)
		if err != nil {
			status = ghStatus
			return ghErr
		}
		hosts = &hList

		return nil

	}); err == nil {
		status = http.StatusOK
	}

	return
}

// doApplyPolicy updates the given hosts with the supplied policy.
func doApplyPolicy(hostPolicy *HostPolicy, hosts *[]Host) (status int, err error) {

	status = http.StatusInternalServerError // default status, overridden at end if no errors

	if err = performDbTx(func(tx *gorm.DB) error {

		return dbEditHosts(*hosts, map[string]interface{}{"HostPolicy": *hostPolicy}, tx) // uses default err status

	}); err == nil {
		status = http.StatusOK
	}
	return
}
