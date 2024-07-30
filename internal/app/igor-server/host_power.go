// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	zl "github.com/rs/zerolog"

	"igor2/internal/pkg/common"
)

const (
	PowerOff   = "off"
	PowerOn    = "on"
	PowerCycle = "cycle"
)

// Ensures the selected power command is recognized and spelled correctly (on/off/cycle, case-insenstive).
func checkPowerCmdSyntax(cmd string) error {
	c := strings.TrimSpace(strings.ToLower(cmd))
	if !(c == PowerOn || c == PowerOff || c == PowerCycle) {
		return fmt.Errorf("power command '%s' not recognized", c)
	}
	return nil
}

// Maps the power command parameters to a list of hosts and checks permissions to ensure the user
// can actually issue a power command for those hosts.
func checkPowerParams(powerParams map[string]interface{}, r *http.Request) (string, []string, int, error) {

	cmd := strings.ToLower(powerParams["cmd"].(string))

	var err error
	var hostNames []string

	if hostExpr, hok := powerParams["hosts"].(string); hok {
		hostNames, err = common.SplitList(hostExpr)
		if err != nil {
			return cmd, nil, http.StatusNotFound, err
		}
		sort.Slice(hostNames, func(i, j int) bool {
			return hostNames[i] < hostNames[j]
		})

	} else if resName, rok := powerParams["resName"].(string); rok {

		queryParams := map[string]interface{}{"name": resName}
		if res, rrErr := dbReadReservationsTx(queryParams, nil); rrErr != nil {
			return cmd, nil, http.StatusInternalServerError, rrErr
		} else {
			if len(res) == 1 {
				hostNames = hostNamesOfHosts(res[0].Hosts)
			} else {
				return cmd, nil, http.StatusNotFound, fmt.Errorf("reservation '%s' not found", resName)
			}
		}
	} else {
		// because we validate params earlier in the call chain we should never reach this code
		return cmd, nil, http.StatusInternalServerError, fmt.Errorf("no parameter specifying hosts or reservations")
	}

	user := getUserFromContext(r)

	authInfo, err := user.getAuthzInfo()
	if err != nil {
		return cmd, hostNames, http.StatusInternalServerError, err
	}

	for _, h := range hostNames {
		powerPerm, _ := NewPermission(NewPermissionString(PermPowerAction, h))
		if !authInfo.IsPermitted(powerPerm) {
			return cmd, hostNames, http.StatusForbidden, fmt.Errorf("user attempted power command on %v but does not have permission to run power commands on host %v", hostNames, h)
		}
	}

	return cmd, hostNames, http.StatusOK, nil
}

// Runs the actual power command for the service that controls host power options.
func doPowerHosts(action string, hostList []string, clog *zl.Logger) (int, error) {

	clog.Info().Msgf("running power operation '%s' on node(s) %v", action, hostList)

	switch action {
	case PowerOff:

		if DEVMODE {
			devUpdatePowerMap(PowerOff, hostList)
			return http.StatusOK, nil
		}

		if igor.ExternalCmds.PowerOff == "" {
			return http.StatusInternalServerError, fmt.Errorf("power-off configuration missing")
		}

		if err := runAll(igor.ExternalCmds.PowerOff, hostList); err != nil {
			return http.StatusInternalServerError, err
		}

	case PowerCycle:

		if DEVMODE {
			devUpdatePowerMap(PowerOn, hostList)
			return http.StatusOK, nil
		}

		var useDefaultCycleCmd = true
		var oioFlag = ""

		if igor.ExternalCmds.PowerCycle == "" && igor.ExternalCmds.PowerOff == "" {
			return http.StatusInternalServerError, fmt.Errorf("power-cycle and power-off configuration missing")
		}

		if strings.HasPrefix(igor.ExternalCmds.PowerCycle, "ipmitool") {
			// ipmitool may not turn a node on as part of a cycle command if it is off to start with
			// so default to using two commands, first off then on
			logger.Debug().Msg("for ipmitool, using power on/off commands instead of cycle")
			useDefaultCycleCmd = false
		}

		if strings.HasPrefix(igor.ExternalCmds.PowerCycle, "ipmipower") &&
			!strings.Contains(igor.ExternalCmds.PowerCycle, "--on-if-off") {
			// if ipmipower is being used and the cycle command doesn't include --on-if-off"
			// then append it to the command
			logger.Debug().Msg("adding on-if-off flag to ipmipower command")
			oioFlag = " --on-if-off"
		}

		if useDefaultCycleCmd {

			if igor.ExternalCmds.PowerCycle == "" {
				return http.StatusInternalServerError, fmt.Errorf("power-cycle configuration missing")
			}

			if err := runAll(igor.ExternalCmds.PowerCycle+oioFlag, hostList); err != nil {
				return http.StatusInternalServerError, err
			}
			// if power cycle command works on its own, we can return from this point
			return http.StatusOK, nil

		} else {

			if igor.ExternalCmds.PowerOff == "" {
				return http.StatusInternalServerError, fmt.Errorf("power-off configuration missing")
			}

			if err := runAll(igor.ExternalCmds.PowerOff, hostList); err != nil {
				return http.StatusInternalServerError, err
			}
		}

		fallthrough // assuming power-off is used in place of power-cycle, execute next case

	case PowerOn:

		if DEVMODE {
			devUpdatePowerMap(PowerOn, hostList)
			return http.StatusOK, nil
		}

		if igor.ExternalCmds.PowerOn == "" {
			return http.StatusInternalServerError, fmt.Errorf("power-on configuration missing")
		}

		if err := runAll(igor.ExternalCmds.PowerOn, hostList); err != nil {
			return http.StatusInternalServerError, err
		}

	default:
		return http.StatusBadRequest, fmt.Errorf("invalid power operation : %s", action)
	}

	return http.StatusOK, nil
}

// powerOffResNodes explicitly sends the power 'off' command to the nodes of a deleted/expired reservation.
func powerOffResNodes(reservation *Reservation) error {
	hostnames := namesOfHosts(reservation.Hosts)
	if _, pErr := doPowerHosts(PowerOff, hostnames, &logger); pErr != nil {
		return fmt.Errorf("problem powering off hosts %v for end of reservation '%s': %v", hostnames, reservation.Name, pErr)
	}
	return nil
}

func devUpdatePowerMap(action string, hostNames []string) {
	powerMapMU.Lock()
	for _, h := range hostNames {
		powerVal := false
		if action == PowerOn {
			powerVal = true
		}
		powerMap[h] = &powerVal
	}
	powerMapMU.Unlock()
}
