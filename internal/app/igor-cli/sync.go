// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"igor2/internal/pkg/api"

	"igor2/internal/pkg/common"

	"github.com/gookit/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {

	cmdSync := &cobra.Command{
		Use:   "sync {arista} [-f] [-q] [-s]",
		Short: "Report/repair status of vlan service " + adminOnly,
		Long: `
Displays status and information about the vlan network service based on command
given.

` + requiredArgs + ` (there is only one supported vlan switch type at this time)

    arista :
       For each host currently associated with a reservation, sync will report
       - the vlan value assigned to the host by the switch
       - the vlan value assigned to the host by the reservation
       - whether the reservation is powered

` + optionalFlags + `

Use the -f flag to force host vlan ids in the switch to the value indicated by
the reservation if the values do not match.

Use the -q flag to only report back on hosts whose reservation vlan value does
not match what's reported by the switch.

Use the -s flag to specify a set of hosts on which to sync. Can be either a
name list or range of hosts
	* name list is comma-delimited: kn1,kn2,kn3,...
    * range is the form prefix[n,m-n,...] where m,n are integers representing
      a single or contiguous ranges of hosts, ex. kn[3,7-9,22-35,47]
or one or more reservations as a list
	* reservation list is comma-delimited: MyReserved1,MyReserved2,...
All the hosts included in a reservation will undergo the sync operation.

` + adminOnlyBanner + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			force := flagset.Changed("force")
			quiet := flagset.Changed("quiet")
			scope, _ := flagset.GetString("scope")
			result := doSync(args[0], force, quiet, scope)
			printSync(result)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return []string{"arista"}, cobra.ShellCompDirectiveNoFileComp
		},
	}

	var force bool
	var quiet bool
	var scope string
	cmdSync.Flags().BoolVarP(&force, "force", "f", false, "force sync with authoritative source")
	cmdSync.Flags().BoolVarP(&quiet, "quiet", "q", false, "only report objects out of sync")
	cmdSync.Flags().StringVarP(&scope, "scope", "s", "", "scope of hosts or reservation to perform sync")

	return cmdSync
}

func doSync(cmd string, force, quiet bool, scope string) *common.ResponseBodySync {
	var params string
	params += "cmd=" + cmd + "&"
	if force {
		params += "force=true" + "&"
	}
	if quiet {
		params += "quiet=true" + "&"
	}
	if scope != "" {
		params += "scope=" + scope + "&"
	}
	if params != "" {
		params = strings.TrimSuffix(params, "&")
		params = "?" + params
	}

	apiPath := api.Sync + params
	body := doSend(http.MethodGet, apiPath, nil)

	rb := common.ResponseBodySync{}
	err := json.Unmarshal(*body, &rb)
	checkUnmarshalErr(err)

	return &rb
}

func printSync(rb *common.ResponseBodySync) {

	if !rb.IsSuccess() {
		printRespSimple(rb)
	}

	syncData := rb.Data["sync"].(map[string]interface{})
	command := syncData["command"].(string)
	force := syncData["force"].(string) == "true"
	quiet := syncData["quiet"].(string) == "true"
	report := syncData["report"].(map[string]interface{})

	cRespSuccess.Printf("sync performed on: %s\n", command)
	if len(report) == 0 {
		printSimple("no hosts are in use, nothing to report", cRespSuccess)
	} else {
		pgt := table.NewWriter()
		headers := table.Row{"HOST", "POWERED", "RESERVATION VLAN", "SWITCH VLAN"}
		if force {
			headers = append(headers, "FORCE RESULTS")
		}
		pgt.AppendHeader(headers)
		fmt.Printf("NOTE - Igor examines only hosts currently engaged in an active reservation\n\n")
		for node, data := range report {
			nodeReportData := data.(map[string]interface{})
			powered := nodeReportData["powered"].(string)
			var poweredColor string
			if powered == "off" {
				poweredColor = color.S256(15, 9).Sprint(powered)
			} else {
				poweredColor = color.FgLightGreen.Sprint(powered)
			}
			resVlan := nodeReportData["res_vlan"].(string)
			switchVlan := nodeReportData["switch_vlan"].(string)
			mismatch := switchVlan != resVlan
			var switchVlanColor string
			if mismatch {
				switchVlanColor = color.S256(15, 9).Sprint(switchVlan)
			} else {
				switchVlanColor = color.FgLightGreen.Sprint(switchVlan)
			}
			if !quiet || mismatch {
				row := []interface{}{node, poweredColor, resVlan, switchVlanColor}
				if force && mismatch {
					status := nodeReportData["status"].(string)
					row = append(row, status)
				}
				pgt.AppendRow(row)
			}
		}
		pgt.SetStyle(table.StyleLight)
		pgt.Style().Options.DrawBorder = false

		fmt.Println(pgt.Render())
		if quiet && pgt.Length() == 0 {
			printSimple("no hosts were out of sync", cRespSuccess)
		}
	}
}
