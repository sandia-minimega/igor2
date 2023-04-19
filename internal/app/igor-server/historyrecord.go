// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"strings"
	"time"
)

const (
	HrCreated   = "created"
	HrInstalled = "installed"
	HrUpdated   = "updated"
	HrDeleted   = "deleted"
	HrFinished  = "finished"
)

type HistoryRecord struct {
	Base
	Hash        string `gorm:"notNull"`
	Status      string
	Name        string `gorm:"notNull"`
	Description string
	Owner       string
	Group       string
	Profile     string
	Distro      string
	Vlan        int
	Start       time.Time
	End         time.Time
	OrigEnd     time.Time
	ExtendCount int
	Hosts       string
}

func NewHistoryRecord(res *Reservation, status string) *HistoryRecord {

	if status == HrCreated || status == HrInstalled {
		result, _ := dbReadReservationsTx(map[string]interface{}{"ID": res.ID}, nil)
		res = &result[0]
	}

	// if the user deleted the reservation, record the end time as now
	end := res.End
	if status == HrDeleted {
		end = time.Now().Round(time.Second)
	}

	hr := &HistoryRecord{
		Hash:        res.Hash,
		Status:      status,
		Name:        res.Name,
		Description: res.Description,
		Owner:       res.Owner.Name,
		Group:       res.Group.Name,
		Profile:     res.Profile.Name,
		Distro:      res.Profile.Distro.Name,
		Vlan:        res.Vlan,
		Start:       res.Start,
		End:         end,
		OrigEnd:     res.OrigEnd,
		ExtendCount: res.ExtendCount,
		Hosts:       strings.Join(namesOfHosts(res.Hosts), ","),
	}

	return hr
}

func doHistoryRecord(res *Reservation, status string) error {
	hr := NewHistoryRecord(res, status)
	return dbCreateHistoryRecordTx(hr)
}
