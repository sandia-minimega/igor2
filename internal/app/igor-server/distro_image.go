// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"sort"

	"igor2/internal/pkg/common"
)

var DistroBreed = []string{
	"debian",
	"freebsd",
	"generic",
	"nexenta",
	"redhat",
	"suse",
	"ubuntu",
	"unix",
	"vmware",
	"windows",
	"xen",
}

// DistroImage represents boot file(s) associated to a distro.
type DistroImage struct {
	Base
	ImageID   string `gorm:"unique; notNull"`
	Type      string `gorm:"notNull"`
	Name      string `gorm:"unique; notNull"`
	Kernel    string
	Initrd    string
	Iso       string
	Breed     string
	LocalBoot bool
	Distros   []Distro
}

func filterDistroImagesList(distroImages []DistroImage) []common.DistroImageData {
	var distroImageList []common.DistroImageData

	for _, image := range distroImages {
		var distros []string
		for _, distro := range image.Distros {
			distros = append(distros, distro.Name)
		}
		local := "no"
		if image.LocalBoot {
			local = "yes"
		}
		distroImageList = append(distroImageList, common.DistroImageData{
			Name:      image.Name,
			ImageID:   image.ImageID,
			ImageType: image.Type,
			Kernel:    image.Kernel,
			Initrd:    image.Initrd,
			Iso:       image.Iso,
			Distros:   distros,
			Breed:     image.Breed,
			Local:     local,
		})
	}

	sort.Slice(distroImageList, func(i, j int) bool {
		return distroImageList[i].Name < distroImageList[j].Name
	})

	return distroImageList
}

func hasValidBreed(breed string) bool {
	for _, b := range DistroBreed {
		if b == breed {
			return true
		}
	}
	return false
}
