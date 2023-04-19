// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"igor2/internal/pkg/common"
	"sort"
)

const (
	PermProfiles = "profiles"
)

// Profile describes the configuration of a Distro to launch on a reserved Host.
//
// A Profile inherits the Group of its Distro unless that Group is "all" in which case
// the Profile creation and editing may accept a different Group argument. This means a Profile
// created with a Distro that belongs to the owner is only available to the owner.
type Profile struct {
	Base
	Name        string `gorm:"uniqueIndex:idx_pname_owner; notNull"`
	Description string
	OwnerID     int `gorm:"uniqueIndex:idx_pname_owner; notNull"`
	Owner       User
	DistroID    int
	Distro      Distro
	IsDefault   bool
	KernelArgs  string // Added to Distro kernel args if they exist.
}

// duplicate makes a deep copy of a profile, setting the given user as the new owner
func (p *Profile) duplicate(user *User) *Profile {
	return &Profile{
		Name:        p.Name,
		Owner:       *user,
		Description: p.Description,
		Distro:      p.Distro,
		KernelArgs:  p.KernelArgs,
	}
}

func filterProfileList(profiles []Profile) []common.ProfileData {
	var profileList []common.ProfileData
	for _, profile := range profiles {
		profileList = append(profileList, common.ProfileData{
			Name:        profile.Name,
			Description: profile.Description,
			Owner:       profile.Owner.Name,
			Distro:      profile.Distro.Name,
			KernelArgs:  profile.KernelArgs,
		})
	}

	sort.Slice(profileList, func(i, j int) bool {
		return profileList[i].Name < profileList[j].Name
	})

	return profileList
}
