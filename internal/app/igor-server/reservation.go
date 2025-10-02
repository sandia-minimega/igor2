// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"igor2/internal/pkg/common"
)

const PermReservations = "reservations"

// Reservation stores the information about a single reservation.
type Reservation struct {
	Base
	Name        string `gorm:"unique; notNull"`
	Description string
	OwnerID     int
	Owner       User
	GroupID     int
	Group       Group
	ProfileID   int
	Profile     Profile
	Vlan        int
	Start       time.Time
	End         time.Time
	OrigEnd     time.Time `gorm:"<-:create"`
	ResetEnd    time.Time
	// ExtendCount increments each time res is extended
	ExtendCount  int
	Hosts        []Host `gorm:"many2many:reservations_hosts;"`
	Installed    bool
	InstallError string
	CycleOnStart bool
	NextNotify   time.Duration
	// Hash is the unique ID used for history tracking
	Hash string `gorm:"<-:create; unique; notNull"`
	// Callback is the unique ID used for history tracking
	HistCallback func(res *Reservation, status string) error `gorm:"-"`
}

func filterReservationList(resList []Reservation, user *User) []common.ReservationData {

	var reportList []common.ReservationData

	refreshPowerChan <- struct{}{}

	for _, r := range resList {

		sort.Slice(r.Hosts, func(i, j int) bool {
			return r.Hosts[i].SequenceID < r.Hosts[j].SequenceID
		})

		hostNameList := namesOfHosts(r.Hosts)

		remaining := time.Until(r.End).Round(time.Hour) / time.Hour

		var groupName string
		if !strings.HasPrefix(r.Group.Name, GroupUserPrefix) {
			groupName = r.Group.Name
		}

		hostRange, _ := igor.ClusterRefs[0].UnsplitRange(hostNameList)

		resHostData := filterHostList(r.Hosts, nil, user)
		var resDownNodes = make([]string, 0, len(r.Hosts))
		var resPowerNaNodes = make([]string, 0, len(r.Hosts))
		var resUpNodes = make([]string, 0, len(r.Hosts))

		for _, h := range hostNameList {
			var isDownOrUnknown = false
			for _, d := range resHostData {
				if h == d.Name {
					if d.Powered == "false" {
						resDownNodes = append(resDownNodes, h)
						isDownOrUnknown = true
						break
					} else if d.Powered == "unknown" {
						resDownNodes = append(resPowerNaNodes, h)
						isDownOrUnknown = true
						break
					}
				}
			}

			if !isDownOrUnknown {
				resUpNodes = append(resUpNodes, h)
			}
		}

		hostsUp, _ := igor.ClusterRefs[0].UnsplitRange(resUpNodes)
		hostsDown, _ := igor.ClusterRefs[0].UnsplitRange(resDownNodes)
		hostsUnknown, _ := igor.ClusterRefs[0].UnsplitRange(resPowerNaNodes)

		resCopy := common.ReservationData{
			Name:         r.Name,
			Description:  r.Description,
			Owner:        r.Owner.Name,
			Group:        groupName,
			Start:        r.Start.Unix(),
			End:          r.End.Unix(),
			OrigEnd:      r.OrigEnd.Unix(),
			ExtendCount:  r.ExtendCount,
			Installed:    r.Installed,
			InstallError: r.InstallError,
			Distro:       r.Profile.Distro.Name,
			Profile:      r.Profile.Name,
			Hosts:        hostNameList,
			HostRange:    hostRange,
			HostsUp:      hostsUp,
			HostsDown:    hostsDown,
			HostsPowerNA: hostsUnknown,
			Vlan:         r.Vlan,
			RemainHours:  int(remaining),
		}

		reportList = append(reportList, resCopy)
	}

	sort.Slice(reportList, func(i, j int) bool {
		return reportList[i].Name < reportList[j].Name
	})

	return reportList
}

// AfterFind populates the history callback method after a reservation is fetched from the DB but
// before it is populated in the DB call result.
func (r *Reservation) AfterFind(_ *gorm.DB) (err error) {
	r.HistCallback = doHistoryRecord
	return nil
}

// DeepCopy clones an existing reservation struct with all refs to underlying structs intact.
func (r *Reservation) DeepCopy() *Reservation {

	clone := *r
	clone.Owner = r.Owner
	clone.Group = r.Group
	clone.Profile = r.Profile
	clone.Profile.Distro = r.Profile.Distro
	clone.Start = r.Start
	clone.End = r.End
	clone.OrigEnd = r.OrigEnd
	clone.ResetEnd = r.ResetEnd
	clone.HistCallback = r.HistCallback
	clone.Hosts = make([]Host, len(r.Hosts))
	copy(clone.Hosts, r.Hosts)

	return &clone
}

// IsActive returns true if the reservation is active at the given time
func (r *Reservation) IsActive(t time.Time) bool {
	return r.Start.Before(t) && r.End.After(t)
}

// Duration returns the duration interval of the reservation. It will
// calculate correctly based on the current end time as modified by
// any extend commands that were issued.
func (r *Reservation) Duration() time.Duration {
	return r.End.Sub(r.Start)
}

// Remaining returns how long the reservation has remaining at the given time
// if the reservation is active. If the reservation is not active, it returns
// how long the reservation will be active for.
func (r *Reservation) Remaining(t time.Time) time.Duration {
	if r.IsActive(t) {
		return r.End.Sub(t)
	}
	return r.Duration()
}

func (r *Reservation) getKernelArgs() string {
	// profile args should append behind distro args if both exist
	kArgs := ""
	if r.Profile.Distro.KernelArgs != "" {
		kArgs = kArgs + r.Profile.Distro.KernelArgs
	}
	if r.Profile.KernelArgs != "" {
		if kArgs != "" {
			kArgs = kArgs + " "
		}
		kArgs = kArgs + r.Profile.KernelArgs
	}
	return kArgs
}

func (r *Reservation) checkHostBootPolicy() error {
	var incompatible []string
	image := r.Profile.Distro.DistroImage
	if image.BiosBoot && image.UefiBoot {
		return nil
	}
	for _, host := range r.Hosts {
		switch host.BootMode {
		case "bios":
			if !image.BiosBoot {
				incompatible = append(incompatible, host.Name)
			}
		case "uefi":
			if !image.UefiBoot {
				incompatible = append(incompatible, host.Name)
			}
		}
	}
	if len(incompatible) > 0 {
		return fmt.Errorf("host(s) %v are not boot-compatible to the image used for distro '%s'", incompatible, r.Profile.Distro.Name)
	}
	return nil
}
