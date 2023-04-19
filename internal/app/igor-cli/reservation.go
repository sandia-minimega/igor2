// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"encoding/json"
	"fmt"
	"igor2/internal/pkg/api"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"

	"igor2/internal/pkg/common"

	"github.com/spf13/cobra"
)

func exGetStart() time.Time {
	return time.Date(igorCliNow.Year(), igorCliNow.Month(), igorCliNow.Day(),
		8, 0, 0, 0, time.Local).AddDate(0, 0, 5)
}

func exStartDts() string {
	return exGetStart().Format(common.DateTimeCompactFormat)
}

func exStartDay() string {
	return exGetStart().Format("Jan 2, 3:04 PM")
}

func exGetEnd() time.Time {
	return time.Date(igorCliNow.Year(), igorCliNow.Month(), igorCliNow.Day(),
		17, 0, 0, 0, time.Local).AddDate(0, 0, 15)
}

func exEndDts() string {
	return exGetEnd().Format(common.DateTimeCompactFormat)
}

func exEndDay() string {
	return exGetEnd().Format("Jan 2, 3:04 PM")
}

func newResCmd() *cobra.Command {

	cmdRes := &cobra.Command{
		Use:   "res",
		Short: "Perform a reservation command",
		Long: `
Reservation primary command. A sub-command must be invoked to do anything.

A reservation is the most common element users encounter in igor. It defines a
set of nodes on the cluster that are reserved by a user for a given length of 
time. The nodes in the reservation will boot an OS image with optional start up
parameters.

Reservations do not enforce access control to nodes at the network level; it is
possible for others to reach your nodes if you do not have OS accounts set up
and properly configured on the image you are using to boot your reserved nodes.
Consult with your cluster admin team for further guidance.
`,
	}

	cmdRes.AddCommand(newResCreateCmd())
	cmdRes.AddCommand(newResShowCmd())
	cmdRes.AddCommand(newResEditCmd())
	cmdRes.AddCommand(newResDelCmd())

	return cmdRes
}

func newResCreateCmd() *cobra.Command {

	cmdCreateRes := &cobra.Command{
		Use: "create NAME -n NODES {-p PROFILE | -d DISTRO} [-s START -e END \n" +
			"           -g GROUP -v VLAN -k \"KARGS\" --desc \"DESCRIPTION\" --no-cycle\n" +
			"           (-o OWNER)]",
		Short: "Create a reservation",
		Long: `
Create a reservation on one or more cluster nodes. A reservation requires a
profile or distro to boot the requested nodes, and the requesting user must 
have access rights to the distro. Run 'igor profile show' or 'igor distro show'
to see lists of available resources. Use the help command on each for more
information.

` + requiredArgs + `

  NAME : reservation name

` + requiredFlags + `

  -n NODES : the desired number, name list or range of hosts
    * the number of nodes (igor chooses): 4
    * name list is comma-delimited: kn1,kn2,kn3,...
    * range is the form prefix[n,m-n,...] where m,n are integers representing
      a single or contiguous ranges of hosts, ex. kn[3,7-9,22-35,47]

  -p PROFILE : the name of a profile
     >> OR <<
  -d DISTRO : the name of a distro

If only required arguments are provided the reservation starts immediately with
the default length determined by the cluster admin team.

` + optionalFlags + `

Use the -s flag to set a start time for the reservation (other than now). Use
the format: ` + exStartDts() + `. (There is no seconds field.) It must be set
at least 5 minutes into the future and cannot start beyond the schedule window
as set by the cluster admin team. If this flag is not used the reservation
begins immediately.

Use the -e flag to set the end time/duration of a reservation. The expression 
can either be a datetime format or an interval specified in days(d), hours(h)
and minutes(m) in that order. A unit-less number is treated as minutes.
Examples:  ` + exEndDts() + ` | 3d | 5h32m | 12d2m | 90 (= 90m)
Days are defined as 24*60 minutes and do not take Daylight Savings offsets 
into account. The length is subject to the maximum allowable time that a
reservation can occupy in the schedule starting from 'now' and the scheduling
window limit as specified by your cluster admin team. If not specified the
default length is used. Default reservation time limits are viewable by
running the command: 'igor settings'

Use the -o flag to set a different owner for the reservation than the person
making it. This flag can only be used by admins and the action is called out in
the application log.

Use the -g flag to set a group that will have access to this reservation. Group
membership confers the ability to extend or delete the reservation and to issue
power commands to its assigned nodes. The reservation creator must be a member
of the provided group.

Use the -v flag to set a VLAN id number or name of an existing reservation. If
a number is provided, the new reservation will use the specified VLAN value if
not already taken. (The id range is available by running the 'igor settings' 
command.) If a name is provided, the VLAN of the new reservation is set to the
same VLAN as the named reservation. If this flag is not used on a VLAN-enabled
cluster then an id will be automatically assigned.

Use the --no-cycle flag to prevent the reservation's nodes from being power-
cycled when it becomes active. This will leave the nodes in whatever power
state they were in prior to the reservation start time (usually off).

Use the -k flag to set kernel arguments you would like to append to the
chosen distro to use with this reservation. Kernel args can only be used in
conjunction with distros. If you wish to change/append a kernel arg to a
profile, then you should update the profile first before using it in a new
reservation. 

` + descFlagText + `
`,
		Example: `

igor res create Shire -p ubu20 -n 7

  * Minimum for command to work.
  Requests a reservation named 'Shire' using profile 'ubu20' on seven nodes
  chosen by igor that begins now and lasts for the default amount of time set
  by the cluster admin team. Note the -d flag could be used instead of -p.


igor res create obiwan -d cent7 -n rx[5,12-15] -e ` + exEndDts() + ` -g jedis

  * Uses node range, timestamp for end, and group sharing.
  Requests a reservation named 'obiwan' using the distro 'cent7' on five nodes
  (rx5,rx12-15) that begins now, lasts until ` + exEndDay() + ` and is accessible
  to the 'jedis' group.


igor res create Twit2 -p twitserv -n dq74,dq9 -s ` + exStartDts() + ` -t 6d -v Twit1

  * Uses node list, future start date, duration to end, and vlan res-name.
  Requests a reservation named 'Twit2' using the profile 'twitserv' on three
  nodes starting ` + exStartDay() + ` for six days and shares the same vlan used
  by the reservation 'Twit1'.
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			nodes, _ := flagset.GetString("nodes")
			profile, _ := flagset.GetString("profile")
			distro, _ := flagset.GetString("distro")
			owner, _ := flagset.GetString("owner")
			group, _ := flagset.GetString("group")
			desc, _ := flagset.GetString("desc")
			start, _ := flagset.GetString("start")
			end, _ := flagset.GetString("end")
			vlan, _ := flagset.GetString("vlan")
			kernelArgs, _ := flagset.GetString("kernel-args")
			var noCycle *bool
			if flagset.Changed("no-cycle") {
				noCycleVal, _ := flagset.GetBool("no-cycle")
				noCycle = &noCycleVal
			}
			printRespSimple(doCreateReservation(args[0], distro, profile, owner, group, desc, start, end, vlan, nodes, kernelArgs, noCycle))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	var nodes,
		start,
		owner,
		desc,
		end,
		profile,
		group,
		vlan,
		kernelArgs,
		distro string
	var noCycle bool

	cmdCreateRes.Flags().StringVarP(&distro, "distro", "d", "", "distro to use")
	cmdCreateRes.Flags().StringVarP(&profile, "profile", "p", "", "profile to use")
	cmdCreateRes.Flags().StringVarP(&nodes, "nodes", "n", "", "node count or expression")
	cmdCreateRes.Flags().StringVarP(&start, "start", "s", "", "future start time")
	cmdCreateRes.Flags().StringVarP(&end, "end", "e", "", "end time (other than default)")
	cmdCreateRes.Flags().StringVarP(&owner, "owner", "o", "", "assign different owner "+adminOnly)
	cmdCreateRes.Flags().StringVarP(&group, "group", "g", "", "group allowed to access")
	cmdCreateRes.Flags().StringVarP(&vlan, "vlan", "v", "", "vlan number or existing res name")
	cmdCreateRes.Flags().StringVarP(&kernelArgs, "kernel-args", "k", "", "kernel args to append to a distro")
	cmdCreateRes.Flags().StringVar(&desc, "desc", "", "description of the reservation")
	cmdCreateRes.Flags().BoolVar(&noCycle, "no-cycle", false, "do not power cycle nodes at startup")

	_ = cmdCreateRes.MarkFlagRequired("nodes")

	// change here when new cobra lib supports exclusive flag groups
	_ = registerFlagArgsFunc(cmdCreateRes, "profile", []string{"PROFILE"})
	_ = registerFlagArgsFunc(cmdCreateRes, "distro", []string{"DISTRO"})

	_ = registerFlagArgsFunc(cmdCreateRes, "start", []string{"DATETIME"})
	_ = registerFlagArgsFunc(cmdCreateRes, "end", []string{"DATE/DUR"})
	_ = registerFlagArgsFunc(cmdCreateRes, "owner", []string{"USER"})
	_ = registerFlagArgsFunc(cmdCreateRes, "group", []string{"GROUP"})
	_ = registerFlagArgsFunc(cmdCreateRes, "vlan", []string{"ID/RES"})
	_ = registerFlagArgsFunc(cmdCreateRes, "kernel-args", []string{"\"KARGS\""})
	_ = registerFlagArgsFunc(cmdCreateRes, "desc", []string{"\"DESCRIPTION\""})

	return cmdCreateRes
}

func newResShowCmd() *cobra.Command {

	cmdShowRes := &cobra.Command{
		Use: "show [-n NAME1,...] [-o OWNER1,...] [-d DIST1,...] [-p PROF1,...]\n" +
			"       [-g GR1,...] [-x]",
		Short: "Show reservation information",
		Long: `
Shows reservation information, returning matches to specified parameters. If no
parameters are provided then all reservations will be returned.

` + optionalFlags + `

Use the -n, -o, -d, -p and -g flags to narrow results. Multiple values for a
given flag should be comma-delimited.

Use the -x flag to render screen output without pretty formatting.
`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			names, _ := flagset.GetStringSlice("names")
			owners, _ := flagset.GetStringSlice("owners")
			distros, _ := flagset.GetStringSlice("distros")
			profiles, _ := flagset.GetStringSlice("profiles")
			groups, _ := flagset.GetStringSlice("groups")
			simplePrint = flagset.Changed("simple")
			printReservations(doShowReservation(names, distros, profiles, owners, groups))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	var names,
		owners,
		groups,
		distros,
		profiles []string

	cmdShowRes.Flags().StringSliceVarP(&names, "names", "n", nil, "search by reservation name(s)")
	cmdShowRes.Flags().StringSliceVarP(&owners, "owners", "o", nil, "search by owner name(s)")
	cmdShowRes.Flags().StringSliceVarP(&groups, "groups", "g", nil, "search by group(s)")
	cmdShowRes.Flags().StringSliceVarP(&distros, "distros", "d", nil, "search by distro(s)")
	cmdShowRes.Flags().StringSliceVarP(&profiles, "profiles", "p", nil, "search by profile(s)")
	cmdShowRes.Flags().BoolVarP(&simplePrint, "simple", "x", false, "use simple text output")
	_ = registerFlagArgsFunc(cmdShowRes, "names", []string{"NAME1"})
	_ = registerFlagArgsFunc(cmdShowRes, "owners", []string{"OWNER1"})
	_ = registerFlagArgsFunc(cmdShowRes, "groups", []string{"GROUP1"})
	_ = registerFlagArgsFunc(cmdShowRes, "distros", []string{"DIST1"})
	_ = registerFlagArgsFunc(cmdShowRes, "profiles", []string{"PROF1"})

	return cmdShowRes
}

func newResEditCmd() *cobra.Command {

	cmdEditRes := &cobra.Command{
		Use: "edit NAME [ {--extend LENGTH | --extend-max} | \n" +
			"       --drop NODES | \n" +
			"       {-p PROFILE | -d DISTRO} | \n" +
			"       [-n NAME] [-o OWNER] [-g GROUP] [-k KARGS] [--desc \"DESCRIPTION\"]]",
		Short: "Edit a reservation",
		Long: `
Edits a reservation. With the exception of the extend flags (see below) changes
can only be made by the reservation owner or an admin.

` + requiredArgs + `

  NAME : reservation name

` + sBold("EXTENDING THE END TIME:") + `

A reservation can be extended with the --extend flag followed by a time value.
Time expressions can either be the datetime format ` + exStartDts() + ` that
specifies a new end time, or an interval specified in days(d), hours(h), and 
minutes(m), in that order. Unitless numbers are treated as minutes. Days are 
defined as 24*60 minutes and do not take Daylight Savings offsets into account.
Example: To extend a reservation for 7 more days: 7d. To extend for 4 days, 
6 hours, 30 minutes: 4d6h30m.

The new end time is subject to the maximum length of time a reservation can 
last starting from now (or from the start time if the reservation hasn't begun
yet) and how far into the future it can be scheduled. Contact your cluster
admin team for this information. The extend option can be performed by the
owner and any member of the reservation group if one is assigned.

Use the --extend-max flag to extend the reservation by the maximum amount of
time possible. Igor will determine the proper time interval needed.

It is not possible to extend future reservations if their length is already the
maximum length allowed.

These flags cannot be used with other edit parameters.

` + sBold("DROPPING HOSTS:") + `

Use the --drop flag to remove hosts from the reservation. The NODES arg is
the same used in 'igor res create'; a comma-delimited list (kn1,kn2,...) or a
multi-node range (kn[3,16-20,34]).

Drop allows reservation owners and admins to free up nodes without deleting the
reservation. This might be a necessity if a node has to be taken offline due to
failure. ` + sItalic("This is a permanent change. Once dropped a node cannot be added back.") + `
Assuming you can re-reserve the dropped node, one potential workaround would be
to provide the new reservation's vlan param with the name of the old reserva-
tion so the node can re-join its prior virtual network. See 'igor res create'
for more info.

This flag cannot be used to drop all nodes. Delete the reservation instead.

This flag cannot be used with other edit parameters.

` + sBold("CHANGING THE PROFILE OR DISTRO:") + `

Use the -p flag to change the profile used on the reserved nodes. An existing
profile name must be provided. Alternatively you may use the -d flag to use a
bare distro instead which uses that distro's default profile. Changing either
does not take effect until a power-cycle operation is performed on the reserva-
tion nodes. (See 'igor host power --help' for more information.)

These flags cannot be used with other edit parameters.

` + sBold("OTHER RESERVATION EDITS:") + `

Use the -n flag to change the reservation name.

Use the -o flag to transfer ownership to another user. After this change the
previous owner can no longer edit the reservation. The previous owner will
retain some access rights if they are a member of the reservation's assigned
group.

Use the -g flag to change/remove a group from the reservation. To remove the
group use the syntax '-g none'.

Use the -k flag to set kernel arguments you would like to append to the distro
being used with this reservation. Kernel args can only be used in conjunction
with the existing distro (temp profile). You cannot specify kernel args while
also changing the distro.

` + descFlagText + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			extend, _ := flagset.GetString("extend")
			extendMax := flagset.Changed("extend-max")
			distro, _ := flagset.GetString("distro")
			profile, _ := flagset.GetString("profile")
			newName, _ := flagset.GetString("name")
			drop, _ := flagset.GetString("drop")
			desc, _ := flagset.GetString("desc")
			owner, _ := flagset.GetString("owner")
			group, _ := flagset.GetString("group")
			kernelArgs, _ := flagset.GetString("kernel-args")
			printRespSimple(doEditReservation(args[0], extend, drop, distro, profile, newName, owner, group, desc, kernelArgs, extendMax))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	var name,
		owner,
		desc,
		profile,
		group,
		extend,
		drop,
		kernelArgs,
		distro string
	var extendMax bool

	cmdEditRes.Flags().StringVar(&extend, "extend", "", "extend reservation by provided time")
	cmdEditRes.Flags().BoolVar(&extendMax, "extend-max", false, "extend reservation by maximum time allowed")
	cmdEditRes.Flags().StringVar(&drop, "drop", "", "drop nodes from the reservation")
	cmdEditRes.Flags().StringVarP(&distro, "distro", "d", "", "update distro")
	cmdEditRes.Flags().StringVarP(&profile, "profile", "p", "", "update profile")
	cmdEditRes.Flags().StringVarP(&name, "name", "n", "", "update reservation name")
	cmdEditRes.Flags().StringVarP(&owner, "owner", "o", "", "update owner")
	cmdEditRes.Flags().StringVarP(&group, "group", "g", "", "update group")
	cmdEditRes.Flags().StringVarP(&kernelArgs, "kernel-args", "k", "", "add kernel args to a distro (temp profile)")
	cmdEditRes.Flags().StringVar(&desc, "desc", "", "update the description of the reservation")
	_ = registerFlagArgsFunc(cmdEditRes, "extend", []string{"DATE/DUR"})
	_ = registerFlagArgsFunc(cmdEditRes, "drop", []string{"NODES"})
	_ = registerFlagArgsFunc(cmdEditRes, "distro", []string{"DISTRO"})
	_ = registerFlagArgsFunc(cmdEditRes, "profile", []string{"PROFILE"})
	_ = registerFlagArgsFunc(cmdEditRes, "name", []string{"NAME"})
	_ = registerFlagArgsFunc(cmdEditRes, "owner", []string{"OWNER"})
	_ = registerFlagArgsFunc(cmdEditRes, "group", []string{"GROUP"})
	_ = registerFlagArgsFunc(cmdEditRes, "kernel-args", []string{"\"KARGS\""})
	_ = registerFlagArgsFunc(cmdEditRes, "desc", []string{"\"DESCRIPTION\""})

	return cmdEditRes
}

func newResDelCmd() *cobra.Command {

	cmdDeleteRes := &cobra.Command{
		Use:   "del NAME",
		Short: "Delete a reservation",
		Long: `
Deletes a reservation. This can only done by the reservation owner, group 
member or an admin.

` + requiredArgs + `

  NAME : reservation name
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printRespSimple(doDeleteReservation(args[0]))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	return cmdDeleteRes
}

func doCreateReservation(resName, distro, profile, owner, group, desc, stime, etime, vlan, nodes, kernelArgs string, noCycle *bool) *common.ResponseBodyBasic {

	params := map[string]interface{}{"name": resName}

	if nodeCount, err := strconv.Atoi(nodes); err != nil {
		params["nodeList"] = nodes
	} else {
		params["nodeCount"] = nodeCount
	}
	if profile != "" {
		params["profile"] = profile
	}
	if distro != "" {
		params["distro"] = distro
	}
	if group != "" {
		params["group"] = group
	}
	if owner != "" {
		params["owner"] = owner
	}
	if stime != "" {
		if _, err := common.ParseTimeFormat(stime); err != nil {
			checkClientErr(err)
		}
		startTime, _ := time.ParseInLocation(common.DateTimeCompactFormat, stime, cli.tzLoc)
		params["start"] = startTime.Unix()
	}
	if etime != "" {
		endTime, err := time.ParseInLocation(common.DateTimeCompactFormat, etime, cli.tzLoc)
		if err != nil {
			if _, pErr := common.ParseDuration(etime); pErr != nil {
				checkClientErr(fmt.Errorf("end time format invalid or not recognized: %v; and %v", err, pErr))
			}
			params["duration"] = etime
		} else {
			params["duration"] = endTime.Unix()
		}
	}
	if vlan != "" {
		params["vlan"] = vlan
	}
	if desc != "" {
		params["description"] = desc
	}
	if kernelArgs != "" {
		params["kernelArgs"] = kernelArgs
	}
	if noCycle != nil && *noCycle {
		params["noCycle"] = true
	}

	body := doSend(http.MethodPost, api.Reservations, params)
	return unmarshalBasicResponse(body)
}

func doShowReservation(names, distros, profiles, owners, groups []string) *common.ResponseBodyReservations {

	var params string

	if len(names) > 0 {
		for _, n := range names {
			params += "name=" + n + "&"
		}
	}
	if len(distros) > 0 {
		for _, o := range distros {
			params += "distro=" + o + "&"
		}
	}
	if len(profiles) > 0 {
		for _, p := range profiles {
			params += "profile=" + p + "&"
		}
	}
	if len(owners) > 0 {
		for _, o := range owners {
			params += "owner=" + o + "&"
		}
	}
	if len(groups) > 0 {
		for _, g := range groups {
			params += "group=" + g + "&"
		}
	}
	if params != "" {
		params = strings.TrimSuffix(params, "&")
		params = "?" + params
	}

	apiPath := api.Reservations + params
	body := doSend(http.MethodGet, apiPath, nil)
	rb := common.ResponseBodyReservations{}
	err := json.Unmarshal(*body, &rb)
	checkUnmarshalErr(err)
	return &rb
}

func doEditReservation(resName, extend, drop, distro, profile, newName, owner, group, desc, kernelArgs string, extendMax bool) *common.ResponseBodyBasic {
	apiPath := api.Reservations + "/" + resName
	params := map[string]interface{}{}

	if extend != "" {
		endTime, err := time.ParseInLocation(common.DateTimeCompactFormat, extend, cli.tzLoc)
		if err != nil {
			if _, pErr := common.ParseDuration(extend); pErr != nil {
				checkClientErr(fmt.Errorf("end time format invalid or not recognized: %v; and %v", err, pErr))
			}
			params["extend"] = extend
		} else {
			params["extend"] = endTime.Unix()
		}
	}
	if extendMax {
		params["extendMax"] = true
	}
	if drop != "" {
		params["drop"] = drop
	}
	if distro != "" {
		params["distro"] = distro
	}
	if profile != "" {
		params["profile"] = profile
	}
	if newName != "" {
		params["name"] = newName
	}
	if owner != "" {
		params["owner"] = owner
	}
	if group != "" {
		params["group"] = group
	}
	if desc != "" {
		params["description"] = desc
	}
	if kernelArgs != "" {
		params["kernelArgs"] = kernelArgs
	}

	body := doSend(http.MethodPatch, apiPath, params)
	return unmarshalBasicResponse(body)
}

func doDeleteReservation(resName string) *common.ResponseBodyBasic {
	apiPath := api.Reservations + "/" + resName
	body := doSend(http.MethodDelete, apiPath, nil)
	return unmarshalBasicResponse(body)
}

func printReservations(rb *common.ResponseBodyReservations) {

	checkAndSetColorLevel(rb)

	resList := rb.Data["reservations"]
	if len(resList) == 0 {
		printSimple("no reservations to show (yet) or no matches based on search criteria", cRespWarn)
	}

	sort.Slice(resList, func(i, j int) bool {
		return resList[i].End < resList[j].End
	})

	oneYearLater := igorCliNow.Add(time.Hour * 24 * 365).Unix()

	timeFmt := "Jan 2 3:04 PM"

	if simplePrint {

		var resInfo string
		for _, r := range resList {

			if r.End > oneYearLater {
				timeFmt = "Jan-02-06.15:04-07:00"
			}

			resInfo = "RESERVATION: " + r.Name + "\n"
			resInfo += "  -DESCRIPTION:  " + r.Description + "\n"
			resInfo += "  -OWNER:        " + r.Owner + "\n"
			resInfo += "  -GROUP:        " + r.Group + "\n"
			resInfo += "  -PROFILE:      " + r.Profile + "\n"
			resInfo += "  -DISTRO:       " + r.Distro + "\n"
			resInfo += "  -HOSTS:        " + r.HostRange + "\n"
			resInfo += "  -VLAN:         " + strconv.Itoa(r.Vlan) + "\n"
			resInfo += "  -START:        " + getLocTime(time.Unix(r.Start, 0)).Format(timeFmt) + "\n"
			resInfo += "  -END:          " + getLocTime(time.Unix(r.End, 0)).Format(timeFmt) + "\n"
			resInfo += "  -ORIG-END:     " + getLocTime(time.Unix(r.OrigEnd, 0)).Format(timeFmt) + "\n"
			resInfo += "  -EXTEND-COUNT: " + strconv.Itoa(r.ExtendCount) + "\n"
			resInfo += "  -INSTALLED:    " + strconv.FormatBool(r.Installed) + "\n"
			if len(r.InstallError) > 0 {
				resInfo += "  -INSTALL-ERR:  " + r.InstallError + "\n"
			}
			fmt.Print(resInfo + "\n\n")
		}

	} else {

		tw := table.NewWriter()
		tw.AppendHeader(table.Row{"NAME", "DESCRIPTION", "OWNER", "GROUP", "PROFILE", "DISTRO", "HOSTS", "DOWN/NA", "VLAN", "START", "END", "ORIG-END", "EXTEND-COUNT", "INSTALLED", "INSTALL-ERR"})
		tw.AppendSeparator()

		// for the table version, only put zone on first column
		startTimeFmt := "Jan 2 3:04 PM MST"

		for _, r := range resList {

			if r.End > oneYearLater {
				startTimeFmt = "Jan 2 2006 3:04 PM MST"
				timeFmt = "Jan 2 2006 3:04 PM"
			}

			downNA := ""
			if len(r.HostsPowerNA) > 0 {
				downNA += cWarning.Sprint(r.HostsPowerNA) + "/"
			}
			if len(r.HostsDown) > 0 {
				downNA += cAlert.Sprint(r.HostsDown)
			}
			downNA = strings.TrimSuffix(downNA, "/")

			tw.AppendRow([]interface{}{
				r.Name,
				r.Description,
				r.Owner,
				r.Group,
				r.Profile,
				r.Distro,
				r.HostRange,
				downNA,
				r.Vlan,
				getLocTime(time.Unix(r.Start, 0)).Format(startTimeFmt),
				getLocTime(time.Unix(r.End, 0)).Format(timeFmt),
				getLocTime(time.Unix(r.OrigEnd, 0)).Format(timeFmt),
				r.ExtendCount,
				r.Installed,
				r.InstallError,
			})
		}

		tw.SetColumnConfigs([]table.ColumnConfig{
			{
				Name:     "DESCRIPTION",
				WidthMax: 40,
			},
		})

		tw.SetStyle(igorTableStyle)
		fmt.Printf("\n" + tw.Render() + "\n\n")
	}

}
