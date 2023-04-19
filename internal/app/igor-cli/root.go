// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"fmt"
	"igor2/internal/pkg/common"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	initConfig()
	cobra.OnInitialize()
}

func Execute() {
	rootCmd := newCmdRoot()
	err := rootCmd.Execute()
	if err != nil {
		checkClientErr(err)
	}
}

func newCmdRoot() *cobra.Command {

	rootCmd := &cobra.Command{
		Use: "igor",
		Long: `
Igor is a scheduler for multi-purpose clusters. Users can reserve and provision
cluster nodes with net-boot or local-boot images of their choice. Users can
also create groups that share access to reservation management. Administrators
can fine-tune reservation creation across a wide variety of settings from open
and permissive to regulated and tightly controlled.

` + sBold("Helpful Notes:") + `

Auto-completion is available using the TAB key or can be enabled by following
instructions found in the 'igor completion' command.

All flags that take arguments can be expressed with or without an equals sign.

  '-f ARG'
  '-f=ARG'

Help on igor topics is available either as a flag or by using it as the second
word in the command.

  'igor res create -h'
  'igor res create --help'
  'igor help res create'

Igor defaults using decorative formatting and color in its output. If you wish
to turn off color, set the NO_COLOR environment variable in your shell or use
-x/--simple flag where available to use ASCII-only, no-color output.
`,
		Run: func(cmd *cobra.Command, args []string) {
			flagSet := cmd.Flags()
			if flagSet.Changed("version") {
				fmt.Println(common.GetVersion("Igor CLI", false))
				os.Exit(0)
			}
		},
	}

	var v bool
	rootCmd.Flags().BoolVarP(&v, "version", "v", false, "version info")

	rootCmd.AddCommand(newElevateCmd())
	rootCmd.AddCommand(newServerConfigCmd())
	rootCmd.AddCommand(newShowCmd())
	rootCmd.AddCommand(newLastCmd())
	rootCmd.AddCommand(newLogoutCmd())
	rootCmd.AddCommand(newUserCmd())
	rootCmd.AddCommand(newGroupCmd())
	rootCmd.AddCommand(newResetSecretCmd())
	rootCmd.AddCommand(newSyncCmd())
	rootCmd.AddCommand(newStatsCmd())
	rootCmd.AddCommand(newClustersCmd())
	rootCmd.AddCommand(newHostCmd())
	rootCmd.AddCommand(newHostPowerCmd()) // adding power command to root menu for user convenience
	rootCmd.AddCommand(newHostPolicyCmd())
	rootCmd.AddCommand(newImageCmd())
	rootCmd.AddCommand(newKSCmd())
	rootCmd.AddCommand(newDistroCmd())
	rootCmd.AddCommand(newProfileCmd())
	rootCmd.AddCommand(newResCmd())
	rootCmd.AddCommand(newCompletionCmd(rootCmd.Name()))

	return rootCmd
}

func newCompletionCmd(rootCmdName string) *cobra.Command {
	return &cobra.Command{
		Use:   "completion {bash|zsh|fish|powershell}",
		Short: "Generate igor auto-completion script",
		Long: fmt.Sprintf(`
Generates shell source scripts that can be used for auto-completion of %[1]s
commands. Once loaded, tap the TAB key once to auto-complete commands or twice
to get suggestions.

To load completions:

Bash:
  
  # current session only use
  $ source <(%[1]s completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ %[1]s completion bash > /etc/bash_completion.d/%[1]s
  # macOS:
  $ %[1]s completion bash > /usr/local/etc/bash_completion.d/%[1]s

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ %[1]s completion zsh > "${fpath[1]}/_%[1]s"

  # You will need to start a new shell for this setup to take effect.

fish:

  $ %[1]s completion fish | source

  # To load completions for each session, execute once:
  $ %[1]s completion fish > ~/.config/fish/completions/%[1]s.fish

PowerShell:

  PS> %[1]s completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> %[1]s completion powershell > %[1]s.ps1
  # and source this file from your PowerShell profile.
`, rootCmdName),
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				_ = cmd.Root().GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				_ = cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				_ = cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				_ = cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}
}
