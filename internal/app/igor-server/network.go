// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"strconv"
	"time"
)

var (
	networkSetFuncs   map[string]func([]Host, int) error
	networkClearFuncs map[string]func([]Host) error
	networkVlanFuncs  map[string]func() (map[string]string, error)
)

// Configure the given nodes into the specified 802.1ad outer VLAN
func networkSet(nodes []Host, vlan int) error {
	// if in dev env, just log and return
	if DEVMODE {
		logger.Debug().Msg("in dev env running networkSet(), no external action taken")
		return nil
	}

	if igor.Vlan.Network == "" {
		// they don't want to do vlan segmentation
		logger.Debug().Msg("not doing vlan segmentation")
		return nil
	}

	f, ok := networkSetFuncs[igor.Vlan.Network]
	if !ok {
		logger.Error().Msgf("no such network mode: %v", igor.Vlan.Network)
	}
	return f(nodes, vlan)
}

// Clear any 802.1ad configuration on the given nodes
func networkClear(nodes []Host) error {
	// if in dev env, just log and return
	if DEVMODE {
		logger.Debug().Msg("in dev env running networkClear(), no external action taken")
		return nil
	}

	if igor.Vlan.Network == "" {
		// they don't want to do vlan segmentation
		logger.Debug().Msg("not doing vlan segmentation")
		return nil
	}

	f, ok := networkClearFuncs[igor.Vlan.Network]
	if !ok {
		logger.Error().Msgf("no such network mode: %v", igor.Vlan.Network)
	}
	return f(nodes)
}

// Collect VLAN status for all nodes
// This should return a key-value map where the key is the host name
// and the value is the string form of the vlan value
func networkVlan() (map[string]string, error) {
	// if in dev env, just return the vlan assigned
	// to the reservation's hosts in a map
	if DEVMODE {
		logger.Debug().Msg("in dev env running networkVlan(), just returning res vlans assigned to hosts")
		reservations, err := dbReadReservationsTx(map[string]interface{}{}, map[string]time.Time{})
		if err != nil {
			return nil, err
		}
		result := map[string]string{}
		for _, res := range reservations {
			for _, host := range res.Hosts {
				result[host.Name] = strconv.Itoa(res.Vlan)
			}
		}
		return result, nil
	}

	if igor.Vlan.Network == "" {
		// they don't want to do vlan segmentation
		logger.Debug().Msg("not doing vlan segmentation")
		return nil, nil
	}

	f, ok := networkVlanFuncs[igor.Vlan.Network]
	if !ok {
		logger.Error().Msgf("no such network mode: %v", igor.Vlan.Network)
	}
	return f()
}

func nextVLAN() (int, error) {
	reservations, err := dbReadReservationsTx(map[string]interface{}{}, map[string]time.Time{})
	if err != nil {
		return 0, err
	}
OuterLoop:
	for i := igor.Vlan.RangeMin; i <= igor.Vlan.RangeMax; i++ {
		for _, res := range reservations {
			if i == res.Vlan {
				continue OuterLoop
			}
		}

		return i, nil
	}

	return 0, fmt.Errorf("no vlans available")
}
