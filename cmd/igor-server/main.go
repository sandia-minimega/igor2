// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package main

import (
	"flag"
	"fmt"
	igorserver "igor2/internal/app/igor-server"
	"igor2/internal/pkg/common"
	"os"
)

// Global Variables
var (
	configFilepath = flag.String("config", "", "path to configuration file")
	version        = flag.Bool("v", false, "version info")
)

func main() {
	flag.Parse()

	if *version {
		fmt.Println(common.GetVersion("Igor Server", false))
		os.Exit(0)
	}

	igorserver.Execute(configFilepath)
}
