// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"time"

	"igor2/internal/pkg/common"
)

// Igor holds globals
type Igor struct {
	Config
	ConfigPath      string
	ClusterConfPath string
	IResInstaller
	IGormDb
	IgorHome         string
	AuthSecondary    IAuth
	AuthToken        IAuth
	AuthBasic        IAuth
	AuthTokenKeypath string
	Started          time.Time
	TFTPPath         string
	PXEBIOSDir       string
	PXEUEFIDir       string
	ImageStoreDir    string
	KickstartDir     string
	ElevateMap       *common.PassiveTtlMap
	ClusterRefs      []common.Range
	IPowerStatus
}

func (i *Igor) getServerConfig() interface{} {

	igorConfig := struct {
		Config           Config         `json:"config"`
		ConfigPath       string         `json:"configPath"`
		ClusterConfPath  string         `json:"clusterConfPath"`
		IgorHome         string         `json:"igorHome"`
		AuthTokenKeypath string         `json:"authTokenKeypath"`
		Started          string         `json:"started"`
		ImageStoreDir    string         `json:"imageStoreDir"`
		TFTPPath         string         `json:"tftpPath"`
		ClusterRefs      []common.Range `json:"clusterRefs"`
	}{
		Config:           i.Config,
		ConfigPath:       i.ConfigPath,
		ClusterConfPath:  i.ClusterConfPath,
		IgorHome:         i.IgorHome,
		AuthTokenKeypath: i.AuthTokenKeypath,
		Started:          i.Started.Format(common.DateTimeServerFormat),
		ImageStoreDir:    i.ImageStoreDir,
		TFTPPath:         i.TFTPPath,
		ClusterRefs:      i.ClusterRefs,
	}

	return igorConfig
}

func (i *Igor) getServerSettings() interface{} {

	igorSettings := struct {
		LocalAuthEnabled       bool  `json:"localAuthEnabled"`
		CanUploadImages        bool  `json:"canUploadImages"`
		VlanEnabled            bool  `json:"vlanEnabled"`
		VlanRangeMin           int   `json:"vlanRangeMin"`
		VlanRangeMax           int   `json:"vlanRangeMax"`
		NodeReservationLimit   int   `json:"nodeReservationLimit"`
		MaxScheduleDays        int   `json:"maxScheduleDays"`
		MinReserveMinutes      int64 `json:"minReserveMinutes"`
		MaxReserveMinutes      int64 `json:"maxReserveMinutes"`
		DefaultReserveMinutes  int64 `json:"defaultReserveMinutes"`
		HostMaintenanceMinutes int   `json:"hostMaintenanceMinutes"`
	}{
		LocalAuthEnabled:       i.localAuthEnabled(),
		CanUploadImages:        i.Server.AllowImageUpload,
		VlanEnabled:            i.vlanEnabled(),
		VlanRangeMin:           i.Vlan.RangeMin,
		VlanRangeMax:           i.Vlan.RangeMax,
		NodeReservationLimit:   i.Scheduler.NodeReserveLimit,
		MaxScheduleDays:        i.Scheduler.MaxScheduleDays,
		MinReserveMinutes:      i.Scheduler.MinReserveTime,
		MaxReserveMinutes:      i.Scheduler.MaxReserveTime,
		DefaultReserveMinutes:  i.Scheduler.DefaultReserveTime,
		HostMaintenanceMinutes: igor.Maintenance.HostMaintenanceDuration,
	}

	return igorSettings
}

func (i *Igor) vlanEnabled() bool {
	if i.Vlan.Network == "" {
		return false
	}
	return true
}

func (i *Igor) localAuthEnabled() bool {
	if i.Auth.Scheme == "local" {
		return true
	}
	return false
}
