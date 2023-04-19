// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

//go:build !DEVMODE

package igorserver

import (
	"bufio"
	"bytes"
	"os/exec"
	"strconv"
	"strings"
)

type NmapPowerStatus struct{}

func NewNmapPowerStatus() IPowerStatus {
	return &NmapPowerStatus{}
}

func (nr *NmapPowerStatus) updateHosts(hosts []Host) {

	if len(powerMap) == 0 {
		logger.Warn().Msg("powerMap has no record of configured hosts")
		return
	}

	if len(hosts) == 0 {
		logger.Warn().Msg("no hosts provided on call to updateHosts")
		return
	}

	hostNames := namesOfHosts(hosts)

	// create a slice of just the host hostnames
	hostHNames := hostNamesOfHosts(hosts)

	// Use nmap to determine what nodes are up
	var args []string
	if igor.Server.DNSServer != "" {
		args = append(args, "--dns-servers", igor.Server.DNSServer)
	}
	args = append(args,
		"-sn",
		"-PS22",
		"--unprivileged",
		"-T5", // super-aggressive scan time. adjust for slower network
		//"--max-rtt-timeout 1s", // adjust timeout if slower network
		"--min-parallelism", // scan all the nodes in parallel
		strconv.Itoa(len(hostHNames)),
		"-oG",
		"-",
	)
	cmd := exec.Command("nmap", append(args, hostHNames...)...)
	logger.Debug().Msgf("running command: %v", cmd)
	out, err := cmd.Output()

	s := bufio.NewScanner(bytes.NewReader(out))
	powerMapMU.Lock()

	if err != nil {
		logger.Warn().Msgf("nmap encountered a problem: %v", err)
		for _, h := range hostNames {
			powerMap[h] = nil
		}
	} else {
		for _, h := range hostNames {
			tmpFalse := false
			powerMap[h] = &tmpFalse
		}
	}

	// Parse the results of nmap

	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		logger.Trace().Msgf("nmap next line: %v\n", line)
		fields := strings.Fields(line)
		if len(fields) != 5 {
			// that's weird
			continue
		}

		// grab the IP address
		ip := fields[1]
		// If we found a node's IP in the output, that means it's reachable,
		// so get its host name from the ipMap and mark it as up
		if name, ok := ipMap[ip]; ok {
			tmpTrue := true
			powerMap[name] = &tmpTrue
			logger.Trace().Msgf("set host %s power to %v", name, *powerMap[name])
		}
	}

	powerMapMU.Unlock()
}
