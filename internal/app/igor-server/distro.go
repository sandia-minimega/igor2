// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"sort"
	"strings"
	"time"

	"igor2/internal/pkg/common"

	"gorm.io/gorm"
)

// Distro represents an OS in file form whether as a pair of kernel and initrd files.
// It also contains the hashes of the files and a
// protected flag in order to determine if a new Distro entity can point to existing
// files on the system rather than uploading identical copies.
//
// A Distro is required for creating a Profile. A Profile will inherit the Distro Group
// if it has one. A Distro that belongs to the "all" group is publicly available for
// anyone to use.
//
// A Distro cannot be deleted if a Profile exists that uses it. The Profile must be deleted first.
//

const (
	PermDistros = "distros"
	// DistroKI indicates the image represents a netboot-only KI pair
	DistroKI = "ki"
	// DistroDistribution indicates the image represents an installable linux/unix distro
	DistroDistribution = "distribution"
	// MaxMemory determines amount of memory to use when parsing multipart form (32MB)
	MaxMemory = 32 << 20
)

// Distro represents an OS in file form
type Distro struct {
	Base
	Name          string `gorm:"unique; notNull"`
	IsDefault     bool
	Description   string
	OwnerID       int
	Owner         User
	Groups        []Group `gorm:"many2many:distros_groups;"`
	DistroImageID int
	DistroImage   DistroImage
	KickstartID   int
	Kickstart     Kickstart
	// Distro kernel args are optional but should only be specified if they are critical for the Distro OS to boot
	// correctly. Otherwise they should be specified in a Profile. Profile kernel args will be appended to Distro kernel args.
	KernelArgs string
}

// isPublic returns true if the distro's group contains the all group
func (d *Distro) isPublic() bool {
	for _, g := range d.Groups {
		if g.Name == GroupAll {
			return true
		}
	}
	return false
}

// returns true if the given distro is currently referenced in any existing profiles
func (d *Distro) isLinkedToProfiles(tx *gorm.DB) (bool, []string, error) {
	profiles, err := dbReadProfiles(map[string]interface{}{"distro_id": d.ID}, tx)
	if err != nil {
		return false, nil, err
	}
	if len(profiles) == 0 {
		return false, nil, nil
	}
	profileNames := profileNamesOfProfiles(profiles)
	return true, profileNames, nil
}

// returns the names of all active reservations where this distro is being used
func (d *Distro) hasActiveReservations() (results []string) {
	reservations, err := dbReadReservationsTx(map[string]interface{}{}, map[string]time.Time{})
	if err != nil {
		return results
	}
	for _, r := range reservations {
		rdName := r.Profile.Distro.Name
		if rdName == d.Name {
			results = append(results, r.Name)
		}
	}
	return results
}

// filters a list of distros to user-consumable objects
// (removes data users should not have access to)
func filterDistroList(distroInfo []Distro) []common.DistroData {
	var distroList []common.DistroData

	for _, distro := range distroInfo {
		var groups []string
		var isPublic bool
		groupNames := groupNamesOfGroups(distro.Groups)
		for _, gn := range groupNames {
			if !(strings.HasPrefix(gn, GroupUserPrefix) || gn == GroupAll) {
				groups = append(groups, gn)
			}
			if gn == GroupAll {
				isPublic = true
			}
		}
		distroList = append(distroList, common.DistroData{
			Name:        distro.Name,
			IsDefault:   distro.IsDefault,
			Description: distro.Description,
			Owner:       distro.Owner.Name,
			Groups:      groups,
			ImageType:   distro.DistroImage.Type,
			Kernel:      distro.DistroImage.Kernel,
			Initrd:      distro.DistroImage.Initrd,
			KernelArgs:  distro.KernelArgs,
			Kickstart:   distro.Kickstart.Name,
			IsPublic:    isPublic,
		})
	}

	sort.Slice(distroList, func(i, j int) bool {
		return distroList[i].Name < distroList[j].Name
	})

	return distroList
}
