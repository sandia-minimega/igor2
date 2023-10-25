// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package common

import "time"

// ShowData contains all information relevant to displaying the top-level
// show command for igor clients.
type ShowData struct {
	Cluster      ClusterData       `json:"cluster"`
	Hosts        []HostData        `json:"hosts"`
	Reservations []ReservationData `json:"reservations"`
	Profiles     []ProfileData     `json:"profiles"`
	Distros      []DistroData      `json:"distros"`
	UserGroups   []string          `json:"groups"`
}

type ReservationData struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Owner        string   `json:"owner"`
	Group        string   `json:"group"`
	Profile      string   `json:"profile"`
	Distro       string   `json:"distro"`
	Vlan         int      `json:"vlan"`
	Start        int64    `json:"start"`
	End          int64    `json:"end"`
	OrigEnd      int64    `json:"origEnd"`
	ExtendCount  int      `json:"extendCount"`
	Hosts        []string `json:"hosts"`
	HostRange    string   `json:"hostRange"`
	HostsUp      string   `json:"hostsUp"`
	HostsDown    string   `json:"hostsDown"`
	HostsPowerNA string   `json:"hostsPowerNA"`
	Installed    bool     `json:"installed"`
	InstallError string   `json:"installError"`
	RemainHours  int      `json:"remainHours"`
}

// DistroData contains the filtered contents of a Distro for user consumption
type DistroData struct {
	Name        string   `json:"name"`
	IsDefault   bool     `json:"isDefault"`
	Description string   `json:"description"`
	Owner       string   `json:"owner"`
	Groups      []string `json:"groups"`
	ImageType   string   `json:"image_type"`
	Kernel      string   `json:"kernel"`
	Initrd      string   `json:"initrd"`
	Iso         string   `json:"iso"`
	KernelArgs  string   `json:"kernelArgs"`
	Kickstart   string   `json:"kickstart"`
	IsPublic    bool     `json:"isPublic"`
}

// DistroImageData contains the filtered contents of a DistroImage for user consumption
type DistroImageData struct {
	Name      string   `json:"name"`
	ImageID   string   `json:"image_id"`
	ImageType string   `json:"image_type"`
	Kernel    string   `json:"kernel"`
	Initrd    string   `json:"initrd"`
	Iso       string   `json:"iso"`
	Distros   []string `json:"distros"`
	Breed     string   `json:"breed"`
	Local     string   `json:"local"`
}

// KickstartData contains the filtered contents of a Kickstart for user consumption
type KickstartData struct {
	Name     string `json:"name"`
	FileName string `json:"fileName"`
	Owner    string `json:"owner"`
}

// ProfileData creates a client-safe filtered result
type ProfileData struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Owner       string `json:"owner"`
	Distro      string `json:"distro"`
	KernelArgs  string `json:"kernelArgs"`
}

type HostData struct {
	Name         string   `json:"name"`
	SequenceID   int      `json:"sequenceID"`
	HostName     string   `json:"hostName"`
	Eth          string   `json:"eth"`
	IP           string   `json:"ip"`
	Mac          string   `json:"mac"`
	State        string   `json:"state"`
	Powered      string   `json:"powered"`
	Cluster      string   `json:"cluster"`
	HostPolicy   string   `json:"hostPolicy"`
	AccessGroups []string `json:"accessGroups"`
	Restricted   bool     `json:"restricted"`
	Reservations []string `json:"reservations"`
}

type ClusterData struct {
	Name          string `json:"name"`
	Prefix        string `json:"prefix"`
	DisplayHeight int    `json:"displayHeight"`
	DisplayWidth  int    `json:"displayWidth"`
	Motd          string `json:"motd"`
	MotdUrgent    bool   `json:"motdUrgent"`
}

// UserData is a struct that only contains fields relevant to responses sent
// back to a client.
type UserData struct {
	Name     string   `json:"name"`
	FullName string   `json:"fullName"`
	Email    string   `json:"email"`
	Groups   []string `json:"groups"`
	JoinDate int64    `json:"joinDate"`
}

// GroupData is textual information about a group that is most relevant to users.
type GroupData struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Owners       []string `json:"owners"`
	Members      []string `json:"members"`
	Distros      []string `json:"distros"`
	Policies     []string `json:"hostPolicies"`
	Reservations []string `json:"reservations"`
}

type HostPolicyData struct {
	Name         string          `json:"name"`
	Hosts        string          `json:"hosts"`
	MaxResTime   string          `json:"maxResTime"`
	AccessGroups []string        `json:"accessGroups"`
	NotAvailable []ScheduleBlock `json:"scheduleBlock"`
}

type StatsData struct {
	Option  string                  `json:"option"`
	Verbose bool                    `json:"verbose"`
	Start   time.Time               `json:"start"`
	End     time.Time               `json:"end"`
	Records []ResHistory            `json:"records"`
	ByUser  map[string]ResStatCount `json:"by_user"`
	Global  ResStatCount            `json:"global"`
}

// ScheduleBlock contains 2 variables:
//
// Start is a cron expression that describes a start date of unavailability.
// Duration is string value of the duration of unavailability.
//
// cron expression reference: https://en.wikipedia.org/wiki/Cron
type ScheduleBlock struct {
	Start    string `json:"start"`    // cron-format string describing when the unavailability period begins
	Duration string `json:"duration"` // value for duration of unavailability (ex "2d" = 2 days)
}

func (sb *ScheduleBlock) ToString() string {
	return sb.Start + " / " + sb.Duration
}

// ResHistory captures the filtered results from HistoryRecord.
type ResHistory struct {
	Order       int
	Hash        string
	Status      string
	Name        string
	Owner       string
	ResGroup    string
	Profile     string
	Distro      string
	Vlan        int
	Start       time.Time
	End         time.Time
	OrigEnd     time.Time
	ExtendCount int
	Hosts       string
}

// ResStatCount is used to count aspects of reservations either globally or by user.
type ResStatCount struct {
	UniqueUsers    int
	NodesUsedCount int
	ResCount       int
	CancelledEarly int
	NumExtensions  int
	TotalResTime   time.Duration
	Entries        []ResHistory
}
