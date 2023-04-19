// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"igor2/internal/pkg/api"

	"igor2/internal/pkg/common"

	"github.com/spf13/cobra"
)

func newStatsCmd() *cobra.Command {

	cmdStats := &cobra.Command{
		Use:   "stats [-o OPTION] [-s START] [-d DURATION] [-v]",
		Short: "Report canned stats for igor " + adminOnly,
		Long: `
Displays stats and information based on igor's reservation history. The start
point is always the time the stats command is called (now). The duration is
the last 7 days from start by default. Both the start point and the duration
can be specified.

` + optionalFlags + `

Use the -o flag to set an option. The only option for now is default.

Use the -s flag to set the start time for stats. It represents the latest time 
in the window. Use the format 2021-Jan-02. The duration will count backwards
starting from this time.

Use the -d flag to set an integer value of the number of days going back from
the start point the stats should be captured from. The default is 7 days. A
value of 0 will include the entire history up to the start point.

Use the -v flag can be specified for verbose output, showing additional stat
usage breakdown by user.

` + adminOnlyBanner + ``,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			option, _ := flagset.GetString("option")
			verbose := flagset.Changed("verbose")
			start, _ := flagset.GetString("start")
			dur, _ := flagset.GetString("duration")
			result := doStats(option, start, dur, verbose)
			printStats(result)
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	var option string
	var start string
	var dur string
	var verbose bool

	cmdStats.Flags().StringVarP(&option, "option", "o", "", "option to use for stats")
	cmdStats.Flags().BoolVarP(&verbose, "verbose", "v", false, "include stats per each user")
	cmdStats.Flags().StringVarP(&start, "start", "s", "", "the latest point in the stats time window")
	cmdStats.Flags().StringVarP(&dur, "duration", "d", "", "the number of days back from start the stats window should span")
	_ = registerFlagArgsFunc(cmdStats, "option", []string{"OPTION"})
	_ = registerFlagArgsFunc(cmdStats, "start", []string{"START"})
	_ = registerFlagArgsFunc(cmdStats, "duration", []string{"DURATION"})

	return cmdStats
}

func doStats(option, start string, dur string, verbose bool) *common.ResponseBodyStats {
	params := ""
	if option != "" {
		params += "option=" + option + "&"
	}
	if start != "" {
		params += "start=" + start + "&"
	}
	if dur != "" {
		params += "duration=" + dur + "&"
	}
	if verbose {
		params += "verbose=true" + "&"
	}
	if params != "" {
		params = strings.TrimSuffix(params, "&")
		params = "?" + params
	}

	apiPath := api.Stats + params
	body := doSend(http.MethodGet, apiPath, nil)
	rb := common.ResponseBodyStats{}
	err := json.Unmarshal(*body, &rb)
	checkUnmarshalErr(err)
	return &rb
}

func printStats(rb *common.ResponseBodyStats) {
	if !rb.IsSuccess() {
		printRespSimple(rb)
	}

	data := rb.Data["stats"]
	fmt.Printf("Option: %v\n", data.Option)
	fmt.Printf("Start Time: %v\n", data.Start)
	fmt.Printf("End Time: %v\n", data.End)

	if data.Verbose {
		fmt.Printf("\nBy User:\n")
		for user, stats := range data.ByUser {
			fmt.Printf("\nUser - %s:\nReservations:\n", user)
			// Sort entries by res start time, keeping original order or equal elements.
			sort.SliceStable(stats.Entries, func(i, j int) bool {
				return stats.Entries[i].Start.Before(stats.Entries[j].Start)
			})
			for _, r := range stats.Entries {
				fmt.Printf("Name: %v\tRes ID: %v\n", r.Name, r.Hash)
				fmt.Printf("Nodes: %v\n", r.Hosts)
				fmt.Printf("start: %v\toriginal end: %v\tactual end: %v\t # extensions: %v\n\n", r.Start, r.OrigEnd, r.End, r.ExtendCount)
			}
			fmt.Printf("%v Summary:\n", user)
			fmt.Printf("Reservation Count: %v\n", stats.ResCount)
			fmt.Printf("Nodes Used (not unique): %v\n", stats.NodesUsedCount)
			fmt.Printf("Reservations Cancelled early: %v\n", stats.CancelledEarly)
			fmt.Printf("Extensions used: %v\n", stats.NumExtensions)
			fmt.Printf("Total Reservation Time: %v\n", stats.TotalResTime)
		}
	}

	fmt.Printf("\nGlobal:\n")
	fmt.Printf("Reservation Count: %v\n", data.Global.ResCount)
	fmt.Printf("Nodes Used (not unique): %v\n", data.Global.NodesUsedCount)
	fmt.Printf("Reservations Cancelled early: %v\n", data.Global.CancelledEarly)
	fmt.Printf("Extensions used: %v\n", data.Global.NumExtensions)
	fmt.Printf("Total Reservation Time: %v\n", data.Global.TotalResTime)

}
