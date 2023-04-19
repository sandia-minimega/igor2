// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"igor2/internal/pkg/common"

	"github.com/robfig/cron/v3"
)

const DefaultPolicyName = "default" // system default policy name with 24/7 availability and igor.MaxReserveTime

// HostPolicy provides an admin-only resource for setting host access in one place. Every node gets a single policy,
// and that policy defines when the node is available for use and by whom. A single policy can be applied to multiple
// nodes. If no other policy is applied to the node at the time it is created/registered, the default host policy is used.
//
// A node can be updated to use a different policy by an admin.
//
// Default policy:
//
//	NotAvailable = [] (no restrictions)
//	MaxResTime = (value set in igor config)
//	AccessGroups = [ALL]
//
// Assigning a policy to a node by default does not affect (current or future) reservations already created.
type HostPolicy struct {
	Base
	Name         string             `gorm:"unique; notNull"` // policy identifier
	Hosts        []Host             // the hosts this policy is assigned to
	MaxResTime   time.Duration      // default is config file value
	AccessGroups []Group            `gorm:"many2many:groups_policies;"`        // Only the listed Group(s) may reserve a node assigned to this policy. Defaults to GroupAll.
	NotAvailable ScheduleBlockArray `gorm:"column:notavailable;type:longtext"` // Can be empty, meaning nodes attached to this policy would not have any unavailability periods.
}

type ScheduleBlockArray []common.ScheduleBlock

// Scan - Override function for embedded struct to DB
func (sla *ScheduleBlockArray) Scan(src interface{}) error {
	return json.Unmarshal([]byte(src.(string)), &sla)
}

// Value - Override function for embedded struct to DB
func (sla ScheduleBlockArray) Value() (driver.Value, error) {
	val, err := json.Marshal(sla)
	return string(val), err
}

// removeSBInstance removes the given ScheduleBlock from the given ScheduleBlockArray
func (h *HostPolicy) removeSBInstance(sb common.ScheduleBlock) ScheduleBlockArray {
	newSBA := ScheduleBlockArray{}
	for _, sbi := range h.NotAvailable {
		if sbi.Start != sb.Start && sbi.Duration != sb.Duration {
			newSBA = append(newSBA, sbi)
		}
	}
	return newSBA
}

// parseSBInstance takes the string cron expression and returns a schedule object
func parseSBInstance(sb string) (cron.Schedule, error) {
	p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	mySB, err := p.Parse(sb)
	return mySB, err
}

func filterHostPoliciesList(hostPolicies []HostPolicy) []common.HostPolicyData {

	if len(igor.ClusterRefs) == 0 {
		return nil
	}

	var result []common.HostPolicyData
	for _, hp := range hostPolicies {
		hosts := namesOfHosts(hp.Hosts)

		hostRange, _ := igor.ClusterRefs[0].UnsplitRange(hosts)

		var groups []string
		for _, group := range hp.AccessGroups {
			groups = append(groups, group.Name)
		}
		result = append(result, common.HostPolicyData{
			Name:         hp.Name,
			Hosts:        hostRange,
			MaxResTime:   hp.MaxResTime.String(),
			AccessGroups: groups,
			NotAvailable: hp.NotAvailable,
		})
	}
	return result
}
