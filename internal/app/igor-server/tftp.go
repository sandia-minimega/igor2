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

func generateBootFile(host *Host, r *Reservation) error {
	var content string
	image := r.Profile.Distro.DistroImage
	kernelPath := filepath.Join(igor.ImageStoreDir, image.ImageID, image.Kernel)
	initrdPath := filepath.Join(igor.ImageStoreDir, image.ImageID, image.Initrd)

	bootMode := host.BootMode
	osType := image.Breed

	masterPath := filepath.Join(igor.TFTPPath, igor.PXEBIOSDir, "igor", host.Name)
	pxePath := getPxePath(host)

	// Construct the auto-install part of the boot file based on OS type
	autoInstallFilePath := ""
	if image.LocalBoot {
		ksFile := r.Profile.Distro.Kickstart.Filename
		autoInstallFilePath = fmt.Sprintf("http://%s:%v/%s/%s", igor.Server.CbHost, igor.Server.CbPort, api.CbKS, ksFile)
	}

	switch bootMode {
	case "bios":
		// Generate content for BIOS
		defaultLabel := fmt.Sprintf("DEFAULT %s", r.Name)
		defaultOptions := ""
		biosLabel := fmt.Sprintf("LABEL %s", r.Name)
		kernel := fmt.Sprintf("\tKERNEL %v", kernelPath)
		appendStmt := fmt.Sprintf("\tAPPEND initrd=%v", initrdPath)
		autoInstallPart := ""
		if autoInstallFilePath != "" {
			switch osType {
			case "redhat":
				appendStmt = "IPAPPEND 2\n" + appendStmt
				autoInstallPart = fmt.Sprintf(" lang=  kssendmac text ksdevice=bootif ks=%s ", autoInstallFilePath)
			case "ubuntu", "debian", "freebsd", "generic", "nexenta", "suse", "unix", "vmware", "windows", "xen":
				autoInstallPart = fmt.Sprintf(" lang=  netcfg/choose_interface=%s text  auto-install/enable=true priority=critical hostname=%s url=%s domain=local.lan", host.Mac, host.Name, autoInstallFilePath)
			default:
				return fmt.Errorf("unknown OS type: %s", osType)
			}
		}
		content = fmt.Sprintf("%s\n%s\n%s\n%s\n%s %s\n", defaultLabel, defaultOptions, biosLabel, kernel, appendStmt, autoInstallPart)
	case "uefi":
		// Generate content for UEFI
		label := fmt.Sprintf("\"Reservation: %s netbooting %s on host %s\"", r.Name, r.Profile.Distro.Name, host.Name)
		autoInstallPart := ""
		if autoInstallFilePath != "" {
			switch osType {
			case "redhat":
				autoInstallPart = fmt.Sprintf(" lang=  inst.kssendmac inst.text inst.ksdevice=bootif inst.ks=%s", autoInstallFilePath)
			case "ubuntu", "debian", "freebsd", "generic", "nexenta", "suse", "unix", "vmware", "windows", "xen":
				autoInstallPart = fmt.Sprintf(" lang=  netcfg/choose_interface=%s text  auto-install/enable=true priority=critical url=%s", host.Mac, autoInstallFilePath)
			default:
				return fmt.Errorf("unknown OS type: %s", osType)
			}
		}
		content = fmt.Sprintf("set default=install-menu\nset timeout=6\n\nmenuentry %s --id install-menu {\n    linuxefi %s %s\n    initrdefi %s\n}\n", label, kernelPath, autoInstallPart, initrdPath)
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
	logger.Debug().Msgf("uninstalling reservation %v", r.Name)
	// Delete all the PXE files in the reservation
	for _, host := range r.Hosts {
		pxePath := getPxePath(&host)

		err := os.Remove(pxePath)
		if err != nil {
			// record the failure but no need to halt
			logger.Warn().Msgf("pxeconfig file for host %v encountered a problem during uninstall: %v", host.Name, err.Error())
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
		label := fmt.Sprint("LABEL local\n")
		labelOptions := "\tLOCALBOOT -1\n"
		content = defaultLabel +
			defaultOptions +
			label +
			labelOptions
	case "uefi":
		grubPath := ""
		switch r.Profile.Distro.DistroImage.Breed {
		case "redhat":
			grubPath = "/EFI/redhat/grubx64.efi"
		default:
			grubPath = "+1"
		}
		label := fmt.Sprintf("\"Reservation: %s booting %s locally on host %s\"", r.Name, r.Profile.Distro.Name, host.Name)
		content = fmt.Sprintf("set default=install-menu\nset timeout=6\n\nmenuentry %s  --id install-menu {\n    insmod part_gpt\n    insmod fat\n    search --no-floppy --set=root --file %s\n    chainloader %s\n}\n", label, grubPath, grubPath)
	default:
		return fmt.Errorf("unknown boot mode: %s", host.BootMode)
	}

	if err := writeFile(path, content); err != nil {
		return err
	}
	return nil
}
