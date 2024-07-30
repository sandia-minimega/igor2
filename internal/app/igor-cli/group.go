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
		Use: "create NAME {[-o USER1,USER2...] [-m USER3,USER4...]\n" +
			"                [--desc \"DESCRIPTION\"] | -L }",
		Short: "Create a group",
		Long: `
Creates a new igor group.

A group is a collection of users that can be given access to various resources
created by other igor users. Using groups is completely optional. When this
action is performed an email will be sent out to members.

Two types of groups are supported by igor: LDAP-synced and igor-only.

LDAP-synced groups derive their information (owners, members, etc.) by looking
up the group name via an LDAP service and matching the usernames it contains
against igor's internal list of users. Once created, an LDAP-synced group will
only periodically check the server for changes and update itself accordingly.
It cannot be edited through the Igor interface, but it can be dropped by its
owner or admin when it is no longer needed.

Igor-only groups are completely created and maintained within Igor. This option
is the default.

` + requiredArgs + `

  NAME : group name (a local name or matching LDAP name)

` + optionalFlags + `

Use the -o flag to specify a comma-delimited list of additional owners who will
have full permissions to edit and delete the group. The user creating a group
is automatically an owner. Users listed here will be treated as members, so 
they don't need to be listed twice if the -m flag is also used.

Use the -m flag to specify a comma-delimited list of non-owning group members.
Members of this kind gain access to group-restricted resources but cannot edit
the group itself.

` + descFlagText + `

Use the -L flag to specify the group as LDAP-sync enabled. It cannot be used
with other flags. Additionally, you must have owner or delegate permissions on
the LDAP group itself in order to use this flag successfully. The command will
fail if the user running the group creation command lacks this permission.
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			isLDAP, _ := flagset.GetBool("LDAP")
			desc, _ := flagset.GetString("desc")
			members, _ := flagset.GetStringSlice("members")
			owners, _ := flagset.GetStringSlice("owners")
			printRespSimple(doCreateGroup(args[0], isLDAP, desc, owners, members))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	var desc string
	var isLDAP bool
	var members []string
	var owners []string
	cmdCreateGroup.Flags().StringVarP(&desc, "desc", "", "", "description of the group")
	cmdCreateGroup.Flags().BoolVarP(&isLDAP, "LDAP", "L", false, "sync with LDAP group of same name")
	cmdCreateGroup.Flags().StringSliceVarP(&owners, "owners", "o", nil, "owners to add to the group")
	cmdCreateGroup.Flags().StringSliceVarP(&members, "members", "m", nil, "members to add to group")
	_ = registerFlagArgsFunc(cmdCreateGroup, "desc", []string{"\"DESCRIPTION\""})
	_ = registerFlagArgsFunc(cmdCreateGroup, "members", []string{"USER1"})
	_ = registerFlagArgsFunc(cmdCreateGroup, "owners", []string{"OWNER1"})

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
		Use: "edit NAME [-n NEWNAME] {[-o OWNER1,...] [-w OWNER1,...] | \n" +
			"                [-a MEMBER1,...] [-r MEMBER1,...]} [--desc \"DESCRIPTION\"]",
		Short: "Edit group information",
		Long: `
Edits group information. This can only be done by the group owner or an admin.
Changes to the group are emailed out to those affected.

` + notesOnUsage + `

This command cannot be used on an LDAP-synced group. Modify the group's proper-
ties using the network's LDAP interface instead.

` + requiredArgs + `

  NAME : group name

` + optionalFlags + `

Use the -n flag to re-name of the group.

Use the -o flag to add a list of users as group owners. If any new owner is not
already a member of the group they will be added to the membership list.

Use the -w flag to remove a list of users as group owners. Owners will remain 
members. This action cannot remove all owners of a group. There must be at
least one owner.

Use the -a flag to add a list of users to the group. ` + sItalic("Note: adding a member to\n"+
			"the 'admins' group gives that user igor admin privileges.") + `

Use the -r flag to remove a list of users from the group.



` + descFlagText + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			name, _ := flagset.GetString("name")
			desc, _ := flagset.GetString("desc")
			addOwners, _ := flagset.GetStringSlice("add-owners")
			rmvOwners, _ := flagset.GetStringSlice("rmv-owners")
			add, _ := flagset.GetStringSlice("add")
			remove, _ := flagset.GetStringSlice("remove")
			printRespSimple(doEditGroup(args[0], name, addOwners, rmvOwners, desc, add, remove))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	var name,
		desc string
	var addUsers,
		rmvUsers,
		addOwners,
		rmvOwners []string
	cmdEditGroup.Flags().StringVarP(&name, "name", "n", "", "update the group name")
	cmdEditGroup.Flags().StringVar(&desc, "desc", "", "update the description of the group")
	cmdEditGroup.Flags().StringSliceVarP(&addOwners, "add-owners", "o", nil, "comma-delimited owners to add")
	cmdEditGroup.Flags().StringSliceVarP(&rmvOwners, "rmv-owners", "w", nil, "comma-delimited owners to remove")
	cmdEditGroup.Flags().StringSliceVarP(&addUsers, "add", "a", nil, "comma-delimited users to add")
	cmdEditGroup.Flags().StringSliceVarP(&rmvUsers, "remove", "r", nil, "comma-delimited users to remove")
	_ = registerFlagArgsFunc(cmdEditGroup, "name", []string{"NAME"})
	_ = registerFlagArgsFunc(cmdEditGroup, "desc", []string{"\"DESCRIPTION\""})
	_ = registerFlagArgsFunc(cmdEditGroup, "add-owners", []string{"OWNER1"})
	_ = registerFlagArgsFunc(cmdEditGroup, "rmv-owners", []string{"OWNER1"})
	_ = registerFlagArgsFunc(cmdEditGroup, "add", []string{"USER1"})
	_ = registerFlagArgsFunc(cmdEditGroup, "remove", []string{"USER1"})

	return cmdEditGroup
}

func newGroupDelCmd() *cobra.Command {

	cmdDeleteGroup := &cobra.Command{
		Use:   "del NAME",
		Short: "Delete a group",
		Long: `
Deletes an igor group. This can only be done by a group owner or an admin.

` + notesOnUsage + `

A group cannot be deleted if it is attached to a host policy. The policy must
be edited to remove the group or deleted prior to running this command.

When used on an LDAP-synced group, igor only deletes its internal references to
the group. It does not affect the LDAP group service object itself. 

` + requiredArgs + `

  NAME : group name

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

func doCreateGroup(name string, isLDAP bool, desc string, owners []string, members []string) *common.ResponseBodyBasic {

	params := map[string]interface{}{}
	params["name"] = name
	if isLDAP {
		params["isLDAP"] = true
	}
	if desc != "" {
		params["description"] = desc
	}
	if len(owners) > 0 {
		params["owners"] = owners
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

func doEditGroup(name string, newName string, addOwners []string, rmvOwners []string, desc string, add []string, remove []string) *common.ResponseBodyBasic {
	apiPath := api.Groups + "/" + name
	params := make(map[string]interface{})
	if newName != "" {
		params["name"] = newName
	}
	if len(addOwners) > 0 {
		params["addOwners"] = addOwners
	}
	if len(rmvOwners) > 0 {
		params["rmvOwners"] = rmvOwners
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

			var members, owners string
			if len(g.Members) == 0 {
				members = "<not shown>"
			} else {
				members = strings.Join(g.Members, ",")
			}
			if len(g.Owners) == 1 {
				owners = g.Owners[0]
			} else {
				owners = strings.Join(g.Owners, ",")
			}

			groupInfo = "GROUP: " + g.Name + "\n"
			groupInfo += "  -DESCRIPTION:  " + g.Description + "\n"
			groupInfo += "  -OWNERS:       " + owners + "\n"
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
		tw.AppendHeader(table.Row{"NAME", "DESCRIPTION", "OWNERS", "MEMBERS", "DISTROS", "RESERVATIONS", "POLICIES"})

		for _, g := range groupList {

			if g.Name == "all" {
				continue
			}

			var members, owners string
			if len(g.Members) == 0 {
				members = "<not shown>"
			} else {
				members = strings.Join(g.Members, "\n")
			}
			if len(g.Owners) == 1 {
				owners = g.Owners[0]
			} else {
				owners = strings.Join(g.Owners, "\n")
			}

			tw.AppendRow([]interface{}{
				g.Name,
				g.Description,
				owners,
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
