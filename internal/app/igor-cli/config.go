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
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	ConfigLocProdDefault = "/etc/igor/igor.yaml"
)

type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port uint16 `yaml:"port"`
	} `yaml:"server"`
	Client struct {
		CertFile      string `yaml:"certFile"`
		KeyFile       string `yaml:"keyFile"`
		CaCert        string `yaml:"caCert"`
		Timezone      string `yaml:"timezone"`
		AuthLocal     *bool  `yaml:"authLocal"`
		PasswordLabel string `yaml:"passwordLabel"`
	} `yaml:"client"`
}

var (
	cli struct {
		Config
		IgorHome       string
		TokenPath      string
		IgorServerAddr string
		tzLoc          *time.Location
	}

	igorCliNow time.Time
)

func initConfig() {

	// You don't need IGOR_HOME defined for the client unless it's getting its config file from
	// a different path than /etc/igor/.
	if _, errEtc := os.Stat(ConfigLocProdDefault); errEtc == nil {
		cli.Config = readConfig(ConfigLocProdDefault)
	} else {

		if cli.IgorHome = os.Getenv("IGOR_HOME"); strings.TrimSpace(cli.IgorHome) == "" {
			checkClientErr(fmt.Errorf("environment variable IGOR_HOME not defined, but needed to find config file"))
		}

		var configLocHome = cli.IgorHome + "/conf/igor.yaml"

		if _, errHome := os.Stat(configLocHome); errHome == nil {
			cli.Config = readConfig(configLocHome)
		} else {
			checkClientErr(fmt.Errorf("unable to find a config file in any well-known location:\n\tetc loc = %s\n\tIGOR_HOME loc = %s",
				ConfigLocProdDefault, configLocHome))
		}
	}

	initConfigCheck()
}

func readConfig(path string) (c Config) {

	f, err := os.Open(path)
	if err != nil {
		checkClientErr(fmt.Errorf("unable to open config file - %v", err))
	}
	defer f.Close()

	if errDecode := yaml.NewDecoder(f).Decode(&c); errDecode != nil {
		checkClientErr(fmt.Errorf("unable to parse yaml config - %v", errDecode))
	}

	return
}

func initConfigCheck() {
	if cli.Server.Port == 0 {
		cli.Server.Port = 8443
	}

	cli.IgorServerAddr = fmt.Sprintf("https://%s:%d", cli.Server.Host, cli.Server.Port)

	if cli.Client.AuthLocal == nil {
		authType := true
		cli.Client.AuthLocal = &authType
	}

	if cli.Client.PasswordLabel == "" {
		cli.Client.PasswordLabel = "igor"
	}

	if cli.Client.Timezone != "" {
		if loc, tzErr := time.LoadLocation(cli.Client.Timezone); tzErr != nil {
			printSimple(fmt.Sprintf("problem with TZ loc info -- %v", tzErr), cRespWarn)
		} else {
			cli.tzLoc = loc
		}
	} else {
		cli.tzLoc = time.Local
	}

	igorCliNow = getLocTime(time.Now())

	return
}

func newServerConfigCmd() *cobra.Command {

	cmdConfig := &cobra.Command{
		Use:   "settings",
		Short: "View igor server settings",
		Long: `
Displays igor-server settings. The output is in JSON format.
`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			flagset := cmd.Flags()
			setAll := flagset.Changed("all")
			doServerConfig(setAll)
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	var allSettings bool
	cmdConfig.Flags().BoolVarP(&allSettings, "all", "a", false, "get complete settings config "+adminOnly)

	return cmdConfig
}

func doServerConfig(setAll bool) {

	var body *[]byte

	if setAll {
		body = doSend(http.MethodGet, api.Config, nil)
	} else {
		body = doSend(http.MethodGet, api.PublicSettings, nil)
	}
	rb := unmarshalBasicResponse(body)
	if rb.IsSuccess() {
		configData, err := json.MarshalIndent(rb.Data["igor"], "", "   ")
		if err != nil {
			checkClientErr(err)
		}
		printSimple(fmt.Sprint(string(configData)), cRespSuccess)
	} else {
		printRespSimple(rb)
	}
}
