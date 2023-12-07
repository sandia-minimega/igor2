// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"encoding/json"
	"fmt"
	"igor2/internal/pkg/api"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/pflag"

	"igor2/internal/pkg/common"

	"github.com/spf13/cobra"
)

const (
	Up         = "UP"
	Down       = "DOWN"
	PowerNA    = "POWER-N/A"
	Blocked    = "BLOCKED"
	Reserved   = "RESERVED"
	Unreserved = "UNRESERVED"
	Restricted = "RESTRICTED"
	InstallErr = "INST ERROR"
)

func newShowCmd() *cobra.Command {

	cmdShow := &cobra.Command{
		Use: "show [-acefgrtx] [--sort-start --sort-name --sort-owner]\n" +
			"            [-n USER1,... -o OWNER1,...] [--no-color --no-map]",
		Short: "Display current cluster/reservation status",
		Long: `
Displays cluster node statuses and reservation list. 

` + sBold("NODE MAP LEGEND:") + `

Node Power Status (text color):

  ` + cUnreservedUp.Sprint("UP") + `   : power is on
  ` + cUnreservedDown.Sprint(Down) + ` : power is off
  ` + cUnreservedPowerNA.Sprint("N/A") + `  : power status not available

Node Reservation Status (background color):

  ` + cUnreservedUp.Sprint(Unreserved) + `  : node currently free to reserve
  ` + cBlockedUp.Sprint(Blocked) + `     : node not accepting reservations
  ` + cRestrictedUp.Sprint(Restricted) + `  : node has group/time access restriction
  ` + cInstError.Sprint("INSTALL ERR") + ` : reservation failed to install

  ` + cOwnerRes.Sprint("RESERVED") + `    : node reserved by you or accessible via member group
  ` + cOtherRes.Sprint("RESERVED") + `    : node reserved by another user
  ` + cFuture.Sprint("FUTURE") + `      : future reservation (only used on reservation table)

The node map displays current-time status only.

Color output will be auto-disabled if the terminal lacks color support.

` + sBold("NODE MAP TABLE:") + `

A summary view of power and availability of each host in the cluster.

` + sBold("NODE STATUS TABLE:") + `

This table summarizes the status information on the node map to assist color-
blind users.

` + sBold("RESERVATION TABLE:") + `

This table presents a view of the reservations present on the cluster. If the
user has no reservations on the system, they will not see the second table
unless they include the --all flag to see reservations made by others.

Reservations that are scheduled to finish within the next 24 hours have their
end times highlighted.

The INFO column uses a shorthand format for information about the reservation
and is especially useful in combination with the -x flag.

  O: you are the owner
  G: you have group access
  F: future reservation (node column shows nodes to be assigned at startup)
  I: res is installed
  E: res has installation error

` + sBold("ADDITIONAL INFORMATION:") + `

The server's local time will be displayed for reference purposes.
An optional "message of the day" can also appear to provide user alerts and info. 

For more details, run the show command for specific subgroups.
Ex. 'igor res show', 'igor host show'

` + optionalFlags + `

Inclusion : 
  Default list only shows reservations the user owns or can access.
  Use the -a flag to show all reservations in the table listing. Mix with
  filtering flags to narrow results. Example: the combination -a -c will list
  all current reservations and exclude future ones.

Filtering :
  Use the -c -f and -g flags to exclude reservations based on time or group.
  Use the -n flag for partial match filtering on reservation name list.
  Use the -o flag for full match filtering on owner name list.

Sorting :
  Default order is by reservation end time. 
  Use the --sort-* flags to specify a different sort column.
  Use the -r flag to reverse the sort order to descending values.

Formatting :
  Use the -t flag to change the end time date column to time remaining format.
  Use the --no-color flag to suppress color output.
  Use the --no-map flag to suppress the node map.
  Use the -x flag to render screen output without pretty formatting.
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			flagset := cmd.Flags()
			showAll := flagset.Changed("all")
			showCurrentOnly := flagset.Changed("current")
			showFutureOnly := flagset.Changed("future")
			showInstallErrOnly := flagset.Changed("error")
			showGroupOnly := flagset.Changed("group")
			filterResList, _ := flagset.GetStringSlice("filter-name")
			filterOwnerList, _ := flagset.GetStringSlice("filter-owner")
			sortStartTime := flagset.Changed("sort-start")
			sortResName := flagset.Changed("sort-name")
			sortOwnerName := flagset.Changed("sort-owner")

			if sortOwnerName && sortResName || ((sortOwnerName || sortResName) && sortStartTime) {
				return fmt.Errorf("more than one sorting method specified")
			}

			if (showCurrentOnly || showInstallErrOnly) && showFutureOnly {
				return fmt.Errorf("excluded all current and future reservations")
			}

			if len(filterOwnerList) > 0 && len(filterResList) > 0 {
				return fmt.Errorf("can't filter by reservation and owner at same time")
			}

			if showAll && showGroupOnly {
				return fmt.Errorf("show group-only not compatible with show all reservations")
			}

			printShow(doShow(), flagset)
			return nil
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	var noMap,
		remainTime,
		showAll,
		showCurrentOnly,
		showFutureOnly,
		showInstallErrOnly,
		showGroupOnly,
		sortStartTime,
		sortResName,
		sortOwnerName,
		sortReverse bool
	var filterResList,
		filterOwnerList []string

	cmdShow.Flags().BoolVarP(&showAll, "all", "a", false, "show all reservations (includes other users)")
	cmdShow.Flags().BoolVarP(&showCurrentOnly, "current", "c", false, "show current reservations only")
	cmdShow.Flags().BoolVarP(&showFutureOnly, "future", "f", false, "show future reservations only")
	cmdShow.Flags().BoolVarP(&showGroupOnly, "group", "g", false, "show group reservations only")
	cmdShow.Flags().BoolVarP(&showInstallErrOnly, "error", "e", false, "show install-errors only")
	cmdShow.Flags().BoolVar(&noColor, "no-color", false, "do not use color in output")
	cmdShow.Flags().BoolVar(&noMap, "no-map", false, "do not print the node status map")
	cmdShow.Flags().BoolVarP(&remainTime, "time-left", "t", false, "display end time as expiration countdown")
	cmdShow.Flags().BoolVar(&sortStartTime, "sort-start", false, "sort by start time")
	cmdShow.Flags().BoolVar(&sortResName, "sort-name", false, "sort by reservation name")
	cmdShow.Flags().BoolVar(&sortOwnerName, "sort-owner", false, "sort by owner name")
	cmdShow.Flags().BoolVarP(&sortReverse, "reverse", "r", false, "reverse sort order")
	cmdShow.Flags().BoolVarP(&simplePrint, "simple", "x", false, "use simple text output (no color/map/lines)")
	cmdShow.Flags().StringSliceVarP(&filterResList, "filter-name", "n", nil, "partial matching by name")
	cmdShow.Flags().StringSliceVarP(&filterOwnerList, "filter-owner", "o", nil, "matching by owner")

	_ = registerFlagArgsFunc(cmdShow, "filter-name", []string{"NAME1"})
	_ = registerFlagArgsFunc(cmdShow, "filter-owner", []string{"OWNER1"})

	return cmdShow
}

func doShow() *common.ResponseBodyShow {
	body := doSend(http.MethodGet, api.BaseUrl, nil)
	rb := common.ResponseBodyShow{}
	err := json.Unmarshal(*body, &rb)
	checkUnmarshalErr(err)
	return &rb
}

func printShow(rb *common.ResponseBodyShow, flagset *pflag.FlagSet) {

	noColor = flagset.Changed("no-color")
	simplePrint = flagset.Changed("simple")
	noColor = flagset.Changed("no-color")
	noMap := flagset.Changed("no-map")
	showAll := flagset.Changed("all")
	showCurrentOnly := flagset.Changed("current")
	showFutureOnly := flagset.Changed("future")
	showGroupOnly := flagset.Changed("group")
	showInstallErrOnly := flagset.Changed("error")
	filterResList, _ := flagset.GetStringSlice("filter-name")
	filterOwnerList, _ := flagset.GetStringSlice("filter-owner")
	sortStartTime := flagset.Changed("sort-start")
	sortResName := flagset.Changed("sort-name")
	sortOwnerName := flagset.Changed("sort-owner")
	sortReverse := flagset.Changed("reverse")
	remainTime := flagset.Changed("time-left")

	checkAndSetColorLevel(rb)

	if len(rb.Data["show"].Hosts) == 0 || rb.Data["show"].Cluster.Name == "" {
		color.FgYellow.Printf("igor: a cluster must be configured first\n")
		os.Exit(0)
	}

	showData := rb.Data["show"]

	maxResNameLength := len(Unreserved) + 1 // start with this since it's always included
	oneYearLater := igorCliNow.Add(time.Hour * 24 * 365).Unix()
	const MaxNodeColWidth = 80

	// Filtering (and time format since we can do that here too)
	var yearFmt string
	var inclResList []common.ReservationData
	var seedResList []common.ReservationData
	var installErrorNodes []string

	// apply broad filter-by-matching methods first to whole res list
	if len(filterOwnerList) > 0 {
		for _, o := range filterOwnerList {
			for _, r := range showData.Reservations {
				if o == r.Owner {
					seedResList = append(seedResList, r)
				}
			}
		}
	} else if len(filterResList) > 0 {
		for _, n := range filterResList {
			for _, r := range showData.Reservations {
				if strings.Contains(r.Name, n) {
					seedResList = append(seedResList, r)
				}
			}
		}
	} else {
		seedResList = showData.Reservations
	}

	// make a list of reservations that the user has group-write access to
	groupResList := make([]common.ReservationData, 0)
	for _, r := range seedResList {

		for _, g := range showData.UserGroups {
			if r.Group == g && g != "all" {
				groupResList = append(groupResList, r)
			}
		}

		// if any reservation lasts a year from now, modify the timestamp
		// highly unlikely so just do it here rather than user another loop
		if r.End > oneYearLater {
			yearFmt = "2006 "
		}
	}

	// method to determine if the group res is included in final list
	isGroupRes := func(res common.ReservationData) bool {
		if len(groupResList) > 0 {
			for _, g := range groupResList {
				if res.Group == g.Group {
					return true
				}
			}
		}
		return false
	}

	for _, r := range seedResList {

		inclRes := false
		resStart := time.Unix(r.Start, 0).Local()

		if showAll {
			if showCurrentOnly {
				if resStart.Before(igorCliNow) {
					inclRes = true
				}
			} else if showFutureOnly {
				if !resStart.Before(igorCliNow) {
					inclRes = true
				}
			} else if showInstallErrOnly {
				if r.InstallError != "" {
					inclRes = true
				}
			} else {
				inclRes = true
			}
		} else {
			if showCurrentOnly {
				if resStart.Before(igorCliNow) {
					if !showGroupOnly {
						if r.Owner == lastAccessUser || isGroupRes(r) {
							inclRes = true
						}
					} else if isGroupRes(r) {
						inclRes = true
					}
				}
			} else if showFutureOnly {
				if !resStart.Before(igorCliNow) {
					if !showGroupOnly {
						if r.Owner == lastAccessUser || isGroupRes(r) {
							inclRes = true
						}
					} else if isGroupRes(r) {
						inclRes = true
					}
				}
			} else if showInstallErrOnly {
				if r.InstallError != "" {
					if !showGroupOnly {
						if r.Owner == lastAccessUser || isGroupRes(r) {
							inclRes = true
						}
					} else if isGroupRes(r) {
						inclRes = true
					}
				}
			} else {
				if r.Owner == lastAccessUser || isGroupRes(r) {
					inclRes = true
				}
			}
		}

		if inclRes {
			inclResList = append(inclResList, r)
			if r.InstallError != "" {
				installErrorNodes = append(installErrorNodes, r.Hosts...)
			}
		}
	}

	// sort the reservations
	if sortStartTime {
		sort.Sort(byStartTime(inclResList))
	} else if sortResName {
		sort.Sort(byResName(inclResList))
	} else if sortOwnerName {
		sort.Sort(byOwner(inclResList))
	} else {
		sort.Sort(byEndTime(inclResList))
	}

	if sortReverse {
		for i := len(inclResList)/2 - 1; i >= 0; i-- {
			opp := len(inclResList) - 1 - i
			inclResList[i], inclResList[opp] = inclResList[opp], inclResList[i]
		}
	}

	// Go through each reservation's host list and map each one's list of reserved nodes
	resNodes := map[int]bool{}
	for _, r := range inclResList {
		// Remember the longest reservation name for formatting later on
		if maxResNameLength < len(r.Name)+1 {
			maxResNameLength = len(r.Name) + 1
		}

		if r.Installed {
			for _, h := range r.Hosts {
				v, err := strconv.Atoi(h[len(showData.Cluster.Prefix):])
				if err != nil {
					//that's weird
					continue
				}
				resNodes[v] = true
			}
		}
	}

	instErrMap := map[int]bool{}
	for _, ien := range installErrorNodes {
		v, _ := strconv.Atoi(ien[len(showData.Cluster.Prefix):])
		instErrMap[v] = true
	}

	restrictMap := make(map[int]bool)
	nameFmt := "%" + strconv.Itoa(maxResNameLength) + "v"
	monthFmt := "Jan "
	dayYearFmt := "2 " + yearFmt
	timeFmt := "3:04 PM"
	if simplePrint {
		monthFmt = "Jan-"
		dayYearFmt = "02-06."
		timeFmt = "15:04"
	}

	// Gather lists of which nodes are blocked, restricted and unreserved
	var unreservedNodes []string
	var blockedNodes []string
	var restrictedNodes []string

	for i := 0; i < len(showData.Hosts); i++ {
		h := &showData.Hosts[i]
		if h.Restricted {
			restrictedNodes = append(restrictedNodes, h.Name)
			restrictMap[h.SequenceID] = true
		}
		if h.State == strings.ToLower(Blocked) {
			blockedNodes = append(blockedNodes, h.Name)
		} else if h.State == strings.ToLower(Reserved) {
			continue
		} else if !resNodes[i+1] {
			unreservedNodes = append(unreservedNodes, h.Name)
		}
	}

	adjServerTime := rb.ServerTime
	if cli.tzLoc != nil {
		sTime, _ := time.Parse(common.DateTimeServerFormat, rb.ServerTime)
		aTime := getLocTime(sTime)
		adjServerTime = aTime.Format(common.DateTimeServerFormat)
	}

	// print out the node table
	if noMap || simplePrint {
		fmt.Printf("\nCluster Name : %v\n", strings.ToTitle(showData.Cluster.Name))
		fmt.Printf("Prefix       : %v\n", showData.Cluster.Prefix)
		fmt.Printf("Total Nodes  : %d\n", len(showData.Hosts))
	} else {
		printNodeMap(showData.Cluster, showData.Hosts, showData.Reservations, showData.UserGroups, restrictMap, instErrMap)
	}

	fmt.Println("")
	// Print node status table
	nst := table.NewWriter()
	nst.AppendHeader(table.Row{"STATUS", "#", "NODES"})

	statusFormat := "%" + strconv.Itoa(len(Unreserved)) + "v"
	if len(installErrorNodes) > 0 {
		statusFormat = "%" + strconv.Itoa(len(InstallErr)) + "v"
	}

	rowHeaderName := func(style color.PrinterFace, name string) string {
		return style.Sprintf(statusFormat, name)
	}
	if simplePrint {
		rowHeaderName = func(_ color.PrinterFace, name string) string {
			return name
		}
	}

	makeNodeRow := func(nodes []string, style *color.Style256, rowType string) {

		r := common.Range{
			Prefix: showData.Cluster.Prefix,
			Min:    showData.Hosts[0].SequenceID,
			Max:    showData.Hosts[len(showData.Hosts)-1].SequenceID,
		}
		nodeRange, _ := r.UnsplitRange(nodes)

		nodeLine := multilineRange(MaxNodeColWidth, nodeRange, showData.Cluster.Prefix)

		nst.AppendRow([]interface{}{
			rowHeaderName(style, rowType),
			strconv.Itoa(len(nodes)),
			nodeLine,
		})
		// we'll use max col width on the table to do row-based text-wrapping
	}

	makeNodeRow(unreservedNodes, cUnreservedUp, Unreserved)
	makeNodeRow(blockedNodes, cBlockedUp, Blocked)

	if len(restrictedNodes) > 0 {
		makeNodeRow(restrictedNodes, cRestrictedUp, Restricted)
	}

	if len(installErrorNodes) > 0 {
		makeNodeRow(installErrorNodes, cInstError, InstallErr)
	}

	if simplePrint {
		nst.Style().Options.SeparateRows = false
		nst.Style().Options.SeparateColumns = false
	} else {
		nst.SetStyle(table.StyleLight)
		nst.SetColumnConfigs([]table.ColumnConfig{
			{Name: "STATUS", AlignHeader: text.AlignRight, Align: text.AlignRight},
			{Name: "NODES", WidthMax: MaxNodeColWidth},
		})
	}

	nst.Style().Options.DrawBorder = false
	fmt.Println(nst.Render())

	fmt.Println("\nServer Time : " + adjServerTime)
	if strings.TrimSpace(showData.Cluster.Motd) != "" {
		printMotd(showData.Cluster)
	} else {
		fmt.Println("")
	}

	if len(inclResList) == 0 {
		if len(filterOwnerList) > 0 || len(filterResList) > 0 {
			fmt.Println(sBold("\nNo reservations returned by this query.\n"))
		} else {
			var noRes = "You have no owned or group-affiliated reservations."
			if showFutureOnly {
				noRes = "You have no future owned or group-affiliated reservations."
			}
			var extra = ""
			if !showAll {
				extra = "(Use the --all flag to see complete reservation list.)"
			}
			fmt.Println(sBold(fmt.Sprintf("\n%s %s\n", noRes, extra)))
		}
		return
	}

	// Print reservation list
	tw := table.NewWriter()
	tw.AppendHeader(table.Row{"NAME", "OWNER", "START", "END", "INFO", "#", "NODES"})

	for _, r := range inclResList {

		resStart := getLocTime(time.Unix(r.Start, 0))
		resEnd := getLocTime(time.Unix(r.End, 0))

		var flags string

		if r.Owner == lastAccessUser {
			flags += "O"
		} else if isGroupRes(r) {
			flags += "G"
		}

		if resStart.After(igorCliNow) {
			flags += "F"
		} else {
			if r.InstallError != "" {
				flags += "E"
			} else {
				flags += "I"
			}
		}

		var name string
		if simplePrint {
			name = r.Name
		} else {
			if r.InstallError != "" {
				name = cInstError.Sprintf(nameFmt, r.Name)
			} else if r.Owner == lastAccessUser || isGroupRes(r) {
				if resStart.Before(igorCliNow) {
					name = cOwnerRes.Sprintf(nameFmt, r.Name)
				} else {
					name = cFuture.Sprintf(nameFmt, r.Name)
				}
			} else {
				if resStart.Before(igorCliNow) {
					name = cOtherRes.Sprintf(nameFmt, r.Name)
				} else {
					name = cFuture.Sprintf(nameFmt, r.Name)
				}
			}
		}

		var endTimeStr string
		if !simplePrint {
			monthStr := resEnd.Format(monthFmt)
			dayYearStr := resEnd.Format(dayYearFmt)
			if strings.Index(dayYearStr, " ") == 1 {
				dayYearStr = " " + dayYearStr
			}
			timeStr := resEnd.Format(timeFmt)
			if strings.Index(timeStr, ":") == 1 {
				timeStr = " " + timeStr
			}
			endTimeStr = monthStr + dayYearStr + timeStr
		} else {
			endTimeStr = resEnd.Format(monthFmt + dayYearFmt + timeFmt)
		}

		durRemaining := resEnd.Sub(igorCliNow)
		if remainTime {
			endTimeStr = common.FormatDuration(durRemaining.Round(time.Minute), true)
		}
		if durRemaining < 12*time.Hour {
			endTimeStr = cAlert.Sprintf(endTimeStr)
		} else if durRemaining < 24*time.Hour && durRemaining >= 12*time.Hour {
			endTimeStr = cWarning.Sprintf(endTimeStr)
		}

		var startTimeStr string
		if !simplePrint {
			monthStr := resStart.Format(monthFmt)
			dayYearStr := resStart.Format(dayYearFmt)
			if strings.Index(dayYearStr, " ") == 1 {
				dayYearStr = " " + dayYearStr
			}
			timeStr := resStart.Format(timeFmt + " MST")
			if strings.Index(timeStr, ":") == 1 {
				timeStr = " " + timeStr
			}
			startTimeStr = monthStr + dayYearStr + timeStr
		} else {
			startTimeStr = resStart.Format(monthFmt + dayYearFmt + timeFmt)
		}

		var hostStatus = ""

		if resStart.After(igorCliNow) {
			hostStatus = cFutureNodes.Sprint(r.HostRange)
		} else {
			if len(r.HostsDown) == 0 && len(r.HostsPowerNA) == 0 {
				hostStatus = cOK.Sprintf(Up) + " " + r.HostRange
			} else if len(r.HostsUp) == 0 && len(r.HostsPowerNA) == 0 {
				hostStatus = cAlert.Sprintf(Down) + " " + r.HostsDown
			} else if len(r.HostsUp) == 0 && len(r.HostsDown) == 0 {
				hostStatus = cWarning.Sprintf(PowerNA) + " " + r.HostsPowerNA
			} else {

				var up, down, na string
				if len(r.HostsUp) > 0 {
					up = fmt.Sprintf(" %s %s /", cOK.Sprintf(Up), r.HostsUp)
				}
				if len(r.HostsPowerNA) > 0 {
					na = fmt.Sprintf(" %s %s /", cWarning.Sprintf(PowerNA), r.HostsPowerNA)
				}
				if len(r.HostsDown) > 0 {
					down = fmt.Sprintf(" %s %s /", cAlert.Sprintf(Down), r.HostsDown)
				}

				hostStatus = strings.TrimSuffix(fmt.Sprintf("%s -%s%s%s", r.HostRange, up, na, down), " /")
			}
		}

		tw.AppendRow([]interface{}{
			name,
			r.Owner,
			startTimeStr,
			endTimeStr,
			flags,
			strconv.Itoa(len(r.Hosts)),
			hostStatus,
		})
	}

	if simplePrint {
		tw.Style().Options.SeparateRows = false
		tw.Style().Options.SeparateColumns = false
		tw.SetColumnConfigs([]table.ColumnConfig{
			{Name: "END", AlignHeader: text.AlignLeft, Align: text.AlignRight},
		})
	} else {
		tw.SetStyle(table.StyleLight)
		tw.SetTitle("RESERVATIONS")
		tw.Style().Title.Align = text.AlignCenter
		tw.Style().Title.Format = text.FormatUpper
		tw.Style().Title.Colors = text.Colors{text.Bold, text.Faint}
		tw.SetColumnConfigs([]table.ColumnConfig{
			{Name: "NAME", AlignHeader: text.AlignRight, Align: text.AlignRight},
			{Name: "START", AlignHeader: text.AlignLeft, Align: text.AlignRight},
			{Name: "END", AlignHeader: text.AlignLeft, Align: text.AlignRight},
		})
	}

	tw.Style().Options.DrawBorder = false

	fmt.Println(tw.Render())
}

func printNodeMap(cData common.ClusterData, hData []common.HostData, rData []common.ReservationData, userGroups []string, restricted map[int]bool, instErr map[int]bool) {
	// figure out how many digits we need per node displayed
	lastNode := hData[len(hData)-1].SequenceID
	nodeWidth := len(strconv.Itoa(lastNode))
	nodeFmt := "%" + strconv.Itoa(nodeWidth) + "v"

	hDataMap := make(map[int]*common.HostData)
	for i := range hData {
		hDataMap[hData[i].SequenceID] = &hData[i]
	}

	hDataKeys := make([]int, 0, len(hDataMap))
	for k := range hDataMap {
		hDataKeys = append(hDataKeys, k)
	}
	sort.Ints(hDataKeys)
	totalNodes := len(hDataKeys)

	// a mapping of hosts to reservations where key = host number and val = index in ReservationData array
	n2r := map[int]int{}
	for i, r := range rData {
		resStart := getLocTime(time.Unix(r.Start, 0))
		if resStart.Before(igorCliNow) {
			for _, hostName := range r.Hosts {
				n := strings.TrimPrefix(hostName, cData.Prefix)
				seqID, err := strconv.Atoi(n)
				if err == nil {
					n2r[seqID] = i
				}
			}
		}
	}

	tw := table.NewWriter()
	tw.SetTitle(cData.Name)

	n := 0
	for i := 0; i < cData.DisplayHeight; i++ {

		var row table.Row
		for j := cData.DisplayWidth*i + 1; j <= cData.DisplayWidth*i+cData.DisplayWidth; j++ {

			colorNode := color.S256()
			if j <= totalNodes {

				// this is the host sequence id
				seqID := hDataKeys[n]

				// color the numbers based on node power status
				if hDataMap[seqID].Powered == "true" {
					colorNode.SetFg(FgUp)
				} else if hDataMap[seqID].Powered == "false" {
					colorNode.SetFg(FgDown).AddOpts(color.Bold)
				} else {
					colorNode.SetFg(FgPowerNA).AddOpts(color.Bold)
				}

				name := fmt.Sprintf(nodeFmt, seqID)

				if instErr[seqID] {
					// show node background as error state
					row = append(row, colorNode.SetBg(BgError).AddOpts(color.Bold).Sprint(name))
				} else if resIndex, ok := n2r[seqID]; ok {

					// set node background based on user reservation access
					res := rData[resIndex]
					isGroupRes := false
					for _, g := range userGroups {
						if res.Group == g {
							isGroupRes = true
						}
					}
					if res.Owner == lastAccessUser || isGroupRes {
						colorNode.SetBg(BgResYes)
						row = append(row, colorNode.Sprint(name))
					} else {
						colorNode.SetBg(BgResNo)
						row = append(row, colorNode.Sprint(name))
					}

				} else if hDataMap[seqID].State == "blocked" {
					// set node background for blocked
					row = append(row, colorNode.SetBg(BgBlocked).AddOpts(color.Bold).Sprint(name))
				} else if restricted[seqID] {
					// set node background for restricted
					row = append(row, colorNode.SetBg(BgRestricted).Sprintf(name))
				} else {
					// and finally nodes that are reservable
					row = append(row, colorNode.SetBg(BgUnreserved).Sprint(name))
				}

				n++

			} else {
				row = append(row, colorNode.SetFg(FgUp).SetBg(BgUnreserved).Sprint(fmt.Sprintf(nodeFmt, "")))
			}
		}

		tw.AppendRow(row)
	}

	tw.SetStyle(table.StyleLight)
	tw.Style().Title.Align = text.AlignCenter
	tw.Style().Title.Format = text.FormatUpper
	tw.Style().Title.Colors = text.Colors{text.Bold, text.Faint}
	tw.Style().Box.PaddingLeft = ""
	tw.Style().Box.PaddingRight = ""
	fmt.Println(tw.Render())
}

func printMotd(clusterData common.ClusterData) {

	finalMotd := "\nMOTD: "
	if (simplePrint || noColor || envNoColor || color.TermColorLevel() == color.LevelNo) && clusterData.MotdUrgent {
		finalMotd += " IMPORTANT! - "
	}

	finalMotd += clusterData.Motd + "\n\n"

	if clusterData.MotdUrgent {
		cMotdUrgent.Printf(finalMotd)
	} else {
		cMotdNotUrgent.Printf(finalMotd)
	}
}

type byStartTime []common.ReservationData

func (resList byStartTime) Len() int      { return len(resList) }
func (resList byStartTime) Swap(i, j int) { resList[i], resList[j] = resList[j], resList[i] }
func (resList byStartTime) Less(i, j int) bool {
	return resList[i].Start < resList[j].Start
}

type byEndTime []common.ReservationData

func (resList byEndTime) Len() int      { return len(resList) }
func (resList byEndTime) Swap(i, j int) { resList[i], resList[j] = resList[j], resList[i] }
func (resList byEndTime) Less(i, j int) bool {
	return resList[i].End < resList[j].End
}

type byOwner []common.ReservationData

func (resList byOwner) Len() int      { return len(resList) }
func (resList byOwner) Swap(i, j int) { resList[i], resList[j] = resList[j], resList[i] }
func (resList byOwner) Less(i, j int) bool {
	return resList[i].Owner < resList[j].Owner
}

type byResName []common.ReservationData

func (resList byResName) Len() int      { return len(resList) }
func (resList byResName) Swap(i, j int) { resList[i], resList[j] = resList[j], resList[i] }
func (resList byResName) Less(i, j int) bool {
	return resList[i].Name < resList[j].Name
}
