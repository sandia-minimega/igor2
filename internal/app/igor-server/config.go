// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"time"

	"igor2/internal/pkg/common"

	"gopkg.in/yaml.v3"
)

const (
	IgorConfHome               = "/etc/igor/"
	IgorConfFileDefault        = "igor-server.yaml"
	IgorConfPathDefault        = IgorConfHome + IgorConfFileDefault
	IgorClusterConfDefault     = "igor-clusters.yaml"
	IgorClusterConfPathDefault = IgorConfHome + IgorClusterConfDefault
	IgorCliPrefix              = "IgorCLI"
	InsomniaPrefix             = "insomnia"
	MaxScheduleDays            = 1457 // 4 years in days including 1 leap day
	MaxReserveMinutes          = 2098080
	DefaultReserveTime         = 60
	DefaultMinReserveTime      = 30
	DefaultMaxReserveTime      = 43200
	LowestMinReserveTime       = 10
	DefaultExtendWithin        = 4320
)

var (
	MaxScheduleMinutes int
)

// Config - The configuration of the system
type Config struct {
	InstanceName string `yaml:"instanceName" json:"instanceName"`

	Server struct {
		Host             string   `yaml:"host" json:"host"`
		CbHost           string   `yaml:"cbHost" json:"cbHost"`
		Port             int      `yaml:"port" json:"port"`
		CbPort           int      `yaml:"cbPort" json:"cbPort"`
		CertFile         string   `yaml:"certFile" json:"certFile"`
		KeyFile          string   `yaml:"keyFile" json:"keyFile"`
		CbUseTLS         *bool    `yaml:"cbUseTLS,omitempty" json:"cbUseTLS"`
		AllowedOrigins   []string `yaml:"allowedOrigins" json:"allowedOrigins"`
		DNSServer        string   `yaml:"dnsServer" json:"dnsServer"`
		AllowPublicShow  bool     `yaml:"allowPublicShow" json:"allowPublicShow"`
		AllowImageUpload bool     `yaml:"allowImageUpload" json:"allowImageUpload"`
		TFTPRoot         string   `yaml:"tftpRoot" json:"tftpRoot"`
		ImageStagePath   string   `yaml:"imageStagePath" json:"imageStagePath"`
		ScriptDir        string   `yaml:"scriptDir" json:"scriptDir"`
		UserLocalBootDC  bool     `yaml:"userLocalBootDC" json:"userLocalBootDC"`
	} `yaml:"server" json:"server"`

	Auth struct {
		Scheme              string `yaml:"scheme,omitempty" json:"scheme"`
		TokenDuration       int    `yaml:"tokenDuration" json:"tokenDuration"`
		DefaultUserPassword string `yaml:"defaultUserPassword"  json:"-"`
		ElevateTimeout      int    `yaml:"elevateTimeout" json:"elevateTimeout"`

		Ldap struct {
			// Host: LDAP server host
			Host string `yaml:"host" json:"host"`
			// PORT: LDAP server port
			Port string `yaml:"port" json:"port"`
			// useTLS: default value is false. useTLS is ignored if scheme=ldaps
			UseTLS bool `yaml:"useTLS" json:"useTLS"`
			// tlsConfig: is used for either SSL connection (LDAPS) and tls connection, whichever is chosen
			// if a cert is not specified, InsecureSkipVerify: true will be used
			TLSConfig struct {
				// tlsCheckPeer: default=true. Set false to use InsecureSkipVerify: true
				TLSCheckPeer bool `yaml:"tlsCheckPeer" json:"tlsCheckPeer"`
				// TLS uses the host's root CA set. If ldap cert cannot be included there, a cert.pem path can be
				// specified which will be read in and added to root CA set
				// Cert: path to ldap-cert.pem
				Cert string `yaml:"cert" json:"cert"`
			} `yaml:"tlsConfig" json:"tlsConfig"`
			// BindDN: represents LDAP DN for searching for the user DN.
			// Typically read only user DN.
			BindDN string `yaml:"bindDN" json:"bindDN"`
			// BindPassword: LDAP password for searching for the user DN.
			// Typically read only user password.
			BindPassword string `yaml:"bindPassword"  json:"-"`
			// Attributes: used for users.
			Attributes []string `yaml:"attributes" json:"attributes"`
			// BaseDN: LDAP domain to use for users.
			BaseDN string `yaml:"baseDN" json:"baseDN"`
			// Filter: for the User Object Filter.
			// if username nedded more than once use fmt index pattern (%[1]s).
			// Otherwise %s.
			Filter    string `yaml:"filter" json:"filter"`
			GroupSync struct {
				// enableGroupSync: default=false Enable user sync feature
				EnableGroupSync bool `yaml:"enableGroupSync" json:"enableGroupSync"`
				// syncFrequency: default=60 Minutes to wait between running sync actions
				SyncFrequency int `yaml:"syncFrequency" json:"syncFrequency"`
				// groupFilter default=blank - for the Group Object Filter
				GroupFilter string `yaml:"groupFilter" json:"groupFilter"`
				// groupAttribute default=blank - the key for the Entity Attribute value which holds the usernames for all members of the group
				GroupAttribute string `yaml:"groupAttribute" json:"groupAttribute"`
				// groupAttributeEmail default=blank - the key for the Entity Attribute email Value.
				GroupMemberAttributeEmail string `yaml:"groupMemberAttributeEmail" json:"groupMemberAttributeEmail"`
				// groupAttributeDisplayName default=blank - the key for the Entity Attribute display name Value.
				GroupMemberAttributeDisplayName string `yaml:"groupMemberAttributeDisplayName" json:"groupMemberAttributeDisplayName"`
			} `yaml:"groupSync" json:"groupSync"`
		} `yaml:"ldap" json:"ldap"`
	} `yaml:"auth" json:"auth"`

	// Database defines which type of database Gorm should interact with
	// Current supported types are MySQL, PostgreSQL, SQLite, SQL Server
	// See https://gorm.io/docs/connecting_to_the_database.html for details
	Database struct {
		Adapter      string `yaml:"adapter" json:"adapter"`
		DbFolderPath string `yaml:"dbFolderPath" json:"dbFolderPath"` // only used for SQLite
	} `yaml:"database" json:"database"`

	Log struct {
		Dir    string `yaml:"dir" json:"dir"`
		File   string `yaml:"file" json:"file"`
		Level  string `yaml:"level" json:"level"`
		Syslog struct {
			Network string `yaml:"network" json:"network"`
			Addr    string `yaml:"addr" json:"addr"`
		} `yaml:"syslog" json:"syslog"`
	} `yaml:"log" json:"log"`

	Scheduler struct {
		NodeReserveLimit int `yaml:"nodeReserveLimit" json:"nodeReserveLimit"`
		MaxScheduleDays  int `yaml:"maxScheduleDays" json:"maxScheduleDays"`
		// MinReserveTime: min time any user can reserve nodes. This cannot be set lower than 10 minutes.
		MinReserveTime int64 `yaml:"minReserveTime" json:"minReserveTime"`
		// DefaultReserveTime: default time a reservation will be set if the duration value isn't in the request.
		// This cannot be less than MinReserveTime.
		DefaultReserveTime int64 `yaml:"defaultReserveTime" json:"defaultReserveTime"`
		// MaxReserveTime: max time a non-admin user can reserve per reservation
		MaxReserveTime int64 `yaml:"maxReserveTime" json:"maxReserveTime"`

		// ExtendWithin is the number of minutes before the end of a reservation
		// that it can be extended. For example, 24*60 would mean that the
		// reservation can be extended within 24 hours of its expiration.
		ExtendWithin int `yaml:"extendWithin" json:"extendWithin"`
	} `yaml:"scheduler" json:"scheduler"`

	Vlan struct {
		// Network: selects which type of switch is in use. Set to "" to disable VLAN segmentation
		Network string `yaml:"network" json:"network"`

		// NetworkUser/NetworkPassword: login info for a switch user capable of configuring ports
		NetworkUser     string `yaml:"networkUser" json:"networkUser"`
		NetworkPassword string `yaml:"networkPassword" json:"-"`

		// NetworkURL: HTTP URL for sending API commands to the switch
		NetworkURL string `yaml:"networkURL" json:"networkURL"`

		// VLAN segmentation options
		// Min/Max: specify a range of VLANs to use
		RangeMin int `yaml:"rangeMin" json:"rangeMin"`
		RangeMax int `yaml:"rangeMax" json:"rangeMax"`
	} `yaml:"vlan" json:"vlan"`

	Email struct {
		SmtpServer    string `yaml:"smtpServer" json:"smtpServer"`
		SmtpPort      int    `yaml:"smtpPort" json:"smtpPort"`
		SmtpUsername  string `yaml:"smtpUsername" json:"smtpUsername"`
		SmtpPassword  string `yaml:"smtpPassword"  json:"-"`
		ReplyTo       string `yaml:"replyTo" json:"replyTo"`
		HelpLink      string `yaml:"helpLink" json:"helpLink"`
		DefaultSuffix string `yaml:"defaultSuffix" json:"defaultSuffix"`
		ResNotifyOn   *bool  `yaml:"resNotifyOn" json:"resNotifyOn"`
		// The number of minutes a warning emails should be sent prior to a reservation expiring.
		ResNotifyTimes string `yaml:"resNotifyTimes" json:"resNotifyTimes"`
	} `yaml:"email" json:"email"`

	Maintenance struct {
		HostMaintenanceDuration int `yaml:"hostMaintenanceDuration" json:"hostMaintenanceDuration"`
	} `yaml:"maintenance" json:"maintenance"`

	ExternalCmds struct {
		ConcurrencyLimit uint   `yaml:"concurrencyLimit" json:"concurrencyLimit"`
		CommandRetries   uint   `yaml:"commandRetries" json:"commandRetries"`
		PowerOn          string `yaml:"powerOn" json:"powerOn"`
		PowerOff         string `yaml:"powerOff" json:"powerOff"`
		PowerCycle       string `yaml:"powerCycle" json:"powerCycle"`
	} `yaml:"externalCmds" json:"externalCmds"`
}

func (c *Config) splitRange(s string) []string {
	var sr []string
	var err error

	for _, r := range igor.ClusterRefs {
		sr, err = r.SplitRange(s)
		if sr != nil {
			return sr
		}
	}
	logger.Error().Msgf("%v", err)
	return nil
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
// - the hard-coded etc conf filepath
//
// - the hard-coded relative filepath under IGOR_HOME
//
// This first option is desirable to override an existing configuration defined by the others if there
// is a need to do so.
func initConfig(configFilepath string) {

	configLocHome := filepath.Join(igor.IgorHome, "conf", IgorConfFileDefault)

	if configFilepath == "" {
		fmt.Printf("no external config filepath provided\n")
		configFilepath = "(none specified)"
	} else {
		fmt.Printf("looking for conf file at location %s\n", configFilepath)
		if _, errArg := os.Stat(configFilepath); errArg == nil {
			igor.Config = readConfig(configFilepath)
			igor.ConfigPath = configFilepath
			return
		}
	}

	fmt.Printf("looking for conf file at location %s\n", IgorConfPathDefault)
	if _, errEtc := os.Stat(IgorConfPathDefault); errEtc == nil {
		igor.Config = readConfig(IgorConfPathDefault)
		igor.ConfigPath = IgorConfPathDefault
		return
	}

	fmt.Printf("looking for conf file at location %s\n", configLocHome)
	if _, errHome := os.Stat(configLocHome); errHome == nil {
		igor.Config = readConfig(configLocHome)
		igor.ConfigPath = configLocHome
		return
	}

	errMsg := "unable to find a config file in any well-known location: \n\tconfig arg = " + configFilepath +
		"\n\tetc loc = " + IgorConfPathDefault +
		"\n\tIGOR_HOME loc = " + configLocHome + "\n"
	exitPrintFatal(errMsg)

}

// Read in the configuration from the specified path. Checks to make sure that
// the config is owned and only writable by the effective user to ensure that
// users can't try to specify their own config when we're running with setuid.
func readConfig(path string) (c Config) {

	fmt.Printf("found conf file at %s\n", path)
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "igor read config: unable to open config file: %v", err)
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
		fmt.Fprintln(os.Stderr, "igor read config: unable to check config ownership/permissions")
	}

	fmt.Printf("reading in conf file at %s\n", path)
	if err = yaml.NewDecoder(f).Decode(&c); err != nil {
		exitPrintFatal(fmt.Sprintf("config error - unable to parse yaml: %v", err))
	}
	fmt.Printf("conf file read success\n")

	return
}

func initConfigCheck() {
	logger.Info().Msgf("config settings loaded from %s", igor.ConfigPath)
	logger.Info().Msg("--- begin: config file settings")
	printConfigToLog(igor.Config, "igor.")

	logger.Warn().Msg("--- begin: important notes and applying defaults/overrides")

	if igor.InstanceName == "" {
		logger.Info().Msg("instanceName not specified; using default : igor")
		igor.InstanceName = "igor"
	}

	if igor.Server.Host == "" {
		logger.Info().Msgf("server.host not specified ... attempting to set to host FQDN")
		if hostFQDN, hostErr := getHostFQDN(); hostErr != nil {
			exitPrintFatal(fmt.Sprintf("config error - FQDN could not be obtained - %v - see server hostname setting", hostErr))
		} else {
			logger.Info().Msgf("server.host set to FQDN : %s", hostFQDN)
			igor.Server.Host = hostFQDN
		}
	}

	if igor.Server.CbHost == "" {
		logger.Info().Msgf("server.cbHost not specified ... attempting to set to host FQDN")
		if cbHostFQDN, cbHostErr := getHostFQDN(); cbHostErr != nil {
			exitPrintFatal(fmt.Sprintf("config error - FQDN could not be obtained - %v - see server hostname setting", cbHostErr))
		} else {
			logger.Info().Msgf("server.cbHost set to FQDN : %s", cbHostFQDN)
			igor.Server.CbHost = cbHostFQDN
		}
	}

	if igor.Server.Port == 0 {
		igor.Server.Port = 8443
		logger.Info().Msgf("server.port not specified; using default : %d", igor.Server.Port)
	}

	if igor.Server.CbPort == 0 {
		igor.Server.CbPort = igor.Server.Port + 1
		logger.Info().Msgf("server.cbPort not specified; using default : %d", igor.Server.CbPort)
	}

	if len(igor.Server.CertFile) == 0 {
		exitPrintFatal("config error - server.certFile required but not specified")
	}

	if len(igor.Server.KeyFile) == 0 {
		exitPrintFatal("config error - server.keyFile required but not specified")
	}

	if igor.Server.CbUseTLS == nil {
		useTLS := true
		igor.Server.CbUseTLS = &useTLS
	}

	if igor.Server.AllowPublicShow {
		logger.Info().Msgf("public reservation info is enabled")
	}

	if igor.Server.AllowImageUpload {
		logger.Info().Msgf("users are allowed to upload OS images")
	}

	if igor.Server.UserLocalBootDC {
		logger.Info().Msgf("Local Boot Distro Creation is enabled for non-admin users")
	}

	// TFTPRoot path
	if igor.Server.TFTPRoot == "" {
		logger.Warn().Msgf("server.tftpRoot not specified, using default (IGOR_HOME) : %v", igor.IgorHome)
		igor.Server.TFTPRoot = igor.IgorHome
	}

	igor.TFTPPath = igor.Server.TFTPRoot
	igor.PXEBIOSDir = "pxelinux.cfg"
	igor.PXEUEFIDir = filepath.Join("efi", "boot")
	igor.ImageStoreDir = "igor_images"
	igor.KickstartDir = "kickstarts"

	// pxe rep paths for bios + igor backup
	tftprep := filepath.Join(igor.TFTPPath, igor.PXEBIOSDir, "igor")
	if _, err := os.Stat(tftprep); errors.Is(err, os.ErrNotExist) {
		logger.Warn().Msgf("TFTP BIOS repository path(s) not found, creating directory")
		createErr := os.MkdirAll(tftprep, 0755)
		if createErr != nil {
			logger.Error().Msgf("TFTP repository path creation failure: %v", createErr)
		}
	}

	// same for uefi
	tftuefiprep := filepath.Join(igor.TFTPPath, igor.PXEUEFIDir, "igor")
	if _, err := os.Stat(tftuefiprep); errors.Is(err, os.ErrNotExist) {
		logger.Warn().Msgf("TFTP UEFI repository path(s) not found, creating directory")
		createErr := os.MkdirAll(tftuefiprep, 0755)
		if createErr != nil {
			logger.Error().Msgf("TFTP repository path creation failure: %v", createErr)
		}
	}

	logger.Info().Msgf("TFTP root path established: %v", igor.TFTPPath)
	logger.Info().Msgf("BIOS cfg repository established: %v", tftprep)
	logger.Info().Msgf("UEFI boot repository established: %v", tftuefiprep)

	// kickstart rep path
	ksPath := filepath.Join(igor.TFTPPath, igor.KickstartDir)
	if _, err := os.Stat(ksPath); errors.Is(err, os.ErrNotExist) {
		logger.Warn().Msgf("Kickstart repository path not found, creating directory")
		createErr := os.MkdirAll(ksPath, 0755)
		if createErr != nil {
			logger.Error().Msgf("Kickstart repository path creation failure: %v", createErr)
		}
	}
	logger.Info().Msgf("Kickstart repository established: %v", ksPath)

	// image store path
	imageStorePath := filepath.Join(igor.TFTPPath, igor.ImageStoreDir)
	if _, err := os.Stat(imageStorePath); errors.Is(err, os.ErrNotExist) {
		createErr := os.MkdirAll(imageStorePath, 0755)
		if createErr != nil {
			exitPrintFatal(fmt.Sprintf("config error - could not create image repository path %s - %v (admin may need to create directory manually)", imageStorePath, createErr))
		}
	}
	logger.Info().Msgf("Image Store repository established: %v", imageStorePath)

	// image stage path
	if igor.Server.ImageStagePath != "" {
		igor.Server.ImageStagePath = filepath.Join(igor.Server.ImageStagePath, "igor_staged_images")
		if _, err := os.Stat(igor.Server.ImageStagePath); errors.Is(err, os.ErrNotExist) {
			createErr := os.MkdirAll(igor.Server.ImageStagePath, 0755)
			if createErr != nil {
				logger.Warn().Msgf("could not establish Image stage Path at %s - %v", igor.Server.ImageStagePath, createErr)
				igor.Server.ImageStagePath = ""
			}
		} else if err != nil {
			igor.Server.ImageStagePath = ""
		}
	} else {
		igor.Server.ImageStagePath = filepath.Join(igor.IgorHome, "igor_staged_images")
		logger.Warn().Msgf("using default (IGOR_HOME) for staged images: %v", igor.IgorHome)
		if _, err := os.Stat(igor.Server.ImageStagePath); errors.Is(err, os.ErrNotExist) {
			createErr := os.MkdirAll(igor.Server.ImageStagePath, 0755)
			if createErr != nil {
				exitPrintFatal(fmt.Sprintf("config error - could not create %s - %v", igor.Server.ImageStagePath, createErr))
			}
		}
	}
	logger.Info().Msgf("image staging repository established: %v", igor.Server.ImageStagePath)

	if igor.Server.ScriptDir != "" {
		igor.Server.ScriptDir = filepath.Join(igor.Server.ScriptDir, "scripts")
		if _, err := os.Stat(igor.Server.ScriptDir); errors.Is(err, os.ErrNotExist) {
			createErr := os.MkdirAll(igor.Server.ScriptDir, 0755)
			if createErr != nil {
				logger.Warn().Msgf("could not create script directory at given path %s - %v", igor.Server.ScriptDir, createErr)
				igor.Server.ScriptDir = ""
			}
		} else if err != nil {
			igor.Server.ScriptDir = ""
		}
	} else {
		igor.Server.ScriptDir = filepath.Join(igor.IgorHome, "scripts")
		logger.Warn().Msgf("server.scriptDir not specified, using default (IGOR_HOME) : %v", igor.Server.ScriptDir)
		if _, err := os.Stat(igor.Server.ScriptDir); errors.Is(err, os.ErrNotExist) {
			createErr := os.MkdirAll(igor.Server.ScriptDir, 0755)
			if createErr != nil {
				exitPrintFatal(fmt.Sprintf("config error - could not create %s - %v", igor.Server.ScriptDir, createErr))
			}
		}
	}

	if len(igor.Auth.Scheme) == 0 {
		igor.Auth.Scheme = "local"
		logger.Warn().Msgf("auth.scheme not specified so using local authentication, LDAP is disabled")
	} else if strings.EqualFold(igor.Auth.Scheme, "local") {
		igor.Auth.Scheme = "local"
		logger.Info().Msgf("igor is using local authentication, LDAP is disabled")
	}

	if igor.Auth.DefaultUserPassword == "" {
		igor.Auth.DefaultUserPassword = DefaultLocalUserPassword
	} else if igor.Auth.Scheme == "local" {
		logger.Info().Msg("igor default user password set from server config file")
	}

	if igor.Auth.TokenDuration <= 0 {
		logger.Info().Msgf("auth.tokenDuration duration not specified, using default (in hours) : %d", DefaultTokenDuration)
		igor.Auth.TokenDuration = DefaultTokenDuration
	} else if igor.Auth.TokenDuration > MaxTokenDuration {
		logger.Warn().Msgf("auth.tokenDuration max value exceeded, using max value (in hours) : %d", MaxTokenDuration)
		igor.Auth.TokenDuration = MaxTokenDuration
	}

	if igor.Auth.ElevateTimeout < 1 || igor.Auth.ElevateTimeout > 1440 {
		igor.Auth.ElevateTimeout = 10
		logger.Warn().Msgf("auth.elevateTimeout not in legal range (1-1440), using default : %d", igor.Auth.ElevateTimeout)
	}

	if strings.HasPrefix(igor.Auth.Scheme, "ldap") {
		if igor.Auth.Ldap.Host == "" {
			exitPrintFatal(fmt.Sprintf("config error - LDAP auth scheme set but no LDAP hostname specified"))
		}

		if len(igor.Auth.Ldap.Port) == 0 {
			if igor.Auth.Scheme == "ldap" {
				igor.Auth.Ldap.Port = "389"
				logger.Warn().Msgf("ldap.port assignment not specified, using default : %v", igor.Auth.Ldap.Port)
			} else if igor.Auth.Scheme == "ldaps" {
				igor.Auth.Ldap.Port = "636"
				logger.Warn().Msgf("ldap.port assignment not specified, using default : %v", igor.Auth.Ldap.Port)
			} else if igor.Auth.Scheme == "ldapi" {
				igor.Auth.Ldap.Port = "0"
				logger.Warn().Msgf("ldap.port assignment not specified, using default : %v", igor.Auth.Ldap.Port)
			}
		}

		if igor.Auth.Ldap.GroupSync.EnableGroupSync {
			if igor.Auth.Ldap.GroupSync.SyncFrequency <= 0 {
				igor.Auth.Ldap.GroupSync.SyncFrequency = 60
			}
			if len(igor.Auth.Ldap.GroupSync.GroupFilter) == 0 {
				exitPrintFatal(fmt.Sprintf("config error - GroupFilter must have a value when LDAP-GroupSync is enabled"))
			}
			if igor.Auth.Ldap.GroupSync.GroupMemberAttributeEmail == "" && igor.Email.DefaultSuffix == "" {
				exitPrintFatal(fmt.Sprintf("config error - Email.DefaultSuffix must have a value when Auth.Ldap.GroupSync is enabled"))
			}
		}

	} else {
		igor.Auth.Ldap.GroupSync.EnableGroupSync = false
	}

	if igor.Database.Adapter == "" {
		exitPrintFatal("config error - database.adapter required but not set")
	} else {
		if igor.Database.Adapter != "sqlite" {
			exitPrintFatal(fmt.Sprintf("database.adapter setting '%s' not recognized", igor.Database.Adapter))
		}
	}

	// Set database path
	if igor.Database.DbFolderPath != "" {
		if _, err := os.Stat(igor.Database.DbFolderPath); errors.Is(err, os.ErrNotExist) {
			createErr := os.MkdirAll(igor.Database.DbFolderPath, 0700)
			if createErr != nil {
				exitPrintFatal(fmt.Sprintf("config error - cannot create igor database folder %s - %v", igor.Database.DbFolderPath, createErr))
			}
		}
	} else {
		igor.Database.DbFolderPath = filepath.Join(igor.IgorHome, ".database")
		logger.Warn().Msgf("database.dbFolderPath not specified, using default (IGOR_HOME) : %v", igor.Database.DbFolderPath)
		createErr := os.MkdirAll(igor.Database.DbFolderPath, 0700)
		if createErr != nil {
			exitPrintFatal(fmt.Sprintf("config error - cannot create igor database folder %s - %v", igor.Database.DbFolderPath, createErr))
		}
	}

	if len(igor.Email.SmtpServer) == 0 {
		logger.Warn().Msg("email.smtpServer not specified -- igor will not send email")
		f := false
		igor.Email.ResNotifyOn = &f
	} else {
		logger.Info().Msg("email is enabled")
		if igor.Email.SmtpPort <= 0 {
			igor.Email.SmtpPort = 587
			logger.Info().Msgf("email.smtpPort port not specified, using default : %d", igor.Email.SmtpPort)
		}
	}

	// set VLAN settings
	if len(igor.Vlan.Network) > 0 {
		if igor.Vlan.Network != "arista" {
			logger.Warn().Msgf("vlan.network setting '%s' not recognized - no service is configured!", igor.Vlan.Network)
		} else {
			if igor.Vlan.NetworkUser == "" {
				igor.Vlan.NetworkUser = "igor"
				logger.Info().Msgf("vlan.networkUser not specified, using default : igor")
			}
			if igor.Vlan.NetworkURL == "" {
				exitPrintFatal("config error - vlan.networkURL cannot be blank when service is configured")
			}
			if igor.Vlan.RangeMin == 0 || igor.Vlan.RangeMax == 0 || igor.Vlan.RangeMin > igor.Vlan.RangeMax {
				exitPrintFatal(fmt.Sprintf("config error - vlan.rangeMin/Max is invalid [%d,%d]", igor.Vlan.RangeMin, igor.Vlan.RangeMax))
			}
		}
	} else {
		logger.Warn().Msg("no VLAN service is configured")
	}

	// email settings
	if len(igor.Email.SmtpServer) > 0 {

		if igor.Email.ResNotifyOn == nil {
			logger.Warn().Msg("email.resNotifyOn not specified, using default : true")
			t := true
			igor.Config.Email.ResNotifyOn = &t
		}
		if igor.Email.DefaultSuffix == "" {
			exitPrintFatal("config error - email.defaultSuffix cannot be blank when email is enabled")
		}

		var resNotify []string

		if !*igor.Config.Email.ResNotifyOn {
			logger.Warn().Msgf("reservation status emails are disabled - ignoring email.resNotifyTimes setting.")
		} else if igor.Config.Email.ResNotifyTimes == "" {
			logger.Warn().Msgf("email.resNotifyTimes not specified - using default : 3d,1d")
			resNotify = []string{"1d", "3d"}
		} else {
			resNotify = strings.Split(igor.Config.Email.ResNotifyTimes, ",")
		}

		for _, n := range resNotify {
			d, dErr := common.ParseDuration(n)
			if dErr != nil {
				exitPrintFatal(fmt.Sprintf("config error - email.resNotifyTimes %s is not a valid time value - %v", n, dErr))
			} else if d < time.Hour {
				exitPrintFatal(fmt.Sprintf("config error - email.resNotifyTimes %s is less than the minimum allowed value of 1 hour", n))
			}
			ResNotifyTimes = append(ResNotifyTimes, d)
		}

		// ensure ResNotifyTimes is in ascending order
		sort.Slice(ResNotifyTimes, func(i, j int) bool {
			return ResNotifyTimes[i] < ResNotifyTimes[j]
		})

		if len(ResNotifyTimes) > 0 {
			var temp []string
			for _, x := range ResNotifyTimes {
				temp = append([]string{common.FormatDuration(x, false)}, temp...)
			}
			logger.Info().Msgf("reservation notification times are: " + strings.Join(temp, ","))
		}
	}

	// scheduler settings
	if igor.Scheduler.MinReserveTime <= 0 {
		logger.Warn().Msgf("scheduler.minReserveTime not specified, using default : %d", DefaultMinReserveTime)
		igor.Scheduler.MinReserveTime = DefaultMinReserveTime
	} else if igor.Scheduler.MinReserveTime < LowestMinReserveTime {
		logger.Warn().Msgf("scheduler.minReserveTime is too small, using lowest acceptable value : %d", LowestMinReserveTime)
		igor.Scheduler.MinReserveTime = LowestMinReserveTime
	}

	if igor.Scheduler.DefaultReserveTime <= 0 {
		logger.Warn().Msgf("scheduler.defaultReserveTime not specified, using default: %d", DefaultReserveTime)
		igor.Scheduler.DefaultReserveTime = DefaultReserveTime
	}

	if igor.Scheduler.DefaultReserveTime < igor.Scheduler.MinReserveTime {
		exitPrintFatal(fmt.Sprintf("config error - scheduler.defaultReserveTime %v cannot be less than scheduler.minReserveTime %v",
			igor.Scheduler.DefaultReserveTime, igor.Scheduler.MinReserveTime))
	}

	if igor.Scheduler.MaxScheduleDays <= 0 {
		logger.Warn().Msgf("scheduler.maxScheduleDays not specified, using default : %d", MaxScheduleDays)
		igor.Scheduler.MaxScheduleDays = MaxScheduleDays
	} else if igor.Scheduler.MaxScheduleDays > MaxScheduleDays {
		logger.Warn().Msgf("scheduler.maxScheduleDays is too large, using max value instead : %d", MaxScheduleDays)
		igor.Scheduler.MaxScheduleDays = MaxScheduleDays
	}

	if igor.Scheduler.MaxReserveTime <= 0 {
		logger.Warn().Msgf("scheduler.maxReserveTime not specified, using default : %d", DefaultMaxReserveTime)
		igor.Scheduler.MaxReserveTime = DefaultMaxReserveTime
	} else if igor.Scheduler.MaxReserveTime > MaxReserveMinutes {
		logger.Warn().Msgf("scheduler.maxReserveTime is too large, using max value instead : %d", MaxReserveMinutes)
		igor.Scheduler.MaxReserveTime = int64(MaxReserveMinutes)
	}

	MaxScheduleMinutes = igor.Scheduler.MaxScheduleDays * 60 * 24

	mrtDays := float64(igor.Scheduler.MaxReserveTime) / (60.0 * 24.0)
	if float64(igor.Scheduler.MaxScheduleDays) < mrtDays {
		exitPrintFatal(fmt.Sprintf("config error - scheduler.maxReserveTime expressed as days (%d = %.3f) cannot be greater than maxScheduleDays (%v)",
			igor.Scheduler.MaxReserveTime, mrtDays, igor.Scheduler.MaxScheduleDays))
	}

	if igor.Scheduler.MaxReserveTime <= int64(MaxScheduleMinutes) {
		if igor.Scheduler.MaxReserveTime < igor.Scheduler.DefaultReserveTime {
			exitPrintFatal(fmt.Sprintf("config error - scheduler.maxReserveTime %v cannot be less than scheduler.defaultReserveTime %v",
				igor.Scheduler.MaxReserveTime, igor.Scheduler.DefaultReserveTime))
		}
	}

	if igor.Scheduler.ExtendWithin == 0 {
		logger.Warn().Msgf("scheduler.extendWithin not specified, using default : %d", DefaultExtendWithin)
		igor.Scheduler.MaxScheduleDays = DefaultExtendWithin
	} else if igor.Scheduler.ExtendWithin < 0 {
		logger.Warn().Msgf("scheduler.extendWithin -- reservation extend command is disabled!")
	}

	if igor.ExternalCmds.ConcurrencyLimit == 0 {
		logger.Info().Msgf("externalCmds.concurrencyLimit not specified, using default : 1")
		igor.ExternalCmds.ConcurrencyLimit = 1
	}

	logger.Warn().Msg("--- end: important notes and applying defaults/overrides")
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
					logger.Info().Msgf("%s = <nil>", finalName)
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
