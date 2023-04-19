// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"math/rand"
	"os"
	"strings"
	"time"

	"igor2/internal/pkg/common"
)

var igor = &Igor{}

// Execute is the entry point from the main package and handles startup operations
func Execute(configFilepath *string) {

	igor.Started = time.Now()

	if igor.IgorHome = os.Getenv("IGOR_HOME"); strings.TrimSpace(igor.IgorHome) == "" {
		exitPrintFatal("environment variable IGOR_HOME not defined")
	}

	// seed rand for hash generator (default naming)
	rand.Seed(igor.Started.UTC().UnixNano())

	initConfig(*configFilepath)
	initLog()
	initConfigCheck()
	initNotify()

	igor.ElevateMap = common.NewPassiveTtlMap(time.Duration(igor.Auth.ElevateTimeout) * time.Minute)

	igor.IPowerStatus = NewNmapPowerStatus()

	// set IResInstaller to tftp
	// we may eventually give them a choice (cobbler, etc.)
	igor.IResInstaller = NewTFTPInstaller()

	initDbBackend()
	initAuth()

	hostList, err := dbReadHostsTx(map[string]interface{}{})
	if err != nil {
		exitPrintFatal(err.Error())
	}

	// need to check igor config to see if nodes have been added or removed
	syncNodes(hostList)

	if len(hostList) > 0 {
		wg.Add(1)
		go powerStatusManager(hostList)
	}

	// This call will not return until the server terminates
	runServer()
}
