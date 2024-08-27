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

	"igor2/internal/pkg/common"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func newHostCmd() *cobra.Command {
	cmdHost := &cobra.Command{
		Use:   "host",
		Short: "Perform a host command",
		Long: `
Host primary command. A sub-command must be invoked to do anything.

Hosts are the nodes being reserved. Though hosts are collectively created using
the cluster command, individual hosts may be searched, modified and deleted as 
needed. This includes assigning a policy to a host.
`,
	}

	cmdHost.AddCommand(newHostShowCmd())
	cmdHost.AddCommand(newHostEditCmd())
	cmdHost.AddCommand(newHostDelCmd())
	cmdHost.AddCommand(newHostBlockCmd())
	cmdHost.AddCommand(newHostUnblockCmd())
	return cmdHost
}

func newHostShowCmd() *cobra.Command {

	cmdShowHosts := &cobra.Command{
		Use: "show [-n NODES] [-d HOSTNAME1,...] [-e ETH1,...] [-i IP1,...]\n" +
			"       [-p POL1,...] [-m MACID1,...] [-s STATE1,...] [-r RES1,...]\n" +
			"       [--powered {true|false}] [-x]",
		Short: "Show host information",
		Long: `
Shows host information, returning matches to specified parameters. If no 
optional parameters are provided then all hosts will be returned.

Output will provide the host name, network info and assigned reservations.

In the formatted table output, powered states are designated with symbols:

 ` + pUp.Sprintf("●") + ` : node is up
 ` + pDown.Sprintf("●") + ` : node is down
 ` + pUnknown.Sprintf("●") + ` : node power status unavailable

The HOSTNAME column will repeat the NODE column unless a different hostname
has been specified independently of the node name identifier. If different the
value in HOSTNAME is informational only. It cannot replace a node name in
igor commands.

The POLICY column will display the name of the policy assigned to the node.
The "default" policy means the node is available 24/7 for any igor user to
reserve. Other policies will have additional information in the ACCESS-GROUPS
or NO-AVAIL columns.

Information in the ACCESS-GROUPS column indicates that specific group(s) have
exclusive access to the node. Reservations made with these nodes must be
assigned to one of the groups listed.

Information in the NO-AVAIL column indicates that the node is unavailable for
reservations during the indicated period of time.

` + optionalFlags + `

Use the -d, -e, -i, -m, -p, -r and -s flags to filter results.

Use -n NODES to filter by name list or range of hosts:
    * name list is comma-delimited: kn1,kn2,kn3,...
    * range is the form prefix[n,m-n,...] where m,n are integers representing
      a single or contiguous ranges of hosts, ex. kn[3,7-9,22-35,47]

Use the --powered flag to only display powered nodes. Set it to false to only 
display unpowered nodes.

When searching by state (-s) acceptable parameters are ` + sBold("available") + `, ` + sBold("reserved") + `,
` + sBold("blocked") + ` and ` + sBold("error") + `.

Use the -x flag to render screen output without pretty formatting.
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {

			flagset := cmd.Flags()
			names, _ := flagset.GetString("nodes")
			hostnames, _ := flagset.GetStringSlice("hostnames")
			eths, _ := flagset.GetStringSlice("eths")
			ips, _ := flagset.GetStringSlice("IPs")
			macs, _ := flagset.GetStringSlice("macs")
			policies, _ := flagset.GetStringSlice("policies")
			reservations, _ := flagset.GetStringSlice("reservations")
			states, _ := flagset.GetStringSlice("states")
			simplePrint = flagset.Changed("simple")
			var powered *bool
			if flagset.Changed("powered") {
				poweredVal, _ := flagset.GetBool("powered")
				powered = &poweredVal
			}
			printHosts(doShowHosts(names, hostnames, eths, ips, macs, policies, reservations, states, powered))
			return nil
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	var macs,

		hostnames,
		ips,
		eths,
		hostPolicies,
		reservations,
		states []string
	var names string
	var powerVal bool

	cmdShowHosts.Flags().StringVarP(&names, "nodes", "n", "", "node list or range")
	cmdShowHosts.Flags().StringSliceVarP(&hostnames, "hostnames", "d", nil, "comma-delimited hostname list")
	cmdShowHosts.Flags().StringSliceVarP(&ips, "IPs", "i", nil, "comma-delimited ipv4 list")
	cmdShowHosts.Flags().StringSliceVarP(&macs, "macs", "m", nil, "comma-delimited MAC address list")
	cmdShowHosts.Flags().StringSliceVarP(&eths, "eths", "e", nil, "comma-delimited eth info list")
	cmdShowHosts.Flags().StringSliceVarP(&hostPolicies, "policies", "p", nil, "comma-delimited policy list")
	cmdShowHosts.Flags().StringSliceVarP(&reservations, "reservations", "r", nil, "comma-delimited reservation list")
	cmdShowHosts.Flags().StringSliceVarP(&states, "states", "s", nil, "comma-delimited state list")
	cmdShowHosts.Flags().BoolVar(&powerVal, "powered", true, "filter on powered or unpowered nodes")
	cmdShowHosts.Flags().BoolVarP(&simplePrint, "simple", "x", false, "use simple text output")

	_ = registerFlagArgsFunc(cmdShowHosts, "states", []string{"available", "reserved", "blocked", "error"})
	_ = registerFlagArgsFunc(cmdShowHosts, "names", []string{"NODES"})
	_ = registerFlagArgsFunc(cmdShowHosts, "hostnames", []string{"HOSTNAME1"})
	_ = registerFlagArgsFunc(cmdShowHosts, "IPs", []string{"IP1"})
	_ = registerFlagArgsFunc(cmdShowHosts, "eths", []string{"ETH1"})
	_ = registerFlagArgsFunc(cmdShowHosts, "policies", []string{"POL1"})
	_ = registerFlagArgsFunc(cmdShowHosts, "reservations", []string{"RES1"})
	_ = registerFlagArgsFunc(cmdShowHosts, "names", []string{"NAME1"})

	return cmdShowHosts
}

func newHostEditCmd() *cobra.Command {

	cmdEditHost := &cobra.Command{
		Use:   "edit NAME {[-p POLICY] [-d HOSTNAME] [-b BOOT] [-e ETH] [-i IP] [-m MACID]}",
		Short: "Edit host information " + adminOnly,
		Long: `
Edits host information.

Editing a host forces an update to the 'igor-clusters.yaml' file with the 
previous version backed up under a modified name.

` + requiredArgs + `
  NAME : host name

` + optionalFlags + `

Use the -p flag to assign a policy to a host. Mass policy/host assignment is
possible using the 'igor policy apply' command.

Use the -d flag to set a hostname or host alias that is different from the
host's name. Igor assumes the host's hostname follows the convention:

  <prefix><seq#>

where prefix is the value given in igor-cluster.yaml and seq# is the number in
the sequence of hosts given in the same config file. Igor uses this hostname to
communicate with the host itself. If the actual hostname is different, specify
it here and igor will use this hostname instead.

Use the -b flag to change the boot type of the host (bios or uefi).

Use the -i flag to change the host's IP.

Use the -e flag to change the host's ethernet switch identifier.

Use the -m flag to change the MAC address.

` + adminOnlyBanner + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			boot, _ := flagset.GetString("boot")
			hostname, _ := flagset.GetString("hostname")
			hostPolicy, _ := flagset.GetString("policy")
			ip, _ := flagset.GetString("ip")
			eth, _ := flagset.GetString("eth")
			mac, _ := flagset.GetString("mac")
			printRespSimple(doEditHost(args[0], boot, hostname, hostPolicy, ip, eth, mac))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	var ip,
		boot,
		eth,
		hostname,
		hostPolicy,
		mac string

	cmdEditHost.Flags().StringVarP(&hostPolicy, "policy", "p", "", "name of policy to assign to this host")
	cmdEditHost.Flags().StringVarP(&hostname, "hostname", "d", "", "hostname of the host")
	cmdEditHost.Flags().StringVarP(&boot, "boot", "b", "", "boot type of the host (bios or uefi)")
	cmdEditHost.Flags().StringVarP(&ip, "ip", "i", "", "ipv4 address")
	cmdEditHost.Flags().StringVarP(&mac, "mac", "m", "", "MAC address")
	cmdEditHost.Flags().StringVarP(&eth, "eth", "e", "", "eth config string")
	_ = registerFlagArgsFunc(cmdEditHost, "policy", []string{"POLICY"})
	_ = registerFlagArgsFunc(cmdEditHost, "hostname", []string{"HOSTNAME"})
	_ = registerFlagArgsFunc(cmdEditHost, "ip", []string{"IP"})
	_ = registerFlagArgsFunc(cmdEditHost, "mac", []string{"MACID"})
	_ = registerFlagArgsFunc(cmdEditHost, "eth", []string{"ETH"})

	return cmdEditHost
}

func newHostDelCmd() *cobra.Command {

	cmdDeleteHost := &cobra.Command{
		Use:   "del NAME",
		Short: "Delete a host " + adminOnly,
		Long: `
Deletes an igor host.

` + requiredArgs + `

  NAME : host name

` + notesOnUsage + `

Deleting a host forces an update to the 'igor-clusters.yaml' file with the 
previous version backed up under a modified name. Deleting a host completely
obscures the physical machine from igor's reservation and command/control
system.

A host cannot be deleted if it has associated reservations. They must expire,
be deleted, or edited to drop the node first.

` + adminOnlyBanner + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printRespSimple(doDeleteHost(args[0]))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	return cmdDeleteHost
}

func newHostPowerCmd() *cobra.Command {

	cmdPowerHosts := &cobra.Command{
		Use:   "power {on|off|cycle} {-r RES | -n NODES}",
		Short: "Send a power command to one or more hosts",
		Long: `
Executes the given power command on a set of hosts specified either explicitly
or through a reservation name.

Power commands can be executed by any admin or any user that owns or belongs
to a group that has an active reservation on the specified hosts. Power 
commands will not be honored if the network status of the node is reported to
be in an error state.

` + requiredArgs + ` (choose one)

       on : turns power on
      off : turns power off
    cycle : turns power off (if on), then power on
  
` + requiredFlags + `

  -r RES : a single reservation name. All hosts assigned to the reservation
     will be subject to the power command.
  >> OR <<
  -n NODES : a name list or range of hosts
    * name list is comma-delimited: kn1,kn2,kn3,...
    * range is the form prefix[n,m-n,...] where m,n are integers representing
      a single or contiguous ranges of hosts, ex. kn[3,7-9,22-35,47]

` + notesOnUsage + `

Power commands are routed through Igor to an external IPMI service that tells
igor immediately if the command was successfully submitted. The actual booting
of a host can fail for many other reasons. Attempts to power command a node
should therefore be followed up with close monitoring to check that the boot
completed, sometimes taking as long as a few minutes before the power status
changes.
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			nodes, _ := flagset.GetString("nodes")
			reservation, _ := flagset.GetString("res")
			printRespSimple(doPowerHosts(args[0], nodes, reservation))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return []string{"on", "off", "cycle"}, cobra.ShellCompDirectiveNoFileComp
		},
	}

	var hosts,
		res string

	cmdPowerHosts.Flags().StringVarP(&hosts, "nodes", "n", "", "node list or range")
	cmdPowerHosts.Flags().StringVarP(&res, "res", "r", "", "reservation name")
	_ = registerFlagArgsFunc(cmdPowerHosts, "nodes", []string{"NODES"})
	_ = registerFlagArgsFunc(cmdPowerHosts, "res", []string{"RES"})

	return cmdPowerHosts
}

func newHostBlockCmd() *cobra.Command {

	cmdBlockHosts := &cobra.Command{
		Use:   "block NODES",
		Short: "Block hosts from being reserved " + adminOnly,
		Long: `
Blocks hosts from being reserved. In this state hosts are unavailable for
reservation operations whether or not they are powered on.

` + requiredArgs + `

  NODES  - a name list or range of hosts
    * name list is comma-delimited: kn1,kn2,kn3,...
    * range is the form prefix[n,m-n,...] where m,n are integers representing
      a single or contiguous ranges of hosts, ex. kn[3,7-9,22-35,47]

` + notesOnUsage + `

This command can be used by admins as a means of quickly taking one or more
hosts out of the reservation pool to prevent their use. Blocking a host is not
the same as applying a policy with scheduled unavailability. A block represents
an unavailable node state with no defined end time. It can only be returned to
a reservable state using the 'igor host unblock' command.

A host cannot be blocked if it has any current or future reservation; the
reservation must expire, be deleted, or edited to drop the node first.

Blocked hosts will still be displayed in 'igor show' but with an indicator of
their blocked status.

` + adminOnlyBanner + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printRespSimple(doBlockHost(true, args[0]))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return []string{"NODES"}, cobra.ShellCompDirectiveNoFileComp
		},
	}

	return cmdBlockHosts

}

func newHostUnblockCmd() *cobra.Command {

	cmdUnblockHosts := &cobra.Command{
		Use:   "unblock NODES",
		Short: "Return hosts to reservable status " + adminOnly,
		Long: `
Removes a blocked status on one or more nodes. See the help section of the
'igor host block' command for info on blocking nodes.

Once executed the specified hosts will be able to accept reservations.

` + requiredArgs + `

  NODES  - a name list or range of hosts
    * name list is comma-delimited: kn1,kn2,kn3,...
    * range is the form prefix[n,m-n,...] where m,n are integers representing
      a single or contiguous ranges of hosts, ex. kn[3,7-9,22-35,47]

` + adminOnlyBanner + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printRespSimple(doBlockHost(false, args[0]))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return []string{"NODES"}, cobra.ShellCompDirectiveNoFileComp
		},
	}

	return cmdUnblockHosts
}

func doShowHosts(names string, hostnames []string, eths []string, ips []string, macs []string, hostPolicies []string, reservations []string, states []string, powered *bool) *common.ResponseBodyHosts {

	var params string
	if len(names) > 0 {
		params += "name=" + names + "&"
	}
	if len(hostnames) > 0 {
		for _, n := range hostnames {
			params += "hostname=" + n + "&"
		}
	}
	if len(eths) > 0 {
		for _, o := range eths {
			params += "eth=" + o + "&"
		}
	}
	if len(ips) > 0 {
		for _, o := range ips {
			params += "ip=" + o + "&"
		}
	}
	if len(macs) > 0 {
		for _, o := range macs {
			params += "mac=" + o + "&"
		}
	}
	if len(hostPolicies) > 0 {
		for _, o := range hostPolicies {
			params += "hostPolicy=" + o + "&"
		}
	}
	if len(reservations) > 0 {
		for _, o := range reservations {
			params += "reservation=" + o + "&"
		}
	}
	if len(states) > 0 {
		for _, o := range states {
			params += "state=" + o + "&"
		}
	}
	if powered != nil {
		params += "powered=" + strconv.FormatBool(*powered) + "&"
	}
	if params != "" {
		params = strings.TrimSuffix(params, "&")
		params = "?" + params
	}
	apiPath := api.Hosts + params
	body := doSend(http.MethodGet, apiPath, nil)
	rb := common.ResponseBodyHosts{}
	err := json.Unmarshal(*body, &rb)
	checkUnmarshalErr(err)
	return &rb
}

func doEditHost(name, boot, hostname, hostPolicy, ip, eth, mac string) *common.ResponseBodyBasic {
	apiPath := api.Hosts + "/" + name
	params := make(map[string]interface{})
	if hostname != "" {
		params["hostname"] = hostname
	}
	if boot != "" {
		params["boot"] = boot
	}
	if hostPolicy != "" {
		params["hostPolicy"] = hostPolicy
	}
	if ip != "" {
		params["ip"] = ip
	}
	if eth != "" {
		params["eth"] = eth
	}
	if mac != "" {
		params["mac"] = mac
	}
	body := doSend(http.MethodPatch, apiPath, params)
	return unmarshalBasicResponse(body)
}

func doDeleteHost(name string) *common.ResponseBodyBasic {
	apiPath := api.Hosts + "/" + name
	body := doSend(http.MethodDelete, apiPath, nil)
	return unmarshalBasicResponse(body)
}

func doPowerHosts(command string, nodes string, reservation string) *common.ResponseBodyBasic {
	params := make(map[string]interface{})
	params["cmd"] = command
	// let the server reject if both are blank/set
	if nodes != "" {
		params["hosts"] = nodes
	}
	if reservation != "" {
		params["resName"] = reservation
	}

	body := doSend(http.MethodPatch, api.HostsPower, params)
	return unmarshalBasicResponse(body)
}

func doBlockHost(block bool, hosts string) *common.ResponseBodyBasic {
	params := make(map[string]interface{})
	params["block"] = block
	params["hosts"] = hosts
	body := doSend(http.MethodPatch, api.HostsBlock, params)
	return unmarshalBasicResponse(body)
}

func printHosts(rb *common.ResponseBodyHosts) {

	checkAndSetColorLevel(rb)

	hosts := rb.Data["hosts"]
	if len(hosts) == 0 {
		printSimple("no hosts to show (yet) or no matches based on search criteria", cRespWarn)
	}

	sort.Slice(hosts, func(i, j int) bool {
		return hosts[i].SequenceID < hosts[j].SequenceID
	})

	stateColor := func(state string) string {
		switch state {
		case "available":
			return hsAvailable.Sprint(state)
		case "reserved":
			return hsReserved.Sprint(state)
		case "blocked":
			return cBlockedUp.Sprint(state)
		default:
			return cInstError.Sprintf(state)
		}
	}

	powerColor := func(powered string) string {
		if simplePrint {
			return powered
		}
		if powered == "true" {
			return pUp.Sprintf("●")
		} else if powered == "false" {
			return pDown.Sprintf("●")
		} else {
			return pUnknown.Sprintf("●")
		}
	}

	tw := table.NewWriter()
	tw.AppendHeader(table.Row{"NODE", "STATE", "POWER", "BOOT-TYPE", "MACID", "HOSTNAME", "IP", "ETH", "POLICY", "ACCESS-GROUPS", "RESTRICTED", "RESERVATIONS"})

	for _, h := range hosts {
		tw.AppendRow([]interface{}{
			sBold(h.Name),
			stateColor(h.State),
			powerColor(h.Powered),
			h.BootMode,
			h.Mac,
			h.HostName,
			h.IP,
			h.Eth,
			h.HostPolicy,
			strings.Join(h.AccessGroups, "\n"),
			h.Restricted,
			strings.Join(h.Reservations, "\n"),
		})
	}

	if simplePrint {
		tw.Style().Options.SeparateRows = false
		tw.Style().Options.SeparateColumns = true
		tw.Style().Options.DrawBorder = false
	} else {
		tw.SetStyle(igorTableStyle)
	}

	fmt.Printf("\n" + tw.Render() + "\n\n")

}
