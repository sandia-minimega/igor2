// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"igor2/internal/pkg/api"
)

type TFTPInstaller struct {
}

func NewTFTPInstaller() IResInstaller {
	return &TFTPInstaller{}
}

func (b *TFTPInstaller) Install(r *Reservation) error {
	logger.Debug().Msgf("installing Reservation %v", r.Name)
	for _, host := range r.Hosts {
		if err := generateBootFile(&host, r); err != nil {
			return err
		}
	}

	return nil
}

// func writeBiosConfig(r *Reservation) error {
// 	// Manual file installation happens now
// 	// create appropriate pxe config file in igor.TFTPRoot/pxelinux.cfg/igor/
// 	image := r.Profile.Distro.DistroImage
// 	kernelPath := filepath.Join(igor.ImageStoreDir, image.ImageID, image.Kernel)
// 	initrdPath := filepath.Join(igor.ImageStoreDir, image.ImageID, image.Initrd)

// 	// create individual PXE boot configs i.e. igor.TFTPRoot+/pxelinux.cfg/00:00:00:00:00:00 by copying config created above
// 	logger.Debug().Msgf("cycling through hosts %v", r.Hosts)
// 	for _, host := range r.Hosts {
// 		// Prepare pxe.cfg content based on net/local and OS breed
// 		defaultLabel := fmt.Sprintf("default %s\n\n", r.Name)
// 		defaultOptions := ""
// 		label := fmt.Sprintf("label %s\n", r.Name)
// 		kernel := fmt.Sprintf("\tkernel %v\n", kernelPath)
// 		labelOptions := ""
// 		appendStmt := fmt.Sprintf("\tappend initrd=%v", initrdPath)
// 		if image.LocalBoot {
// 			ksFile := r.Profile.Distro.Kickstart.Filename
// 			ksServer := fmt.Sprintf("http://%s:%v/%s/%s", igor.Server.CbHost, igor.Server.CbPort, api.CbKS, ksFile)
// 			defaultOptions = fmt.Sprintf("%sprompt 0\ntimeout 1\n", defaultOptions)
// 			labelOptions = fmt.Sprintf("%s\tipappend 2\n", labelOptions)
// 			switch image.Breed {
// 			case "redhat":
// 				appendStmt = fmt.Sprintf("%s ksdevice=bootif lang=  kssendmac text  ks=%s", appendStmt, ksServer)
// 			case "ubuntu", "debian", "freebsd", "generic", "nexenta", "suse", "unix", "vmware", "windows", "xen":
// 				// Assume same setup as Ubuntu until we can test on these distributions
// 				appendStmt = fmt.Sprintf("%s lang=  netcfg/choose_interface=auto text  auto-install/enable=true priority=critical url=%s hostname=%s domain=local.lan suite=bionic", appendStmt, ksServer, host.HostName)
// 			default:
// 				return fmt.Errorf("distro breed not supported for tftp image service")
// 			}
// 		}
// 		appendStmt = fmt.Sprintf("%s %v\n", appendStmt, r.getKernelArgs())
// 		content := defaultLabel +
// 			defaultOptions +
// 			label +
// 			kernel +
// 			labelOptions +
// 			appendStmt

// 		// create file path for master (reference) file
// 		masterPath := filepath.Join(igor.TFTPPath, igor.PXEDir, "igor", host.Name)
// 		if err := writeFile(masterPath, content); err != nil {
// 			return err
// 		}

// 		// Create file path to copy master file to pxe.cfg
// 		pxePath := getPxeFilePath(&host)
// 		logger.Debug().Msgf("saving PXE file to path %v", pxePath)
// 		if err := writeFile(pxePath, content); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

func generateBootFile(host *Host, r *Reservation) error {
	var content string
	image := r.Profile.Distro.DistroImage
	kernelPath := filepath.Join(igor.ImageStoreDir, image.ImageID, image.Kernel)
	initrdPath := filepath.Join(igor.ImageStoreDir, image.ImageID, image.Initrd)

	bootMode := host.BootMode
	osType := image.Breed

	masterPath := filepath.Join(igor.TFTPPath, igor.PXEBIOSDir, "igor", host.Name)
	pxePath := getPxePath(host)

	// Construct the autoinstall part of the boot file based on OS type
	autoInstallPart := ""
	if image.LocalBoot {
		ksFile := r.Profile.Distro.Kickstart.Filename
		autoInstallFilePath := fmt.Sprintf("http://%s:%v/%s/%s", igor.Server.CbHost, igor.Server.CbPort, api.CbKS, ksFile)
		switch osType {
		case "redhat":
			autoInstallPart = fmt.Sprintf("ks=%s ksdevice=bootif", autoInstallFilePath)
		case "ubuntu", "debian", "freebsd", "generic", "nexenta", "suse", "unix", "vmware", "windows", "xen":
			autoInstallPart = fmt.Sprintf("auto url=%s", autoInstallFilePath)
		default:
			return fmt.Errorf("unknown OS type: %s", osType)
		}
	}

	switch bootMode {
	case "bios":
		// Generate content for BIOS
		defaultLabel := fmt.Sprintf("DEFAULT %s", r.Name)
		defaultOptions := ""
		label := fmt.Sprintf("LABEL Reservation: %s netbooting %s on host %s\n", r.Name, r.Profile.Distro.Name, host.Name)
		kernel := fmt.Sprintf("\tKERNEL %v", kernelPath)
		appendStmt := fmt.Sprintf("\tAPPEND initrd=%v", initrdPath)
		if autoInstallPart != "" {
			defaultOptions = fmt.Sprintf("%sprompt 0\ntimeout 1", defaultOptions)
			switch osType {
			case "redhat":
				appendStmt = "IPAPPEND 2\n" + appendStmt
				autoInstallPart = " lang=  kssendmac text" + autoInstallPart
			case "ubuntu", "debian", "freebsd", "generic", "nexenta", "suse", "unix", "vmware", "windows", "xen":
				autoInstallPart = fmt.Sprintf(" lang=  netcfg/choose_interface=auto text  auto-install/enable=true priority=critical hostname=%s %s domain=local.lan suite=bionic", host.Name, autoInstallPart)
			default:
				return fmt.Errorf("unknown OS type: %s", osType)
			}
		}
		content = fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s %s\n", defaultLabel, defaultOptions, label, kernel, appendStmt, autoInstallPart)
	case "uefi":
		// Generate content for UEFI
		content = fmt.Sprintf("menuentry 'reservation %s netbooting distro %s on host %s' {\n    linuxefi %s %s\n    initrdefi %s\n}\n", r.Name, r.Profile.Distro.Name, host.Name, kernelPath, autoInstallPart, initrdPath)
		masterPath = filepath.Join(igor.TFTPPath, igor.PXEUEFIDir, "igor", host.Name)
	default:
		return fmt.Errorf("unknown boot mode: %s", bootMode)
	}

	// Write master to backup
	if err := writeFile(masterPath, content); err != nil {
		return err
	}

	// Write the content to the file
	return writeFile(pxePath, content)
}

func (b *TFTPInstaller) Uninstall(r *Reservation) error {
	logger.Debug().Msgf("uninstalling Reservation %v", r.Name)
	// Delete all the PXE files in the reservation
	for _, host := range r.Hosts {
		pxePath := getPxePath(&host)

		err := os.Remove(pxePath)
		if err != nil {
			// record the failure but no need to halt
			logger.Error().Msgf("failed to uninstall pxeconfig file for host %v: %v", host.Name, err.Error())
		}
	}
	return nil
}

func macToPxeString(macAddr string) string {
	return strings.ToLower(strings.ReplaceAll(macAddr, ":", "-"))
}

func getPxePath(host *Host) string {
	macString := "01:" + host.Mac
	switch host.BootMode {
	case "bios":
		return filepath.Join(igor.TFTPPath, igor.PXEBIOSDir, macToPxeString(macString))
	case "uefi":
		return filepath.Join(igor.TFTPPath, igor.PXEUEFIDir, "grub.cfg-"+macToPxeString(macString))
	default:
		return ""
	}
}

func writeFile(path string, content string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create %v -- %v", path, err)
	}
	defer file.Close()
	file.WriteString(content)
	return nil
}

func setLocalConfig(host *Host, r *Reservation) error {
	path := getPxePath(host)
	content := ""
	switch host.BootMode {
	case "bios":
		defaultLabel := "DEFAULT local\n"
		defaultOptions := "PROMPT 0\nTIMEOUT 0\nTOTALTIMEOUT 0\nONTIMEOUT local\n"
		label := fmt.Sprintf("LABEL Reservation: %s netbooting %s on host %s\n", r.Name, r.Profile.Distro.Name, host.Name)
		labelOptions := "\tLOCALBOOT -1\n"
		content = defaultLabel +
			defaultOptions +
			label +
			labelOptions
	case "uefi":
		content = fmt.Sprintf("menuentry 'Reservation: %s booting %s locally on host %s' {\n    insmod chain\n    set root=(hd0,1)  # Assuming the OS is installed on the first partition of the first hard drive\n   chainloader +1", r.Name, r.Profile.Distro.Name, host.Name)
	default:
		return fmt.Errorf("unknown boot mode: %s", host.BootMode)
	}

	if err := writeFile(path, content); err != nil {
		return err
	}
	return nil
}
