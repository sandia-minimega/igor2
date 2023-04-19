// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"sort"

	"igor2/internal/pkg/common"
)

// Kickstart (ks) represents an OS boot script which can be associated with a Distro.
//
// A ks script is not required. When a ks script is attached to a Distro, the tftp boot
// script created for the reservation adds the ks path to the Append line
//

// Kickstart represents an OS boot script file which contains everything the OS needs to install
type Kickstart struct {
	Base
	Name     string
	Filename string `gorm:"unique; notNull"`
	OwnerID  int
	Owner    User
}

func filterKickstartList(kickstarts []Kickstart) []common.KickstartData {
	var kickstartList []common.KickstartData

	for _, ks := range kickstarts {
		kickstartList = append(kickstartList, common.KickstartData{
			Name:     ks.Name,
			FileName: ks.Filename,
			Owner:    ks.Owner.Name,
		})
	}

	sort.Slice(kickstartList, func(i, j int) bool {
		return kickstartList[i].Name < kickstartList[j].Name
	})

	return kickstartList
}
