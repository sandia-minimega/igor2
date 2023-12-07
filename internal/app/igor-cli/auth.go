// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"igor2/internal/pkg/api"
	"igor2/internal/pkg/common"
	"io"
	"net/http"
	"os"
	"os/user"
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

// CLIENT COMMANDS...

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Starts a new auth session",
		Long: `
Gets a valid authentication token for the user. This action will ask for the
user's account credentials when executed.
`,
		RunE: func(cmd *cobra.Command, args []string) error {

			osUser, osErr := user.Current()
			if osErr != nil {
				return osErr
			}

			username, password, rucErr := reqUserCreds(osUser)
			if rucErr != nil {
				return rucErr
			}

			response, lErr := doLogin(username, password)
			if lErr != nil {
				return lErr
			}
			printRespSimple(response)
			return nil
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}
}

func doLogin(username string, password string) (*common.ResponseBodyBasic, error) {

	req, _ := http.NewRequest("GET", cli.IgorServerAddr+api.Login, nil)
	req.SetBasicAuth(username, password)
	setUserAgent(req)
	lastAccessUser = username

	resp := sendRequest(req)
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		checkClientErr(readErr)
	}

	cookies := resp.Cookies()
	for i, c := range cookies {
		if c.Name == "auth_token" {
			if err := writeAuthToken(cookies[i]); err != nil {
				return nil, err
			}
			if err := writeLastAccessUser(); err != nil {
				fmt.Printf("%v\n", err)
			}
		}
	}
	return unmarshalBasicResponse(&body), nil
}

// these client commands don't call the server

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
