// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"fmt"
	"igor2/internal/pkg/api"
	"igor2/internal/pkg/common"
	"net/http"

	"github.com/spf13/cobra"
)

func newElevateCmd() *cobra.Command {
	cmdElevate := &cobra.Command{
		Use:   "elevate [{-s | -c}]",
		Short: "Temporarily allow execution of admin commands " + adminOnly,
		Long: `
Grants members of the ` + sBold("admins") + ` group the ability to execute admin commands or
normal commands using parameters that exceed standard limitations (e.g., 
extending reservations beyond max time allowed).

Use the bare command to request elevated mode.

` + optionalFlags + `

The -s flag will show the status of elevated mode for the user and time remaining if active.

The -c flag will cancel any currently applied elevated mode.

` + adminOnlyBanner + `
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			flagset := cmd.Flags()
			var optFlag string
			if setCancel := flagset.Changed("cancel"); setCancel {
				optFlag = "cancel"
			}
			if setStatus := flagset.Changed("status"); setStatus {
				if optFlag == "cancel" {
					return fmt.Errorf("status and cancel flags cannot be used together")
				}
				optFlag = "status"
			}

			printRespSimple(doElevate(optFlag))
			return nil
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	var elevateStatus, elevateCancel bool
	cmdElevate.Flags().BoolVarP(&elevateStatus, "status", "s", false, "get elevate status")
	cmdElevate.Flags().BoolVarP(&elevateCancel, "cancel", "c", false, "cancel elevate privilege")

	return cmdElevate
}

func doElevate(optFlag string) *common.ResponseBodyBasic {

	method := http.MethodPatch

	if optFlag != "" {
		switch optFlag {
		case "cancel":
			method = http.MethodDelete
		case "status":
			method = http.MethodGet
		}
	}

	body := doSend(method, api.Elevate, nil)
	return unmarshalBasicResponse(body)
}
