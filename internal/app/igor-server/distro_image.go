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
	"nexenta",
	"redhat",
	"suse",
	"ubuntu",
	"unix",
	"vmware",
	"windows",
	"xen",
	"generic-linux",
}

// DistroImage represents boot file(s) associated to a distro.
type DistroImage struct {
	Base
	ImageID    string `gorm:"unique; notNull"`
	Type       string `gorm:"notNull"`
	Name       string `gorm:"unique; notNull"`
	KernelInfo string `gorm:"notNull;default:''"`
	InitrdInfo string `gorm:"notNull;default:''"`
	Kernel     string
	Initrd     string
	Breed      string
	LocalBoot  bool
	BiosBoot   bool `gorm:"notNull; default:false"`
	UefiBoot   bool `gorm:"notNull; default:false"`
	Distros    []Distro
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
		var boot []string
		if image.BiosBoot {
			boot = append(boot, "bios")
		}
		if image.UefiBoot {
			boot = append(boot, "uefi")
		}
		distroImageList = append(distroImageList, common.DistroImageData{
			Name:      image.Name,
			ImageID:   image.ImageID,
			ImageType: image.Type,
			Kernel:    image.KernelInfo,
			Initrd:    image.InitrdInfo,
			Distros:   distros,
			Breed:     image.Breed,
			Local:     local,
			Boot:      boot,
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
