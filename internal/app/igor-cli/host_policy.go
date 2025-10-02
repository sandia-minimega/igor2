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
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"igor2/internal/pkg/common"

	"github.com/spf13/cobra"
)

func newHostPolicyCmd() *cobra.Command {

	cmdHostPolicy := &cobra.Command{
		Use:   "policy",
		Short: "Perform a policy command",
		Long: `
Policy primary command. A sub-command must be invoked to do anything.

Policies define access restrictions to the host they are assigned to. These
restrictions can include specific groups that can access the host, the max time
the host can be reserved, and scheduling times when the host is unavailable to
be reserved.

Policies are a powerful tool for tailoring parts of a cluster (or all of it)
with different rules about reservation scheduling.

For example, if you wanted to specify 100 nodes on your 1000-node cluster to
only offer short-term reservation periods (max time limit of 7 days), whereas
the rest of the cluster allows for max time of 2 months, a policy would be
the way to do this.

Or perhaps there is a research group that needs 20 nodes on the cluster for
an indefinite period of time. You can make a policy that gives their users
sole reservation rights to those nodes.

You can also make policies that turn off reservation scheduling on hosts for
certain periods of time. Unlike the 'igor host block' command that offers an
immediate way to remove hosts indefinitely from the reservation pool, a policy
that defines an unavailable period is more flexible in disallowing reservations
as part of a publishable schedule -- holidays, scheduled maintenance, and other
dates that admins usually know about well in advance.

Furthermore, policies don't interrupt current reservations on hosts when they
are applied. No one will lose their spot on a node as the result of a policy 
change, but admins might have to ask users to prematurely end reservations
that conflict with the policy if the need is time-sensitive.

A policy may be assigned to multiple hosts, but a host can only be associated 
with one policy. Every host is assigned to the 'default' policy at startup
which only applies the default max duration of a reservation to all nodes.

` + sBold("All policy commands except 'show' are admin-only.") + `
`,
	}

	cmdHostPolicy.AddCommand(newHostPolicyCreateCmd())
	cmdHostPolicy.AddCommand(newHostPolicyShowCmd())
	cmdHostPolicy.AddCommand(newHostPolicyEditCmd())
	cmdHostPolicy.AddCommand(newHostPolicyApplyCmd())
	cmdHostPolicy.AddCommand(newHostPolicyDelCmd())
	return cmdHostPolicy
}

func newHostPolicyCreateCmd() *cobra.Command {

	cmdCreateHostPolicy := &cobra.Command{
		Use:   "create NAME {[-t MAXTIME -g GRP1,... -u \"EXP1\",...]}",
		Short: "Create a policy " + adminOnly,
		Long: `
Creates a new igor policy. A policy is a defined set of restrictions that can
be assigned to hosts. All three varieties of policy restrictions may be
combined into a single policy.

See 'igor policy -h' for a description of what a policy is and use cases.

` + requiredArgs + `

  NAME : policy name

` + sBold("RESTRICT BY MAX RESERVATION LENGTH:") + `

Use the -t flag to set a time interval that limits how long a host can be
reserved. Possible units are days(d), hours(h) and minutes(m) in that order. A
unit-less number is treated as minutes. Days are defined as 24*60 minutes and
do not take Daylight Savings offsets into account. 
Ex. 3d | 5h32m | 12d2m | 90 (= 90m)

` + sBold("RESTRICT BY GROUP MEMBERSHIP:") + `

Use the -g flag to set one or more groups that are allowed to reserve the hosts
this policy is associated with. A reservation must specify a group that matches
to this list in order to use this policy's hosts. Policies that don't use this
flag allow any user to reserve a host.

` + sBold("RESTRICT BY SCHEDULE:") + `

Use the -u flag to set one or more periods during which the policy will not
allow reservations to be made on its hosts. The format is "cron:duration" as 
explained below.

The first part of the expression defines when the host will start to be un-
available to reserve. This value is given in the form of cron syntax. A basic
cron expression consists of 5 character fields, each separated by a single
space. Each field represents a unit of time (min,hr,day,mo,DoW) and allows
values and characters specific to its respective unit.

For more information on cron expressions: https://en.wikipedia.org/wiki/Cron

Example cron expression:
"0 0 * * 6:3d2h" -> every Saturday at midnight, lasting 3 days and 2 hours

The second part of the expression defines the duration, or how long the host
should remain unavailable after the cron-defined start time. The duration is
given in the same form as the -t flag described above.

Together, a complete expression would look like "0 0 * * 6:3d2h"

` + adminOnlyBanner + `
`,
		Example: `
igor policy create WinterBreak -u "0 17 24 12 *:8d15h"

This defines a period that starts every year on Dec 24 at 5 PM and lasts
until Jan 2 at 8 AM.
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flagset := cmd.Flags()
			maxResTime, _ := flagset.GetString("max-time")
			groups, _ := flagset.GetStringSlice("groups")
			unavailable, _ := flagset.GetStringSlice("unavail")
			if res, err := doCreateHostPolicy(args[0], maxResTime, groups, unavailable); err != nil {
				return err
			} else {
				printRespSimple(res)
				return nil
			}
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	var maxTime string
	var groups, unavailable []string

	cmdCreateHostPolicy.Flags().StringVarP(&maxTime, "max-time", "t", "", "max time limit for reserving hosts assigned to this policy")
	cmdCreateHostPolicy.Flags().StringSliceVarP(&groups, "groups", "g", nil, "comma-delimited list of groups to grant access")
	cmdCreateHostPolicy.Flags().StringSliceVarP(&unavailable, "unavail", "u", nil, "comma-delimited list of schedule block entries")
	_ = registerFlagArgsFunc(cmdCreateHostPolicy, "max-time", []string{"MAXTIME"})
	_ = registerFlagArgsFunc(cmdCreateHostPolicy, "groups", []string{"GRP1"})
	_ = registerFlagArgsFunc(cmdCreateHostPolicy, "unavail", []string{"\"EXP1\""})

	return cmdCreateHostPolicy
}

func newHostPolicyShowCmd() *cobra.Command {

	cmdShowHostPolicy := &cobra.Command{
		Use:   "show [-n NAME1,...] [-g GRP1,...] [--hosts HOST1,...] [-x]",
		Short: "Show policy information",
		Long: `
Shows policy information, returning matches to specified parameters. If no
optional filtering parameters are provided then all policies will be returned.

` + optionalFlags + `

Use the -n, -g and --hosts flags to filter the returned list by policy names,
groups, and hosts respectively.

Use the -x flag to render screen output without pretty formatting.
`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			names, _ := flagset.GetStringSlice("names")
			groups, _ := flagset.GetStringSlice("groups")
			hosts, _ := flagset.GetStringSlice("hosts")
			simplePrint = flagset.Changed("simple")
			printPolicies(doShowHostPolicy(names, groups, hosts))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	var groups,
		names,
		hosts []string

	cmdShowHostPolicy.Flags().StringSliceVarP(&names, "names", "n", nil, "comma-delimited list of policy names")
	cmdShowHostPolicy.Flags().StringSliceVarP(&groups, "groups", "g", nil, "comma-delimited list of group names")
	cmdShowHostPolicy.Flags().StringSliceVar(&hosts, "hosts", nil, "comma-delimited list of host names")
	cmdShowHostPolicy.Flags().BoolVarP(&simplePrint, "simple", "x", false, "use simple text output")
	_ = registerFlagArgsFunc(cmdShowHostPolicy, "names", []string{"NAME1"})
	_ = registerFlagArgsFunc(cmdShowHostPolicy, "groups", []string{"GRP1"})
	_ = registerFlagArgsFunc(cmdShowHostPolicy, "hosts", []string{"HOST1"})

	return cmdShowHostPolicy
}

func newHostPolicyEditCmd() *cobra.Command {

	cmdEditHostPolicy := &cobra.Command{
		Use: "edit NAME { [-n NEWNAME] [-t MAXTIME] [-g GRP1,...] [-r GRP1,...]\n" +
			"            [-u \"EXP1\",...] [-x \"EXP1\",...] }",
		Short: "Edit a policy " + adminOnly,
		Long: `
Edits policy information.

Care should be taken to inform users of policy changes and how they affect
availability of cluster resources.

` + requiredArgs + `

  NAME : policy name

` + optionalFlags + `

Use the -n flag to re-name a policy.

Use the -t flag to reset the time interval that limits how long a host can be
reserved. Possible units are days(d), hours(h) and minutes(m) in that order. A
unit-less number is treated as minutes. Days are defined as 24*60 minutes and
do not take Daylight Savings offsets into account. 
Ex. 3d | 5h32m | 12d2m | 90 (= 90m)

Use the -g flag to add groups and the -r flag to remove groups from the policy.
If the last group is removed from the policy, then all users will be able to
reserve its hosts.

Use the -u flag to add unavailability periods and the -x flag to remove them
from the policy.

` + adminOnlyBanner + `
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flagset := cmd.Flags()
			name, _ := flagset.GetString("name")
			maxResTime, _ := flagset.GetString("max-time")
			groupAdd, _ := flagset.GetStringSlice("add-groups")
			groupRemove, _ := flagset.GetStringSlice("remove-groups")
			unavailableAdd, _ := flagset.GetStringSlice("add-unavail")
			unavailableRemove, _ := flagset.GetStringSlice("remove-unavail")
			if res, err := doEditHostPolicy(args[0], name, maxResTime, groupAdd, groupRemove, unavailableAdd, unavailableRemove); err != nil {
				return err
			} else {
				printRespSimple(res)
				return nil
			}
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	var name,
		duration string
	var groupA,
		groupR,
		unavailableA,
		unavailableR []string

	cmdEditHostPolicy.Flags().StringVarP(&name, "name", "n", "", "new name to assign to this policy")
	cmdEditHostPolicy.Flags().StringVarP(&duration, "max-time", "t", "", "max time limit for reservations under this policy")
	cmdEditHostPolicy.Flags().StringSliceVarP(&groupA, "add-groups", "g", nil, "comma-delimited list of groups to grant access")
	cmdEditHostPolicy.Flags().StringSliceVarP(&groupR, "remove-groups", "r", nil, "comma-delimited list of groups to remove access")
	cmdEditHostPolicy.Flags().StringSliceVarP(&unavailableA, "add-unavail", "u", nil, "comma-delimited list of schedule block entries to add")
	cmdEditHostPolicy.Flags().StringSliceVarP(&unavailableR, "remove-unavail", "x", nil, "comma-delimited list of schedule block entries to remove")
	_ = registerFlagArgsFunc(cmdEditHostPolicy, "name", []string{"NAME"})
	_ = registerFlagArgsFunc(cmdEditHostPolicy, "max-time", []string{"MAXTIME"})
	_ = registerFlagArgsFunc(cmdEditHostPolicy, "add-groups", []string{"GRP1"})
	_ = registerFlagArgsFunc(cmdEditHostPolicy, "remove-groups", []string{"GRP1"})
	_ = registerFlagArgsFunc(cmdEditHostPolicy, "add-unavail", []string{"EXP1"})
	_ = registerFlagArgsFunc(cmdEditHostPolicy, "remove-unavail", []string{"EXP1"})

	return cmdEditHostPolicy
}

func newHostPolicyApplyCmd() *cobra.Command {

	cmdApplyHostPolicy := &cobra.Command{
		Use:   "apply NAME NODES",
		Short: "Apply a policy to nodes " + adminOnly,
		Long: `
Applies an igor policy to a set of nodes, replacing any policy(s) that is
currently in place for those nodes.

` + requiredArgs + `

  NAME : policy name
  NODES  - a name list or range of hosts
    * name list is comma-delimited: kn1,kn2,kn3,...
    * range is the form prefix[n,m-n,...] where m,n are integers representing
      a single or contiguous ranges of hosts, ex. kn[3,7-9,22-35,47]

` + notesOnUsage + `

Care should be taken to inform users of new policies and how they affect
availability of cluster resources.

To revert hosts back to the default policy, use "default" for the NAME param.

A reservation on a node that has a new policy applied will be honored until
the reservation expires or is deleted. Such a reservation can only be extended
if the reservation's owner, group and time parameters are compliant with the
new policy's restrictions.

` + adminOnlyBanner + `
`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			printRespSimple(doApplyHostPolicy(args[0], args[1]))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return []string{"NAME", "NODES"}, cobra.ShellCompDirectiveNoFileComp
		},
	}

	return cmdApplyHostPolicy
}

func newHostPolicyDelCmd() *cobra.Command {

	cmdDeleteHostPolicy := &cobra.Command{
		Use:   "del NAME",
		Short: "Delete a policy " + adminOnly,
		Long: `
Deletes an igor policy. A policy cannot be deleted if it is applied to a host.
Change the policy on the affected host first.

` + requiredArgs + `

  NAME : policy name

` + adminOnlyBanner + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printRespSimple(doDeleteHostPolicy(args[0]))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	return cmdDeleteHostPolicy
}

func doCreateHostPolicy(name string, maxResTime string, groups []string, unavailable []string) (*common.ResponseBodyBasic, error) {

	params := map[string]interface{}{"name": name}
	if maxResTime != "" {
		params["maxResTime"] = maxResTime
	}
	if len(groups) > 0 {
		params["accessGroups"] = groups
	}
	if len(unavailable) > 0 {
		var sb []map[string]string
		for _, block := range unavailable {
			s := strings.Split(block, ":")
			if len(s) != 2 {
				return nil, fmt.Errorf("error splitting entry %v did not result in 2 expressions", block)
			} else {
				sb = append(sb, map[string]string{"start": s[0], "duration": s[1]})
			}
		}
		if len(sb) > 0 {
			params["notAvailable"] = sb
		}
	}

	body := doSend(http.MethodPost, api.HostPolicy, params)
	return unmarshalBasicResponse(body), nil
}

func doShowHostPolicy(names []string, groups []string, hosts []string) *common.ResponseBodyPolicies {

	var params string
	if len(names) > 0 {
		for _, n := range names {
			params += "name=" + n + "&"
		}
	}
	if len(groups) > 0 {
		for _, o := range groups {
			params += "accessGroups=" + o + "&"
		}
	}
	if len(hosts) > 0 {
		for _, o := range hosts {
			params += "hosts=" + o + "&"
		}
	}
	if params != "" {
		params = strings.TrimSuffix(params, "&")
		params = "?" + params
	}
	apiPath := api.HostPolicy + params
	body := doSend(http.MethodGet, apiPath, nil)
	rb := common.ResponseBodyPolicies{}
	err := json.Unmarshal(*body, &rb)
	checkUnmarshalErr(err)
	return &rb
}

func doEditHostPolicy(name string, newName string, maxResTime string, groupAdd []string, groupRemove []string, unavailableAdd []string, unavailableRemove []string) (*common.ResponseBodyBasic, error) {
	apiPath := api.HostPolicy + "/" + name
	params := make(map[string]interface{})
	if newName != "" {
		params["name"] = newName
	}
	if maxResTime != "" {
		params["maxResTime"] = maxResTime
	}
	if len(groupAdd) > 0 {
		params["addGroups"] = groupAdd
	}
	if len(groupRemove) > 0 {
		params["removeGroups"] = groupRemove
	}
	if len(unavailableAdd) > 0 {
		var sba []map[string]string
		for _, block := range unavailableAdd {
			s := strings.Split(block, ":")
			if len(s) != 2 {
				return nil, fmt.Errorf("error splitting entry %v did not result in 2 expressions", block)
			} else {
				sba = append(sba, map[string]string{"start": s[0], "duration": s[1]})
			}
		}
		if len(sba) > 0 {
			params["addNotAvailable"] = sba
		}
	}
	if len(unavailableRemove) > 0 {
		var sbr []map[string]string
		for _, block := range unavailableRemove {
			s := strings.Split(block, ":")
			if len(s) != 2 {
				return nil, fmt.Errorf("error splitting entry %v did not result in 2 expressions", block)
			} else {
				sbr = append(sbr, map[string]string{"start": s[0], "duration": s[1]})
			}
		}
		if len(sbr) > 0 {
			params["removeNotAvailable"] = sbr
		}
	}
	body := doSend(http.MethodPatch, apiPath, params)
	return unmarshalBasicResponse(body), nil
}

func doApplyHostPolicy(policyName string, nodeList string) *common.ResponseBodyBasic {
	params := make(map[string]interface{})
	params["policy"] = policyName
	params["nodeList"] = nodeList
	apiPath := api.HostApplyPolicy
	body := doSend(http.MethodPatch, apiPath, params)
	return unmarshalBasicResponse(body)
}

func doDeleteHostPolicy(name string) *common.ResponseBodyBasic {
	apiPath := api.HostPolicy + "/" + name
	body := doSend(http.MethodDelete, apiPath, nil)
	return unmarshalBasicResponse(body)
}

func printPolicies(rb *common.ResponseBodyPolicies) {

	checkAndSetColorLevel(rb)

	hpList := rb.Data["hostPolicies"]
	if len(hpList) == 0 {
		printSimple("no policies to show (yet) or no matches based on search criteria", cRespWarn)
	}

	sort.Slice(hpList, func(i, j int) bool {
		return strings.ToLower(hpList[i].Name) < strings.ToLower(hpList[j].Name)
	})

	if simplePrint {

		var hpinfo string
		for _, hp := range hpList {

			var nas []string
			for _, na := range hp.NotAvailable {
				nas = append(nas, na.ToString())
			}

			maxResTime, _ := time.ParseDuration(hp.MaxResTime)

			hpinfo = "POLICY: " + hp.Name + "\n"
			hpinfo += "  -HOSTS:         " + hp.Hosts + "\n"
			hpinfo += "  -MAX-RES-TIME:  " + common.FormatDuration(maxResTime, true) + "\n"
			hpinfo += "  -ACCESS-GROUPS: " + strings.Join(hp.AccessGroups, ",") + "\n"
			hpinfo += "  -NOT-AVAIL:     " + strings.Join(nas, ",") + "\n"
			fmt.Print(hpinfo + "\n\n")
		}

	} else {

		tw := table.NewWriter()
		tw.AppendHeader(table.Row{"NAME", "HOSTS", "MAX-RES-TIME", "ACCESS-GROUPS", "NOT-AVAIL"})
		tw.AppendSeparator()

		for _, hp := range hpList {

			var nas []string
			for _, na := range hp.NotAvailable {
				nas = append(nas, na.ToString())
			}
			maxResTime, _ := time.ParseDuration(hp.MaxResTime)

			tw.AppendRow([]interface{}{
				hp.Name,
				multilineNodeList(20, hp.Hosts, ""),
				common.FormatDuration(maxResTime, true),
				strings.Join(hp.AccessGroups, "\n"),
				strings.Join(nas, "\n"),
			})
		}

		tw.SetColumnConfigs([]table.ColumnConfig{
			{Name: "HOSTS", WidthMax: 20},
			{Name: "MAX-RES-TIME", Align: text.AlignRight},
			{Name: "KERNEL-ARGS", WidthMax: 40},
		})

		tw.SetStyle(igorTableStyle)
		fmt.Printf("\n" + tw.Render() + "\n\n")
	}

}
