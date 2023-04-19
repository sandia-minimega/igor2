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
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func newGroupCmd() *cobra.Command {

	cmdGroup := &cobra.Command{
		Use:   "group",
		Short: "Perform a group command",
		Long: `
Group primary command. A sub-command must be invoked to do anything.

Groups are collections of users that are granted exclusive access to resources
on igor that are created by individuals. This can be OS images, reservations 
and even certain cluster nodes if allowed by admins.

The user who creates a group manages its members and can transfer ownership to
another igor user if desired.`,
	}

	cmdGroup.AddCommand(newGroupCreateCmd())
	cmdGroup.AddCommand(newGroupShowCmd())
	cmdGroup.AddCommand(newGroupEditCmd())
	cmdGroup.AddCommand(newGroupDelCmd())

	return cmdGroup
}

func newGroupCreateCmd() *cobra.Command {

	cmdCreateGroup := &cobra.Command{
		Use:   "create NAME [-m USER1,USER2...] [--desc \"DESCRIPTION\"]",
		Short: "Create a group",
		Long: `
Creates a new igor group.

A group is a collection of users that can be given access to various resources
created by other igor users. Using groups is completely optional. When this
action is performed an email will be sent out to members.

Once created only the owner is allowed to edit or delete the group.

` + requiredArgs + `

  NAME : group name

` + optionalFlags + `

Use the -m flag to specify a comma-delimited list of members. The list must
accurately match existing igor users. The group creator does not need to add
their name to the member list, thus if a list is not provided the group creator
will be the only member.

` + descFlagText + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			desc, _ := flagset.GetString("desc")
			members, _ := flagset.GetStringSlice("members")
			printRespSimple(doCreateGroup(args[0], desc, members))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	var desc string
	var members []string
	cmdCreateGroup.Flags().StringVarP(&desc, "desc", "", "", "description of the group")
	cmdCreateGroup.Flags().StringSliceVarP(&members, "members", "m", nil, "members to add to group")
	_ = registerFlagArgsFunc(cmdCreateGroup, "desc", []string{"\"DESCRIPTION\""})
	_ = registerFlagArgsFunc(cmdCreateGroup, "members", []string{"USER1"})

	return cmdCreateGroup

}

func newGroupShowCmd() *cobra.Command {

	cmdShowGroups := &cobra.Command{
		Use:   "show [{-n USER1,... | -o OWNER1,...}] [-m] [-x]",
		Short: "Show group information",
		Long: `
Shows group information. If no optional parameters are provided then all groups
will be returned.

` + optionalFlags + `

Use the -n and -o flags to narrow results. Multiple values for a given flag
should be comma-delimited.

Use the -m flag to display members in the group. (Can result in long output.)

Use the -x flag to render screen output without pretty formatting.
`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			names, _ := flagset.GetStringSlice("names")
			owners, _ := flagset.GetStringSlice("owners")
			showMembers := flagset.Changed("members")
			simplePrint = flagset.Changed("simple")
			printShowGroups(doShowGroups(names, owners, showMembers))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	var names,
		owners []string
	var showMembers bool
	cmdShowGroups.Flags().StringSliceVarP(&names, "names", "n", nil, "search by group name(s)")
	cmdShowGroups.Flags().StringSliceVarP(&owners, "owners", "o", nil, "search by owner name(s)")
	cmdShowGroups.Flags().BoolVarP(&showMembers, "members", "m", false, "include members in output")
	cmdShowGroups.Flags().BoolVarP(&simplePrint, "simple", "x", false, "use simple text output")

	_ = registerFlagArgsFunc(cmdShowGroups, "names", []string{"USER1"})
	_ = registerFlagArgsFunc(cmdShowGroups, "owners", []string{"OWNER1"})

	return cmdShowGroups
}

func newGroupEditCmd() *cobra.Command {

	cmdEditGroup := &cobra.Command{
		Use: "edit NAME { [-n NEWNAME] [-o OWNER] [-a USER1,...] [-r USER1,...]\n" +
			"                [--desc \"DESCRIPTION\"] }",

		Short: "Edit group information",
		Long: `
Edits group information. This can only be done by the group owner or an admin.
Changes to the group are emailed out to those affected.

` + requiredArgs + `

  NAME : group name

` + optionalFlags + `

Use the -n flag to re-name of the group.

Use the -o flag to transfer ownership to another user. After this the original
owner can no longer edit the group. The new owner is added to the group if they
are not already a member.

Use the -a flag to add a list of users to the group. ` + sItalic("Note: adding a member to\n"+
			"the 'admins' group gives that user igor admin privileges.") + `

Use the -r flag to remove a list of users from the group. This can be combined
with the -o flag to both transfer ownership and completely remove themselves
from the group at the same time.

` + descFlagText + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			name, _ := flagset.GetString("name")
			desc, _ := flagset.GetString("desc")
			owner, _ := flagset.GetString("owner")
			add, _ := flagset.GetStringSlice("add")
			remove, _ := flagset.GetStringSlice("remove")
			printRespSimple(doEditGroup(args[0], name, owner, desc, add, remove))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	var name,
		desc,
		owner string
	var names,
		owners []string
	cmdEditGroup.Flags().StringVarP(&name, "name", "n", "", "update the group name")
	cmdEditGroup.Flags().StringVar(&desc, "desc", "", "update the description of the group")
	cmdEditGroup.Flags().StringVarP(&owner, "owner", "o", "", "new owner (user) name")
	cmdEditGroup.Flags().StringSliceVarP(&names, "add", "a", nil, "comma-delimited users to add")
	cmdEditGroup.Flags().StringSliceVarP(&owners, "remove", "r", nil, "comma-delimited users to remove")
	_ = registerFlagArgsFunc(cmdEditGroup, "name", []string{"NAME"})
	_ = registerFlagArgsFunc(cmdEditGroup, "desc", []string{"\"DESCRIPTION\""})
	_ = registerFlagArgsFunc(cmdEditGroup, "owner", []string{"OWNER"})
	_ = registerFlagArgsFunc(cmdEditGroup, "add", []string{"USER1"})
	_ = registerFlagArgsFunc(cmdEditGroup, "remove", []string{"USER1"})

	return cmdEditGroup
}

func newGroupDelCmd() *cobra.Command {

	cmdDeleteGroup := &cobra.Command{
		Use:   "del NAME",
		Short: "Delete a group",
		Long: `
Deletes an igor group. This can only be done by the group owner or an admin.

` + requiredArgs + `

  NAME : group name

` + notesOnUsage + `

A group cannot be deleted if it is attached to a host policy. The policy must
be edited to remove the group or deleted prior to running this command.
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printRespSimple(doDeleteGroup(args[0]))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	return cmdDeleteGroup

}

func doCreateGroup(name string, desc string, members []string) *common.ResponseBodyBasic {

	params := map[string]interface{}{}
	params["name"] = name
	if desc != "" {
		params["description"] = desc
	}
	if len(members) > 0 {
		params["members"] = members
	}
	body := doSend(http.MethodPost, api.Groups, params)
	return unmarshalBasicResponse(body)
}

func doShowGroups(names []string, owners []string, showMembers bool) *common.ResponseBodyGroups {

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
	if showMembers {
		params += "showMembers=true"
	}
	if params != "" {
		params = strings.TrimSuffix(params, "&")
		params = "?" + params
	}
	apiPath := api.Groups + params
	body := doSend(http.MethodGet, apiPath, nil)
	rb := common.ResponseBodyGroups{}
	err := json.Unmarshal(*body, &rb)
	checkUnmarshalErr(err)
	return &rb
}

func doEditGroup(name string, newName string, owner string, desc string, add []string, remove []string) *common.ResponseBodyBasic {
	apiPath := api.Groups + "/" + name
	params := make(map[string]interface{})
	if newName != "" {
		params["name"] = newName
	}
	if owner != "" {
		params["owner"] = owner
	}
	if desc != "" {
		params["description"] = desc
	}
	if len(add) > 0 {
		params["add"] = add
	}
	if len(remove) > 0 {
		params["remove"] = remove
	}

	body := doSend(http.MethodPatch, apiPath, params)
	return unmarshalBasicResponse(body)
}

func doDeleteGroup(name string) *common.ResponseBodyBasic {
	apiPath := api.Groups + "/" + name
	body := doSend(http.MethodDelete, apiPath, nil)
	return unmarshalBasicResponse(body)
}

func printShowGroups(rb *common.ResponseBodyGroups) {

	checkAndSetColorLevel(rb)

	owned := rb.Data["owner"]
	notOwned := rb.Data["member"]

	// for now just group everything together
	var groupList []common.GroupData
	groupList = append(groupList, owned...)
	groupList = append(groupList, notOwned...)

	if len(groupList) == 0 {
		printSimple("no group matches based on search criteria", cRespWarn)
	}

	sort.Slice(groupList, func(i, j int) bool {
		return strings.ToLower(groupList[i].Name) < strings.ToLower(groupList[j].Name)
	})

	if simplePrint {

		var groupInfo string
		for i, g := range groupList {

			if g.Name == "all" {
				continue
			}

			var members string
			if len(g.Members) == 0 {
				members = "<not shown>"
			} else {
				members = strings.Join(g.Members, ",")
			}

			groupInfo = "GROUP: " + g.Name + "\n"
			groupInfo += "  -DESCRIPTION:  " + g.Description + "\n"
			groupInfo += "  -OWNER:        " + g.Owner + "\n"
			groupInfo += "  -MEMBERS:      " + members + "\n"
			groupInfo += "  -DISTROS:      " + strings.Join(g.Distros, ",") + "\n"
			groupInfo += "  -RESERVATIONS: " + strings.Join(g.Reservations, ",") + "\n"
			groupInfo += "  -POLICIES:     " + strings.Join(g.Policies, ",") + "\n\n"
			if i < len(owned)-1 {
				groupInfo += "-------------------------------\n\n"
			}
			fmt.Print(groupInfo)
		}

	} else {

		tw := table.NewWriter()
		tw.AppendHeader(table.Row{"NAME", "DESCRIPTION", "OWNER", "MEMBERS", "DISTROS", "RESERVATIONS", "POLICIES"})

		for _, g := range groupList {

			if g.Name == "all" {
				continue
			}

			var members string
			if len(g.Members) == 0 {
				members = "<not shown>"
			} else {
				members = strings.Join(g.Members, "\n")
			}

			tw.AppendRow([]interface{}{
				g.Name,
				g.Description,
				g.Owner,
				members,
				strings.Join(g.Distros, "\n"),
				strings.Join(g.Reservations, "\n"),
				strings.Join(g.Policies, "\n"),
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
