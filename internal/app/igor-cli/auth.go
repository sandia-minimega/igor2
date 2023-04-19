// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"errors"
	"igor2/internal/pkg/api"
	"igor2/internal/pkg/common"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

func newResetSecretCmd() *cobra.Command {

	cmdJwtReset := &cobra.Command{
		Use:   "auth-reset",
		Short: "Generate a new global token secret " + adminOnly,
		Long: `
Deletes the existing secret used to sign JWT tokens and generates a new one. 
This will invalidate all existing tokens, forcing all users to re-authenticate.

` + adminOnlyBanner + `
`,
		Run: func(cmd *cobra.Command, args []string) {
			printRespSimple(doJwtReset())
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	return cmdJwtReset
}

func doJwtReset() *common.ResponseBodyBasic {
	body := doSend(http.MethodPut, api.AuthReset, nil)
	return unmarshalBasicResponse(body)
}

// CLIENT COMMANDS... these don't call the server

func newLogoutCmd() *cobra.Command {

	return &cobra.Command{
		Use:   "logout",
		Short: "Ends the current auth session",
		Long: `
Deletes the user's cached login token. Igor will request login credentials on
the following command after executing this one.
`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := os.Remove(getAuthTokenPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
				checkClientErr(err)
			} else {
				printSimple("auth session closed", cRespSuccess)
			}
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}
}

func newLastCmd() *cobra.Command {

	return &cobra.Command{
		Use:   "last",
		Short: "Print last access information",
		Long: `
Prints out the last known time this user successfully contacted the igor server
and the username when making that call.
`,
		Run: func(cmd *cobra.Command, args []string) {
			if access, err := getLastAccessInfo(); err != nil {
				checkClientErr(err)
			} else {
				printSimple(access, cRespSuccess)
			}
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}
}
