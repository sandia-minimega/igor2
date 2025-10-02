// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"encoding/json"
	"fmt"
	"igor2/internal/pkg/api"
	"igor2/internal/pkg/common"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gookit/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func newClustersCmd() *cobra.Command {

	cmdClusters := &cobra.Command{
		Use:   "cluster",
		Short: "Perform a cluster command " + adminOnly,
		Long: `
Cluster primary command. A sub-command must be invoked to do anything.

Igor sees a cluster as a discrete group of hosts with a name and a
similar hostname pattern. It stores networking information igor used
to issue external commands.

` + sBold("All cluster commands are admin-only.") + `
`,
	}

	cmdClusters.AddCommand(newClusterConfigCmd())
	cmdClusters.AddCommand(newClusterShowCmd())
	cmdClusters.AddCommand(newClusterUpdateMotdCmd())
	return cmdClusters
}

func newClusterConfigCmd() *cobra.Command {

	return &cobra.Command{
		Use:   "config",
		Short: "Configure a cluster " + adminOnly,
		Long: `
Configures a cluster and its nodes or adds new nodes to an existing cluster by
reading the definition file 'igor-clusters.yaml'.

Prior to changing the file you should probably create a backup for historical
purposes, although this is not required. Use 'igor cluster show --dump' to do
this.

` + adminOnlyBanner + `
`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			printRespSimple(doClusterConfig())
		},
		ValidArgsFunction: validateNoArgs,
	}
}

func newClusterUpdateMotdCmd() *cobra.Command {

	cmdClusterUpdateMotd := &cobra.Command{
		Use:   "motd MESSAGE [-u]",
		Short: "Update the cluster MOTD " + adminOnly,
		Long: `
Sets (or unsets) a "message of the day" to be displayed on igor clients.

The MESSAGE argument should be a double-quoted string containing the message
to be displayed when 'igor show' is run. To unset the message use the same 
command with "" as the argument.

` + optionalFlags + `

Supplying the optional -u flag sends a display hint to the cli that the
message should be highlighted in some fashion.

` + adminOnlyBanner + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			urgent := flagset.Changed("urgent")
			printRespSimple(doMotdUpdate(args[0], urgent))
		},

		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return []string{"\"MESSAGE\""}, cobra.ShellCompDirectiveNoFileComp
		},
	}

	var motdUrgent bool
	cmdClusterUpdateMotd.Flags().BoolVarP(&motdUrgent, "urgent", "u", false, "set if message is urgent")

	return cmdClusterUpdateMotd
}

func doMotdUpdate(motd string, urgent bool) *common.ResponseBodyBasic {

	params := map[string]interface{}{}
	params["motd"] = motd
	if urgent {
		params["motdUrgent"] = true
	} else {
		params["motdUrgent"] = false
	}
	body := doSend(http.MethodPatch, api.ClusterMotd, params)
	return unmarshalBasicResponse(body)
}

func newClusterShowCmd() *cobra.Command {

	cmdShowClusters := &cobra.Command{
		Use:   "show [{-d -x | -y}]",
		Short: "Show cluster information " + adminOnly,
		Long: `
Shows information about a cluster's metadata (not its nodes).

` + optionalFlags + `

Use the -d flag to write the current configuration of the cluster as stored in
the database to a new 'igor-clusters.yaml' file, storing the old version as a
timestamped backup file in the same directory. You will still get the normal
output display on the terminal along with a message about file creation.

Use the -x flag to render screen output without pretty formatting.

Use the -y flag to return the text of the current 'igor-clusters.yaml' through
the screen for viewing. You cannot combine this flag with others.

` + adminOnlyBanner + `
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			flagset := cmd.Flags()
			dump := flagset.Changed("dump")
			getYaml := flagset.Changed("yaml")
			simplePrint = flagset.Changed("simple")
			two := getYaml && (dump || simplePrint)
			if two {
				return fmt.Errorf("cannot use yaml flag with other options")
			}
			if getYaml {
				printYaml(doShowYaml([]string{}, getYaml))
			} else {
				printClusters(doShowClusters([]string{}, dump))
			}
			return nil
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	var dump, getYaml bool

	cmdShowClusters.Flags().BoolVarP(&dump, "dump", "d", false, "write the current config to disk")
	cmdShowClusters.Flags().BoolVarP(&getYaml, "yaml", "y", false, "view the existing yaml config")
	cmdShowClusters.Flags().BoolVarP(&simplePrint, "simple", "x", false, "use simple text output")

	return cmdShowClusters
}

func doShowYaml(names []string, getYaml bool) *common.ResponseBodyBasic {
	var params string

	// If we want to support multiple clusters someday
	if len(names) > 0 {
		for _, n := range names {
			params += "name=" + n + "&"
		}
	}
	if getYaml {
		params += "getYaml=true" + "&"
	}
	if params != "" {
		params = strings.TrimSuffix(params, "&")
		params = "?" + params
	}
	apiPath := api.Clusters + params
	body := doSend(http.MethodGet, apiPath, nil)
	rb := common.ResponseBodyBasic{}
	err := json.Unmarshal(*body, &rb)
	checkUnmarshalErr(err)
	return &rb
}

func doShowClusters(names []string, dump bool) *common.ResponseBodyClusters {
	var params string

	// If we want to support multiple clusters someday
	if len(names) > 0 {
		for _, n := range names {
			params += "name=" + n + "&"
		}
	}
	if dump {
		params += "dump=true" + "&"
	}
	if params != "" {
		params = strings.TrimSuffix(params, "&")
		params = "?" + params
	}
	apiPath := api.Clusters + params
	body := doSend(http.MethodGet, apiPath, nil)
	rb := common.ResponseBodyClusters{}
	err := json.Unmarshal(*body, &rb)
	checkUnmarshalErr(err)
	return &rb
}

func doClusterConfig() *common.ResponseBodyBasic {
	body := doSend(http.MethodPost, api.Clusters, nil)
	return unmarshalBasicResponse(body)
}

func printYaml(rb *common.ResponseBodyBasic) {

	checkColorLevel()
	yaml := rb.Data["yaml"]
	color.S256(11).Println(yaml)
}

func printClusters(rb *common.ResponseBodyClusters) {

	checkAndSetColorLevel(rb)

	msg := rb.ResponseBodyBase.Message

	clusters := rb.Data["clusters"]
	if len(clusters) == 0 {
		printSimple("no cluster to show (yet) or no matches based on search criteria", cRespWarn)
	}

	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Name < clusters[j].Name
	})

	if simplePrint {

		var distroInfo string
		for _, d := range clusters {

			distroInfo = "\nNAME: " + d.Name + "\n"
			distroInfo += "      -PREFIX: " + d.Prefix + "\n"
			distroInfo += "  -DISPLAY-WIDTH: " + strconv.Itoa(d.DisplayWidth) + "\n"
			distroInfo += " -DISPLAY-HEIGHT: " + strconv.Itoa(d.DisplayHeight) + "\n"
			distroInfo += " -MOTD-URGENT: " + strconv.FormatBool(d.MotdUrgent) + "\n"
			distroInfo += "        -MOTD: " + d.Motd + "\n"

			if len(msg) > 0 {
				distroInfo = distroInfo + "\n" + msg
			}

			fmt.Print(distroInfo + "\n")
		}

	} else {

		tw := table.NewWriter()
		tw.AppendHeader(table.Row{"NAME", "PREFIX", "DISPLAY-W", "DISPLAY-H", "MOTD-URGENT", "MOTD"})
		tw.AppendSeparator()

		for _, d := range clusters {

			tw.AppendRow([]interface{}{
				d.Name,
				d.Prefix,
				d.DisplayWidth,
				d.DisplayHeight,
				d.MotdUrgent,
				d.Motd,
			})
		}

		tw.SetColumnConfigs([]table.ColumnConfig{
			{
				Name:     "MOTD",
				WidthMax: 50,
			},
		})

		tw.SetStyle(igorTableStyle)

		fmt.Printf("\n" + tw.Render())
		if len(msg) > 0 {
			fmt.Printf("\n\n" + color.FgLightYellow.Sprint(msg) + "\n\n")
		} else {
			fmt.Printf("\n\n")
		}
	}

}
