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
	"os/user"
	"sort"
	"strings"
	"time"

	"igor2/internal/pkg/common"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func newUserCmd() *cobra.Command {

	cmdUser := &cobra.Command{
		Use:   "user",
		Short: "Perform a user command",
		Long: `
User primary command. A sub-command must be invoked to do anything.

Users are tracked within igor independent of the OS. They are not required to 
have an account on the system where igor server is installed, but they will 
need to recognized by LDAP on that node if that feature is enabled.
`,
	}

	cmdUser.AddCommand(newUserCreateCmd())
	cmdUser.AddCommand(newUserShowCmd())
	cmdUser.AddCommand(newUserEditCmd())
	cmdUser.AddCommand(newUserDelCmd())
	cmdUser.AddCommand(newResetPassCmd())

	return cmdUser
}

func newUserCreateCmd() *cobra.Command {

	cmdCreateUser := &cobra.Command{

		Short: "Create a user " + adminOnly,
		Long: `
Creates a new igor user. An email will be sent to the user informing them of
this event.

` + requiredArgs + `

  NAME : user account name
  EMAIL : user's email address

` + notesOnUsage + `

This is a required step before a user can interact with igor regardless of the
configured authentication method. A user name should match the one on the 
network where the cli is running if LDAP authentication is enabled. It is
highly recommended to match for local authentication but is not required.

` + optionalFlags + `

The -f flag provides a more user-readable name and should be enclosed
in quotes if it contains spaces. It can be up to 32 characters long. This
value with NOT replace the user's login name.

` + adminOnlyBanner + `
`,
		Use:  "create NAME EMAIL [-f \"FULLNAME\"]",
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			fullName, _ := flagset.GetString("full-name")
			printRespSimple(doCreateUser(args[0], args[1], fullName))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return []string{"NAME", "EMAIL"}, cobra.ShellCompDirectiveNoFileComp
		},
	}

	var fullName string
	cmdCreateUser.Flags().StringVarP(&fullName, "full-name", "f", "", "include a more readable name")
	_ = registerFlagArgsFunc(cmdCreateUser, "full-name", []string{"\"FULLNAME\""})
	return cmdCreateUser
}

func newUserShowCmd() *cobra.Command {

	cmdShowUsers := &cobra.Command{
		Use:   "show [-a] [-n NAME1,NAME2,...] [-x]",
		Short: "Show user information",
		Long: `
Shows igor user information. Without optional flags this command will only 
display the user's information.

Normal users are allowed to see the names of other users and the date they
they were granted access to igor. Email and group membership of other users
are viewable by elevated admins.

` + optionalFlags + `

Use the -a flag to show all users.

Use the -n flag to filter users by name.

Use the -x flag to render screen output without pretty formatting.
`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			names, _ := flagset.GetStringSlice("names")
			showAll := flagset.Changed("all")
			simplePrint = flagset.Changed("simple")

			if len(names) > 0 {
				// is searching by name, show all to display the returned results
				showAll = true
			}
			printShowUsers(doShowUsers(names), showAll)
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	var names []string
	var showAll bool
	cmdShowUsers.Flags().StringSliceVarP(&names, "names", "n", nil, "comma-separated user list")
	cmdShowUsers.Flags().BoolVarP(&showAll, "all", "a", false, "show all users")
	cmdShowUsers.Flags().BoolVarP(&simplePrint, "simple", "x", false, "use simple text output")

	_ = registerFlagArgsFunc(cmdShowUsers, "names", []string{"NAME1,NAME2"})

	return cmdShowUsers
}

func newUserEditCmd() *cobra.Command {

	cmdEditUser := &cobra.Command{
		Use:   "edit { -e EMAIL -f \"FULLNAME\" (-n NAME) | --password } ",
		Short: "Edit user information",
		Long: `
Allows editing user information.

` + requiredFlags + `

  -e : Changes the user's email address.
    >> AND/OR <<
  -f : Changes the full name (enclose in double-quotes if using spaces).

  >> OR <<

  --password : Initiates a local password change (prompts will follow).

` + notesOnUsage + `

Users are allowed to change their email address igor will use to send them
notifications. They may also change their password if igor is using local user
authentication. Igor passwords must be 8-16 chars in length and have a minimum
of 1 letter, 1 number and 1 symbol. Choose a password according to organization
best practices and do not re-use sensitive passwords from other systems.

` + sItalic("The password change option will not work if LDAP authentication is configured.") + `

If a user cannot remember their password they should request an igor admin to
reset it. See 'igor user reset -h' for more information. Admins may only reset
another user's password, not change it.

` + sBold("IMPORTANT:") + `

By default this command will use the last known successful igor login to obtain
the username that is being edited. This can be checked with the 'igor last'
command. Use the -n flag to override this behavior.

Admins can change another user's email address and/or full name field provided
they include the -n flag.
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {

			var name string
			if osUser, err := user.Current(); err != nil {
				return err
			} else if name, err = readLastAccessUser(osUser); err != nil {
				return err
			}

			flagset := cmd.Flags()
			username, _ := flagset.GetString("name")
			if username != "" {
				name = username
			}

			email, _ := flagset.GetString("email")
			fullName, _ := flagset.GetString("full-name")
			changePass := flagset.Changed("password")
			printRespSimple(doEditUser(name, email, fullName, changePass))
			return nil
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	var email,
		fullName,
		name string
	var changePass bool
	cmdEditUser.Flags().StringVarP(&email, "email", "e", "", "update user email address")
	cmdEditUser.Flags().StringVarP(&fullName, "full-name", "f", "", "update user full name")
	cmdEditUser.Flags().StringVarP(&name, "name", "n", "", "target user name")
	cmdEditUser.Flags().BoolVar(&changePass, "password", false, "initiate local password change")

	_ = registerFlagArgsFunc(cmdEditUser, "email", []string{"EMAIL"})
	_ = registerFlagArgsFunc(cmdEditUser, "full-name", []string{"FULLNAME"})
	_ = registerFlagArgsFunc(cmdEditUser, "name", []string{"NAME"})

	return cmdEditUser
}

func newResetPassCmd() *cobra.Command {

	cmdResetPassword := &cobra.Command{
		Use:   "reset NAME",
		Short: "Reset user password " + adminOnly,
		Long: `
Resets an existing igor user's password to the default if authentication is set
to use local accounts. An email will be sent to the user informing them of this
event.

` + sItalic("This command will not work if LDAP authentication is configured.") + `

` + requiredArgs + `

  NAME : user account name

` + adminOnlyBanner + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printRespSimple(doResetUserPswd(args[0]))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	return cmdResetPassword
}

func newUserDelCmd() *cobra.Command {

	cmdDeleteUser := &cobra.Command{
		Use:   "del NAME",
		Short: "Delete a user " + adminOnly,
		Long: `
Deletes an igor user. An email will be sent to the user informing them of this
event.

` + requiredArgs + `

  NAME : user account name

` + notesOnUsage + `

This cannot be performed if the user is an owner of a reservation, group or
distro. These resources must be deleted first or transferred to a new owner.

Deleting a user has no disparate impact other than denying access to igor. It
does not affect any underlying OS user account.

` + adminOnlyBanner + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printRespSimple(doDeleteUser(args[0]))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	return cmdDeleteUser
}

func doCreateUser(name string, email string, fullName string) *common.ResponseBodyBasic {

	params := map[string]interface{}{"name": name, "email": email}
	if fullName != "" {
		params["fullName"] = fullName
	}
	body := doSend(http.MethodPost, api.Users, params)
	return unmarshalBasicResponse(body)
}

func doEditUser(name string, email string, fullName string, changePswd bool) *common.ResponseBodyBasic {

	apiPath := api.Users + "/" + name
	changes := make(map[string]interface{})

	if changePswd {

		if !*cli.Client.AuthLocal && name != "igor-admin" {
			pDeny := "password not managed by igor"
			if cli.Client.PasswordLabel != "igor" {
				pDeny = cli.Client.PasswordLabel + " " + pDeny
			}
			printSimple(pDeny, cRespWarn)
		}

		oPswd, nPswd, err := reqPassChange(name)
		if err != nil {
			checkClientErr(err)
		}
		changes["oldPassword"] = oPswd
		changes["password"] = nPswd
	}

	if email != "" {
		changes["email"] = email
	}

	if fullName != "" {
		changes["fullName"] = fullName
	}

	body := doSend(http.MethodPatch, apiPath, changes)
	uBody := unmarshalBasicResponse(body)
	if changePswd && uBody.IsSuccess() {
		os.Remove(getAuthTokenPath())
	}
	return uBody
}

func doResetUserPswd(name string) *common.ResponseBodyBasic {

	if !*cli.Client.AuthLocal && name != "igor-admin" {
		pDeny := "password not managed by igor"
		if cli.Client.PasswordLabel != "igor" {
			pDeny = cli.Client.PasswordLabel + " " + pDeny
		}
		printSimple(pDeny, cRespWarn)
	}

	apiPath := api.Users + "/" + name
	body := doSend(http.MethodPatch, apiPath, map[string]interface{}{"reset": true})
	return unmarshalBasicResponse(body)
}

func doShowUsers(names []string) *common.ResponseBodyUsers {
	var params string
	if len(names) > 0 {
		for _, n := range names {
			params += "name=" + n + "&"
		}
	}
	if params != "" {
		params = strings.TrimSuffix(params, "&")
		params = "?" + params
	}

	apiPath := api.Users + params
	body := doSend(http.MethodGet, apiPath, nil)
	rb := common.ResponseBodyUsers{}
	err := json.Unmarshal(*body, &rb)
	checkUnmarshalErr(err)
	return &rb
}

func doDeleteUser(name string) *common.ResponseBodyBasic {
	apiPath := api.Users + "/" + name
	body := doSend(http.MethodDelete, apiPath, nil)
	return unmarshalBasicResponse(body)
}

func printShowUsers(rb *common.ResponseBodyUsers, showAll bool) {

	checkAndSetColorLevel(rb)

	users := rb.Data["users"]
	if len(users) == 0 {
		printSimple("no users matches based on search criteria", cRespWarn)
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].Name < users[j].Name
	})

	tw := table.NewWriter()
	tw.AppendHeader(table.Row{"NAME", "FULL NAME", "JOINED", "EMAIL", "GROUPS"})

	for _, u := range users {

		// only display user's info unless showAll was set
		if lastAccessUser != u.Name && !showAll {
			continue
		}

		var groups string
		var joinTime string
		if simplePrint {
			groups = strings.Join(u.Groups, ",")
			joinTime = getLocTime(time.Unix(u.JoinDate, 0)).Format("Jan-02-2006")
		} else {
			groups = strings.Join(u.Groups, "\n")
			joinTime = getLocTime(time.Unix(u.JoinDate, 0)).Format("Jan 02 2006")
		}

		tw.AppendRow([]interface{}{
			u.Name,
			u.FullName,
			joinTime,
			u.Email,
			groups,
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
