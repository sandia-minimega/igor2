// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package common

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const Copyright = "\u00A92023 NTESS, LLC"

var (
	Date      string
	GitTag    string
	GoVersion string
)

// GetVersion returns version information about igor apps that can be
// seen by running an app with a version flag or formatted into a single
// line for inclusion in logger output.
func GetVersion(appName string, asLog bool) string {

	const dtsFormat = "02-Jan-2006 15:04:05 MST"
	var tags = []string{appName + ":", "Go-Version:", "Build-Date:", "Copyright:"}
	var vals = []string{GitTag, GoVersion, Date, Copyright}

	if len(GoVersion) == 0 {
		// This path should execute in an IDE setting if no ldflags have been set up in the run profile
		if bInfo, ok := debug.ReadBuildInfo(); !ok {
			vals[0] = "no version info"
			vals[1] = runtime.Version()[2:]
			vals[2] = time.Now().UTC().Format(dtsFormat)
		} else {
			vals[1] = runtime.Version()[2:]
			for _, setting := range bInfo.Settings {
				if setting.Key == "vcs.revision" {
					vals[0] = setting.Value[0:8] + " (commit hash)"
				} else if setting.Key == "vcs.time" {
					dt, _ := time.Parse(time.RFC3339, setting.Value)
					vals[2] = dt.Format(dtsFormat)
				}
			}
		}
	}

	var buildInfo = ""

	if asLog {
		for i := 0; i < 2; i++ {
			buildInfo += tags[i] + " " + vals[i] + "; "
		}
		buildInfo += tags[2] + " " + vals[2]
	} else {
		var n = 0
		for _, t := range tags {
			if len(t) > n {
				n = len(t)
			}
		}
		var format = "%" + strconv.Itoa(n) + "s"
		for i := range tags {
			buildInfo += fmt.Sprintf(format, tags[i])
			buildInfo += " " + vals[i] + "\n"
		}
	}

	return strings.TrimSuffix(buildInfo, "\n")
}
