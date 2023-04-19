// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net"
	"sort"
	"time"

	"igor2/internal/pkg/common"

	"gorm.io/gorm"
)

// Hardware information (tbd)
// BIOS information (tbd)
// Powered on/off
// state - Available, Reserved, Drain (temp notAvailable)
// Network information (IP, DHCP info)
// AccessGroup - defines node policy (only the group listed may use this node) the default is 'all'. When making a
// reservation for a group, Igor should prioritize using the group's access-restricted nodes first.
// current reservation profile

const (
	PermHosts = "hosts"
)

// Host is the compute resource being reserved. It's data contains all relevant information needed by
// igor to make reservations and issue commands to interact with a given host or get information
// about its current status.
type Host struct {
	Base
	Name           string `gorm:"unique; notNull"`
	HostName       string `gorm:"unique; notNull"`
	SequenceID     int    `gorm:"notNull; uniqueIndex:idx_cluster_seq"`
	Eth            string
	Mac            string `gorm:"unique; notNull"`
	IP             string
	State          HostState // State is the HostState of this node. Default when created is HostBlocked.
	ClusterID      int       `gorm:"notNull; uniqueIndex:idx_cluster_seq"`
	Cluster        Cluster   `gorm:"->;<-:create; notNull"` // read/create only; hosts never change clusters
	HostPolicyID   int
	HostPolicy     HostPolicy       `gorm:"notNull"` // host policy assigned to this host. Assigned to policy DefaultPolicyName at host creation.
	Reservations   []Reservation    `gorm:"many2many:reservations_hosts;"`
	MaintenanceRes []MaintenanceRes `gorm:"many2many:maintenanceres_hosts;"`
}

func (h *Host) GetHostIPs() ([]net.IP, error) {
	if len(h.IP) > 0 {
		return []net.IP{net.ParseIP(h.IP)}, nil
	}
	if DEVMODE {
		return []net.IP{}, nil
	}
	ips, err := net.LookupIP(h.Name)
	if err != nil {
		return ips, fmt.Errorf("failure looking up %v: %v", h, err)
	}
	return ips, nil
}

func (h *Host) BeforeDelete(_ *gorm.DB) (delErr error) {

	if h.State == HostReserved {
		return fmt.Errorf("cannot delete node %s - active reservation present", h.Name)
	}

	if len(h.Reservations) > 0 {
		return fmt.Errorf("cannot delete node %s - future reservation(s) present", h.Name)
	}
	return nil
}

func (h *Host) getHostData(powered *bool, user *User) common.HostData {

	resNames := resNamesOfResList(h.Reservations)
	groups := make([]string, 0, 10)
	for _, group := range h.HostPolicy.AccessGroups {
		groups = append(groups, group.Name)
	}

	// check if restricted by group access
	restricted := !user.isMemberOfAnyGroup(h.HostPolicy.AccessGroups)

	// then if the user is in an access group, check for time availability conditions
	if !restricted && len(h.HostPolicy.NotAvailable) > 0 {
		now := time.Now()
		restricted, _, _ = hasScheduleBlockConflict(h.HostPolicy.NotAvailable, now, now.Add(getDurationToClockTime(time.Minute)), &logger)
	}

	ips, err := h.GetHostIPs()
	ip := "Unavailable"
	if err == nil && len(ips) > 0 {
		// assume the first IP is the current in-use
		// try IPv4
		ip = ips[0].To4().String()
		if ip == "" {
			// try IPv6
			ip = ips[0].To16().String()
		}
		if ip == "<nil>" {
			ip = "Unknown"
		}
	}

	poweredOn := "unknown"
	if powered != nil {
		if *powered {
			poweredOn = "true"
		} else {
			poweredOn = "false"
		}
	} else {
		logger.Warn().Msgf("node power status not available for '%s'", h.Name)
	}

	hd := common.HostData{
		Name:         h.Name,
		SequenceID:   h.SequenceID,
		HostName:     h.HostName,
		Eth:          h.Eth,
		IP:           ip,
		Mac:          h.Mac,
		State:        h.State.String(),
		Powered:      poweredOn,
		Cluster:      h.Cluster.Name,
		HostPolicy:   h.HostPolicy.Name,
		AccessGroups: groups,
		Restricted:   restricted,
		Reservations: resNames,
	}

	return hd
}

func filterHostList(hostList []Host, filterPowered *bool, user *User) []common.HostData {

	var hostDetails = make([]common.HostData, 0, len(hostList))

	powerMapMU.Lock()
	for _, h := range hostList {

		var hd common.HostData
		if _, ok := powerMap[h.Name]; ok {
			// if the powered boolean search param was included only send hosts that match that
			// power condition, otherwise send everything
			if filterPowered != nil {
				if filterPowered == powerMap[h.Name] {
					hd = h.getHostData(powerMap[h.Name], user)
				} else {
					continue
				}
			} else {
				hd = h.getHostData(powerMap[h.Name], user)
			}
		} else {
			hd = h.getHostData(nil, user)
		}

		hostDetails = append(hostDetails, hd)
	}
	powerMapMU.Unlock()

	// Sort hosts in numeric order
	sort.Slice(hostDetails, func(i, j int) bool {
		return hostDetails[i].SequenceID < hostDetails[j].SequenceID
	})

	return hostDetails
}
