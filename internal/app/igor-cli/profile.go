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

	"github.com/jedib0t/go-pretty/v6/table"

	"igor2/internal/pkg/common"

	"github.com/spf13/cobra"
)

func newProfileCmd() *cobra.Command {

	cmdProfile := &cobra.Command{
		Use:   "profile",
		Short: "Perform a profile command",
		Long: `
Profile primary command. A sub-command must be invoked to do anything.

Profiles allow adding kernel arguments that modify how a distro boots and runs.
A distro's default profile makes no changes and runs the base distro as-is, 
unless the owner edits the default profile's arguments.

Users with access to a distro can create any number of custom profiles. To
apply a different profile, update the reservation's profile field and issue a
power cycle command.

Before using a new profile, verify the details of its associated distro OS to
ensure compatibility.
`,
	}

	cmdProfile.AddCommand(newProfileCreateCmd())
	cmdProfile.AddCommand(newProfileShowCmd())
	cmdProfile.AddCommand(newProfileEditCmd())
	cmdProfile.AddCommand(newProfileDelCmd())
	return cmdProfile
}

func newProfileCreateCmd() *cobra.Command {

	cmdCreateProfile := &cobra.Command{
		Use:   "create NAME DISTRO [ -k \"KARGS\" --desc \"DESCRIPTION\"]",
		Short: "Create a profile",
		Long: `
Creates a new igor profile. A profile is a distro wrapper for adding kernel
arguments to its startup.

Once created, only the owner is allowed to edit or delete the profile.

` + requiredArgs + `

  NAME : profile name
  DISTRO : distro to be used

` + optionalFlags + `

Use the -k flag to add kernel arguments that will be executed after any kernel
arguments specified in the distro, if present. Use a double-quotes around the
field if it contains spaces.

` + descFlagText + `
`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			desc, _ := flagset.GetString("desc")
			kargs, _ := flagset.GetString("kargs")
			res := doCreateProfile(args[0], args[1], desc, kargs)
			printRespSimple(res)
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return []string{"NAME", "DISTRO"}, cobra.ShellCompDirectiveNoFileComp
		},
	}

	var desc, kernelArgs string

	cmdCreateProfile.Flags().StringVar(&desc, "desc", "", "description of the profile")
	cmdCreateProfile.Flags().StringVarP(&kernelArgs, "kargs", "k", "", "kernel arguments to add to the profile")
	_ = registerFlagArgsFunc(cmdCreateProfile, "kargs", []string{"\"KARGS\""})
	_ = registerFlagArgsFunc(cmdCreateProfile, "desc", []string{"\"DESCRIPTION\""})

	return cmdCreateProfile
}

func newProfileShowCmd() *cobra.Command {

	cmdShowProfile := &cobra.Command{
		Use: "show [-n NAME1,NAME2,...] [-o OWNER1,OWNER2,...] [-d DIST1,DIST2,...]\n" +
			"       [-k \"KARGS1\",\"KARGS2\",...] [-x]",
		Short: "Show group information",
		Long: `
Shows profile information, returning matches to specified parameters. If no
parameters are provided then all profiles will be returned.

Output will provide the name of the profile and its owner, name of the
associated distro, and any profile kernel args, if present.

` + optionalFlags + `

Use the -n. -o, -d and -k flags to narrow results. Multiple values for a given
flag should be comma-delimited.

Use the -x flag to render screen output without pretty formatting.
`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			names, _ := flagset.GetStringSlice("names")
			owners, _ := flagset.GetStringSlice("owners")
			kargs, _ := flagset.GetStringSlice("kernel-args")
			distros, _ := flagset.GetStringSlice("distros")
			simplePrint = flagset.Changed("simple")
			printProfiles(doShowProfile(names, owners, kargs, distros))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	var names,
		owners,
		kargs,
		distros []string

	cmdShowProfile.Flags().StringSliceVarP(&names, "names", "n", nil, "search by profile name(s)")
	cmdShowProfile.Flags().StringSliceVarP(&owners, "owners", "o", nil, "search by profile owner(s)")
	cmdShowProfile.Flags().StringSliceVarP(&kargs, "kernel-args", "k", nil, "search by kernel arg(s)")
	cmdShowProfile.Flags().StringSliceVarP(&distros, "distros", "d", nil, "search by distro(s)")
	cmdShowProfile.Flags().BoolVar(&simplePrint, "simple", false, "use simple text output")
	_ = registerFlagArgsFunc(cmdShowProfile, "names", []string{"NAME1"})
	_ = registerFlagArgsFunc(cmdShowProfile, "owners", []string{"OWNER1"})
	_ = registerFlagArgsFunc(cmdShowProfile, "kernel-args", []string{"\"KARGS1\""})
	_ = registerFlagArgsFunc(cmdShowProfile, "distros", []string{"DIST1"})

	return cmdShowProfile
}

func newProfileEditCmd() *cobra.Command {

	cmdEditProfile := &cobra.Command{
		Use:   "edit NAME { [-n NEWNAME] [-k \"KARGS\"] [--desc \"DESCRIPTION\"] }",
		Short: "Edit profile information",
		Long: `
Edits profile information. This can only be done by the profile owner or an 
admin.

` + requiredArgs + `

  NAME : profile name

` + optionalFlags + `

Use the -n flag to re-name the profile.

Use the -k flag to replace the kernel arguments field. Use a double-quotes around
the field if it contains spaces.

` + descFlagText + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			name, _ := flagset.GetString("name")
			desc, _ := flagset.GetString("desc")
			kargs, _ := flagset.GetString("kernel-args")
			printRespSimple(doEditProfile(args[0], name, desc, kargs))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	var name,
		desc,
		kernelArgs string

	cmdEditProfile.Flags().StringVarP(&name, "name", "n", "", "update the profile name")
	cmdEditProfile.Flags().StringVar(&desc, "desc", "", "update the description")
	cmdEditProfile.Flags().StringVarP(&kernelArgs, "kernel-args", "k", "", "update kernel arguments")
	_ = registerFlagArgsFunc(cmdEditProfile, "name", []string{"NAME"})
	_ = registerFlagArgsFunc(cmdEditProfile, "kernel-args", []string{"\"KARGS\""})
	_ = registerFlagArgsFunc(cmdEditProfile, "desc", []string{"\"DESCRIPTION\""})

	return cmdEditProfile
}

func newProfileDelCmd() *cobra.Command {

	cmdDeleteProfile := &cobra.Command{
		Use:   "del NAME",
		Short: "Delete a profile",
		Long: `
Deletes an igor profile. This can only be done by the profile owner or an
admin.

` + requiredArgs + `

  NAME : profile name

` + notesOnUsage + `

A profile cannot be deleted if it is associated with a reservation.
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printRespSimple(doDeleteProfile(args[0]))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	return cmdDeleteProfile
}

func doCreateProfile(name, distro, desc, kargs string) *common.ResponseBodyBasic {

	params := map[string]interface{}{}
	params["name"] = name
	params["distro"] = distro
	if desc != "" {
		params["description"] = desc
	}
	if kargs != "" {
		params["kernelArgs"] = kargs
	}

	body := doSend(http.MethodPost, api.Profiles, params)
	return unmarshalBasicResponse(body)
}

func doShowProfile(names, owners, kargs, distros []string) *common.ResponseBodyProfiles {
	var params string
	if len(names) > 0 {
		for _, n := range names {
			params += "name=" + n + "&"
		}
	}
	if len(owners) > 0 {
		for _, o := range owners {
			params += "owner=" + o + "&"
		}
	}
	if len(kargs) > 0 {
		for _, o := range kargs {
			params += "kernelArgs=" + o + "&"
		}
	}
	if len(distros) > 0 {
		for _, o := range distros {
			params += "distro=" + o + "&"
		}
	}
	if params != "" {
		params = strings.TrimSuffix(params, "&")
		params = "?" + params
	}

	apiPath := api.Profiles + params
	body := doSend(http.MethodGet, apiPath, nil)
	rb := common.ResponseBodyProfiles{}
	err := json.Unmarshal(*body, &rb)
	checkUnmarshalErr(err)
	return &rb
}

func doEditProfile(name, newName, desc, kargs string) *common.ResponseBodyBasic {
	apiPath := api.Profiles + "/" + name
	params := map[string]interface{}{}
	if newName != "" {
		params["name"] = newName
	}
	if desc != "" {
		params["description"] = desc
	}
	if kargs != "" {
		params["kernelArgs"] = kargs
	}

	body := doSend(http.MethodPatch, apiPath, params)
	return unmarshalBasicResponse(body)
}

func doDeleteProfile(name string) *common.ResponseBodyBasic {
	apiPath := api.Profiles + "/" + name

	body := doSend(http.MethodDelete, apiPath, nil)
	return unmarshalBasicResponse(body)
}

func printProfiles(rb *common.ResponseBodyProfiles) {

	checkAndSetColorLevel(rb)

	profileList := rb.Data["profiles"]
	if len(profileList) == 0 {
		printSimple("no profiles to show (yet) or no matches based on search criteria", cRespWarn)
	}

	sort.Slice(profileList, func(i, j int) bool {
		return strings.ToLower(profileList[i].Name) < strings.ToLower(profileList[j].Name)
	})

	if simplePrint {

		var profileInfo string
		for _, d := range profileList {

			profileInfo = "PROFILE: " + d.Name + "\n"
			profileInfo += "  -DESCRIPTION: " + d.Description + "\n"
			profileInfo += "  -OWNER:       " + d.Owner + "\n"
			profileInfo += "  -DISTRO:      " + d.Distro + "\n"
			profileInfo += "  -KERNEL-ARGS: " + d.KernelArgs + "\n"
			fmt.Print(profileInfo + "\n\n")
		}

	} else {

		tw := table.NewWriter()
		tw.AppendHeader(table.Row{"NAME", "DESCRIPTION", "OWNER", "DISTRO", "KERNEL-ARGS"})
		tw.AppendSeparator()

		for _, p := range profileList {

			tw.AppendRow([]interface{}{
				p.Name,
				multiline(35, p.Description),
				p.Owner,
				p.Distro,
				multiline(40, p.KernelArgs),
			})
		}

		tw.SetColumnConfigs([]table.ColumnConfig{
			{
				Name:     "DESCRIPTION",
				WidthMax: 35,
			},
			{
				Name:     "KERNEL-ARGS",
				WidthMax: 40,
			},
		})

		tw.SetStyle(igorTableStyle)
		fmt.Printf("\n" + tw.Render() + "\n\n")
	}

}
