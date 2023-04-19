// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"igor2/internal/pkg/common"
	"sort"
)

const (
	PermGroups = "groups"
	// GroupAdmins is the immutable name of the group listing all users who are igor administrators. This is a protected
	// unique name.
	GroupAdmins = "admins"
	// GroupAll is the immutable name of the group listing all users recognized by igor. This is a protected
	// unique name.
	GroupAll = "all"
	// GroupUserPrefix is the prefix applied to all user-private groups forming the conjunction prefix+username. This
	// is a protected group identifier and no other kind of group should start with these characters.
	GroupUserPrefix = "u_"
	// GroupNoneAlias is a group name parameter passed from the client indicating the current group associated with a
	// resource should be removed. In cases (like reservation) where a group is required, this means reverting the group
	// to the owner's pug. This is a protected unique name.
	GroupNoneAlias = "none"
)

// Group contains a list of users as membership and
// may be assigned to different resources to define
// access to the assigned resources by its members
type Group struct {
	Base
	Name          string `gorm:"unique; notNull"`
	Description   string
	IsUserPrivate bool
	OwnerID       int
	Owner         User          // Group belongs-to Owner
	Members       []User        `gorm:"many2many:groups_users;"`
	Permissions   []Permission  // Group has-many permissions
	Reservations  []Reservation // Group has-many reservations
	Distros       []Distro      `gorm:"many2many:distros_groups;"`
	Policies      []HostPolicy  `gorm:"many2many:groups_policies;"`
}

func (g *Group) getGroupData() *common.GroupData {

	gd := &common.GroupData{
		Name:        g.Name,
		Description: g.Description,
		Owner:       g.Owner.Name,
	}

	if len(g.Members) > 0 {
		if g.Name == GroupAll || g.Name == GroupAdmins {
			for i, u := range g.Members {
				if u.Name == IgorAdmin {
					g.Members = append(g.Members[:i], g.Members[i+1:]...)
					break
				}
			}
		}
		gd.Members = userNamesOfUsers(g.Members)
		sort.Strings(gd.Members)
	}

	if len(g.Reservations) > 0 {
		gd.Reservations = resNamesOfResList(g.Reservations)
		sort.Strings(gd.Reservations)
	}

	if len(g.Distros) > 0 {
		gd.Distros = distroNamesOfDistros(g.Distros)
		sort.Strings(gd.Distros)
	}

	if len(g.Policies) > 0 {
		gd.Policies = hostPolicyNamesOfHostPolicies(g.Policies)
		sort.Strings(gd.Policies)
	}

	return gd
}
