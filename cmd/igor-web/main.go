// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package main

import (
	"flag"
	"fmt"
	igorweb "igor2/internal/app/igor-web"
	"igor2/internal/pkg/common"
	"os"
)

var (
	configFilepath = flag.String("config", "", "path to configuration file")
	version        = flag.Bool("v", false, "version info")
)

func main() {
	flag.Parse()

	if *version {
		fmt.Println(common.GetVersion("IgorWeb Server", false))
		os.Exit(0)
	}

	igorweb.Execute(configFilepath)
}
