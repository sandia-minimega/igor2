// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorweb

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
)

const (
	IgorConfHome        = "/etc/igor/"
	IgorConfFileDefault = "igor-web.yaml"
	IgorConfPathDefault = IgorConfHome + IgorConfFileDefault
)

type Config struct {
	WebServer struct {
		Host     string `yaml:"host"`
		Port     uint   `yaml:"port"`
		CertFile string `yaml:"certFile"`
		KeyFile  string `yaml:"keyFile"`
		FileDir  string `yaml:"fileDir"`
	} `yaml:"webserver"`

	Log struct {
		Dir    string `yaml:"dir"`
		File   string `yaml:"file"`
		Level  string `yaml:"level"`
		Syslog struct {
			Network string `yaml:"network"`
			Addr    string `yaml:"addr"`
		} `yaml:"syslog"`
	} `yaml:"log"`
}

func getHostFQDN() (string, error) {
	cmd := exec.Command("hostname", "-f")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	fqdn := out.String()
	fqdn = fqdn[:len(fqdn)-1]
	return fqdn, nil
}

// initConfig searches for the config in well-known locations and will use the first one it finds in order
// of preference:
//
// - the filepath passed into command-line args
//
// - the hard-coded /etc configuration filepath
//
// - the hard-coded relative filepath under IGOR_HOME
//
// This first option is desirable to override an existing configuration defined by the others if there
// is a need to do so.
func initConfig(configFilepath string) {

	var configLocHome = filepath.Join(igorweb.IgorHome, "conf", IgorConfFileDefault)

	if _, errArg := os.Stat(configFilepath); errArg == nil {
		fmt.Printf("looking for conf file in %s\n", configFilepath)
		igorweb.Config = readConfig(configFilepath)
		igorweb.ConfigPath = configFilepath
	} else if _, errEtc := os.Stat(IgorConfPathDefault); errEtc == nil {
		fmt.Printf("looking for conf file in %s\n", IgorConfPathDefault)
		igorweb.Config = readConfig(IgorConfPathDefault)
		igorweb.ConfigPath = IgorConfPathDefault
	} else if _, errHome := os.Stat(configLocHome); errHome == nil {
		fmt.Printf("looking for conf file in %s\n", configLocHome)
		igorweb.Config = readConfig(configLocHome)
		igorweb.ConfigPath = configLocHome
	} else {
		if configFilepath == "" {
			configFilepath = "(none specified)"
		}

		errMsg := "unable to find a config file in any well-known location: \n\tconfig arg = " + configFilepath +
			"\n\tetc loc = " + IgorConfPathDefault +
			"\n\tIGOR_HOME loc = " + configLocHome + "\n"
		exitPrintFatal(errMsg)
	}
}

// Read in the configuration from the specified path. Checks to make sure that
// the config is owned and only writable by the effective user to ensure that
// users can't try to specify their own config when we're running with setuid.
func readConfig(path string) (c Config) {

	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "igorweb read config: unable to open config file: %v", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		exitPrintFatal(fmt.Sprintf("unable to stat config file: %v", err))
	}

	switch fi := fi.Sys().(type) {
	case *syscall.Stat_t:
		euid := syscall.Geteuid()
		if fi.Uid != uint32(euid) {
			exitPrintFatal(fmt.Sprintf("config file must be owned by running user"))
		}

		if fi.Mode&0400 == 0 || fi.Mode&0200 == 0 || fi.Mode&0040 == 0 || (fi.Mode&0020 == 0 && fi.Mode&0002 != 0) || fi.Mode&0007 != 0 {
			exitPrintFatal(fmt.Sprintf("check permissions on YAML config file - must be 0660 or 0640"))
		}

	default:
		fmt.Fprintln(os.Stderr, "igorweb read config: unable to check config ownership/permissions")
	}

	if err = yaml.NewDecoder(f).Decode(&c); err != nil {
		exitPrintFatal(fmt.Sprintf("config error: unable to parse yaml: %v", err))
	}

	return
}

func initConfigCheck() {
	logger.Info().Msgf("server settings loaded from %s", igorweb.ConfigPath)
	logger.Info().Msg("--- begin: config file settings")
	printConfigToLog(igorweb.Config, "igorweb.Config.")

	logger.Warn().Msg("--- begin: applying defaults and overrides")

	if igorweb.WebServer.Host == "" {
		if hostFQDN, hostErr := getHostFQDN(); hostErr != nil {
			exitPrintFatal(fmt.Sprintf("config error: FQDN could not be obtained - %v - see server hostname setting", hostErr))
		} else {
			logger.Info().Msgf("setting hostname to FQDN : %s", hostFQDN)
			igorweb.WebServer.Host = hostFQDN
		}
	}

	if igorweb.WebServer.Port == 0 {
		igorweb.WebServer.Port = 3000
		logger.Info().Msgf("server port not specified; using default : %d", igorweb.WebServer.Port)
	}

	if igorweb.WebServer.FileDir == "" {
		igorweb.WebServer.FileDir = "../../web/dist"
		logger.Info().Msgf("folder for web content not specified; using default for development: %s", igorweb.WebServer.FileDir)
	}

	if _, err := os.Stat(igorweb.WebServer.FileDir); os.IsNotExist(err) {
		exitPrintFatal(fmt.Sprintf("config error: web app folder '%s' doesn't exist -- aborting", igorweb.WebServer.FileDir))
	}

	logger.Warn().Msg("--- end: applying defaults and overrides")
	logger.Info().Msg("--- end: config file settings")
}

// printConfigToLog iterates through the given interface recursively to find all settings in
// all child data structures and send them to the log.
func printConfigToLog(s interface{}, namePrefix string) {

	v := reflect.ValueOf(s)

	for i := 0; i < v.NumField(); i++ {

		p := v.Type().Field(i).Type.Kind().String()
		finalName := namePrefix + v.Type().Field(i).Name

		if p == "struct" {
			n := v.Type().Field(i).Name
			name := namePrefix + n + "."
			printConfigToLog(v.Field(i).Interface(), name)
		} else if p == "map" {
			iter := v.Field(i).MapRange()
			for iter.Next() {
				logger.Info().Msgf("%s : %v = %v", finalName, iter.Key(), iter.Value())
			}
		} else {
			if v.Field(i).Kind() == reflect.Ptr {
				if v.Field(i).IsNil() {
					// format output of pointer to nil
					logger.Warn().Msgf("%s = <nil>", finalName)
				} else {
					// format output of de-referenced pointer
					logger.Info().Msgf("%s = %v", finalName, v.Field(i).Elem())
				}
			} else {
				field := v.Field(i).Interface()
				if strings.Contains(strings.ToLower(finalName), "password") {
					// format output of password field
					logger.Info().Msgf("%s = *****", finalName)
				} else {
					// format everything else
					logger.Info().Msgf("%s = %v", finalName, field)
				}
			}
		}
	}
}
