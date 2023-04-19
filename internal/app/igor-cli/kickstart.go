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

	"github.com/jedib0t/go-pretty/v6/table"

	"igor2/internal/pkg/common"

	"github.com/spf13/cobra"
)

func newKSCmd() *cobra.Command {

	cmdKickstart := &cobra.Command{
		Use:   "kickstart",
		Short: "Perform a kickstart command " + adminOnly,
		Long: `
Kickstart primary command. A sub-command must be invoked to do anything.

A kickstart (or pre-seed) script is a file that's served to booting nodes 
performing a local installation of its OS. The kickstart script provides
the parameters needed to perform the local installation. When creating a
new Distro using a local boot image, a registered kickstart script must 
be referenced to include with the distro. 

The kickstart script can also allow the node to call for a shell script to
execute additional functions or add packages as needed after the main
installation process completes. See Igor's kickstart documentation for further
details and requirements.

` + sBold("All kickstart commands are admin-only.") + `
`,
	}

	cmdKickstart.AddCommand(newKSRegisterCmd())
	cmdKickstart.AddCommand(newKSShowCmd())
	cmdKickstart.AddCommand(newKSEditCmd())
	cmdKickstart.AddCommand(newKSDelCmd())
	return cmdKickstart
}

func newKSRegisterCmd() *cobra.Command {

	cmdRegisterKS := &cobra.Command{
		Use:   "register -k KICKSTART.FILE ",
		Short: "Register kickstart file " + adminOnly,
		Long: `
Upload and register a kickstart file to Igor.

When creating a new distro using a local boot image, the kickstart must be
included and referenced by file name.

` + requiredFlags + `

Use -k flag to specify the name of the kickstart file

` + adminOnlyBanner + `
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			flagset := cmd.Flags()
			ks, _ := flagset.GetString("kickstart")
			res, err := doRegisterKS(ks)
			if err != nil {
				return err
			}
			printRespSimple(res)
			return nil
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	var ks string
	cmdRegisterKS.Flags().StringVarP(&ks, "kickstart", "k", "", "name of the kickstart file to register")
	_ = cmdRegisterKS.MarkFlagRequired("kickstart")
	_ = registerFlagArgsFunc(cmdRegisterKS, "kickstart", []string{"FILENAME"})

	return cmdRegisterKS
}

func newKSShowCmd() *cobra.Command {

	cmdShowKS := &cobra.Command{
		Use:   "show [-x]",
		Short: "Show kickstart information " + adminOnly,
		Long: `
Shows all kickstart information known to igor's database. No parameters are
accepted. Full list of kickstart files are always returned.

` + optionalFlags + `

Use the -x flag to render screen output without pretty formatting.

` + adminOnlyBanner + `
`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			simplePrint = flagset.Changed("simple")
			printKickstart(doShowKS())
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	cmdShowKS.Flags().BoolVarP(&simplePrint, "simple", "x", false, "use simple text output")
	return cmdShowKS

}

func newKSEditCmd() *cobra.Command {

	cmdEditKS := &cobra.Command{
		Use:   "edit NAME -k KICKSTART.FILE ",
		Short: "Replace kickstart file " + adminOnly,
		Long: `
Upload and register a kickstart file to Igor to replace the existing Kickstart file.

When creating or modifying a distro using a local boot image, the kickstart must be
included and referenced by file name.

` + requiredFlags + `

NAME : kickstart name to replace file to

Use -k flag to specify the name of the new kickstart file

` + adminOnlyBanner + `
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flagset := cmd.Flags()
			ks, _ := flagset.GetString("kickstart")
			res, err := doUpdateKS(args[0], ks)
			if err != nil {
				return err
			}
			printRespSimple(res)
			return nil
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	var ks string
	cmdEditKS.Flags().StringVarP(&ks, "kickstart", "k", "", "name of the kickstart file to register")
	_ = cmdEditKS.MarkFlagRequired("kickstart")
	_ = registerFlagArgsFunc(cmdEditKS, "kickstart", []string{"FILENAME"})

	return cmdEditKS
}

func newKSDelCmd() *cobra.Command {

	return &cobra.Command{
		Use:   "del NAME",
		Short: "Delete a kickstart file " + adminOnly,
		Long: `
Deletes an igor kickstart file from the database and designated directory.

` + requiredArgs + `

  NAME : kickstart name

` + notesOnUsage + `

A kickstart cannot be deleted if it is currently associated to an existing distro.
Any distros using the kickstart file must be deleted first.

` + adminOnlyBanner + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printRespSimple(doDeleteKS(args[0]))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}
}

func doRegisterKS(ks string) (*common.ResponseBodyBasic, error) {

	params := map[string]interface{}{}
	params["kickstart"] = openFile(ks)
	body := doSendMultiform(http.MethodPost, api.KickstartRegister, params)
	return unmarshalBasicResponse(body), nil
}

func doShowKS() *common.ResponseBodyKickstarts {
	var params string
	apiPath := api.Kickstarts + params
	body := doSend(http.MethodGet, apiPath, nil)
	rb := common.ResponseBodyKickstarts{}
	err := json.Unmarshal(*body, &rb)
	checkUnmarshalErr(err)
	return &rb
}

func doUpdateKS(name, ks string) (*common.ResponseBodyBasic, error) {
	apiPath := api.Kickstarts + "/" + name
	params := map[string]interface{}{}
	params["kickstart"] = openFile(ks)
	body := doSendMultiform(http.MethodPatch, apiPath, params)
	return unmarshalBasicResponse(body), nil
}

func doDeleteKS(name string) *common.ResponseBodyBasic {
	apiPath := api.Kickstarts + "/" + name
	body := doSend(http.MethodDelete, apiPath, nil)
	return unmarshalBasicResponse(body)
}

func printKickstart(rb *common.ResponseBodyKickstarts) {

	checkAndSetColorLevel(rb)

	ksList := rb.Data["kickstarts"]
	if len(ksList) == 0 {
		printSimple("no kickstart files to show (yet)", cRespWarn)
	}

	sort.Slice(ksList, func(i, j int) bool {
		return strings.ToLower(ksList[i].Name) < strings.ToLower(ksList[j].Name)
	})

	tw := table.NewWriter()
	tw.AppendHeader(table.Row{"NAME", "FILE NAME", "OWNER"})

	for _, ks := range ksList {
		tw.AppendRow([]interface{}{
			ks.Name,
			ks.FileName,
			ks.Owner,
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
