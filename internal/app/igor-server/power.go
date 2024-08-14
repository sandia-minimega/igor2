// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"gorm.io/gorm"
	"net"
	"sync"
	"time"
)

var (
	// powerMap is storage for power status of a node. true = on, false =  off, nil = unknown
	powerMap   map[string]*bool
	ipMap      map[string]string
	powerMapMU sync.Mutex
)

// IPowerStatus is an interface that provides methods for an external application to fetch power
// information about cluster nodes.
type IPowerStatus interface {
	// updateHosts gathers power information about the slice of hosts provided and updates powerMap
	// with the results.
	updateHosts(hosts []Host)
}

// powerStatusManager is called as a go routine and polls the assigned IPowerStatus more frequently
// when clients are active and less frequently otherwise
func powerStatusManager(hosts []Host) {
	defer wg.Done()

	ipMap = make(map[string]string, len(hosts))
	for _, h := range hosts {
		ip := h.IP
		if ip == "" {
			// we need to get an IP for this node
			ips, err := net.LookupIP(h.HostName)
			if err != nil {
				// error getting host IP, skip
				continue
			}
			ip = ips[0].To4().String()
			if ip == "" {
				// try IPv6
				ip = ips[0].To16().String()
			}
			if ip == "<nil>" {
				// error getting host IP, skip
				continue
			} else {
				// update the host record with the new IP
				logger.Warn().Msgf("updating %s with new host IP: %s", h.Name, ip)
				if err = performDbTx(func(tx *gorm.DB) error {
					if err = dbEditHosts([]Host{h}, map[string]interface{}{"ip": ip}, tx); err != nil {
						return err
					}
					return nil
				}); err != nil {
					logger.Error().Msgf("problem updating host IP: %v", err)
				}
			}
		}
		ipMap[ip] = h.HostName
	}

	logger.Debug().Msgf("%v", ipMap)

	startup := 10 * time.Millisecond
	timeoutFast := 3 * time.Second
	timeoutSlow := 10 * time.Second // during no user activity, reduce call frequency
	timeout := timeoutFast
	fastRefreshes := 20
	countdown := time.NewTimer(startup)
	hostNames := hostNamesOfHosts(hosts)
	powerMap = make(map[string]*bool, len(hostNames))
	for _, h := range hostNames {
		powerMap[h] = nil
	}

	for {
		select {
		case <-shutdownChan:
			logger.Info().Msg("stopping node power status background worker")
			if !countdown.Stop() {
				<-countdown.C
			}
			return
		case <-refreshPowerChan:
			if fastRefreshes == 0 {
				if !countdown.Stop() {
					<-countdown.C
				}
				// when user activity starts after slow period, refresh immediately
				countdown.Reset(startup)
			}
			fastRefreshes = 20
		case <-countdown.C:
			if fastRefreshes == 0 {
				logger.Debug().Msg("slow nmap power update")
				timeout = timeoutSlow
			} else {
				logger.Debug().Msgf("fast nmap power update - countdown %d", fastRefreshes)
				timeout = timeoutFast
				fastRefreshes--
			}

			igor.IPowerStatus.updateHosts(hosts)
			countdown.Reset(timeout)
		}
	}
}
