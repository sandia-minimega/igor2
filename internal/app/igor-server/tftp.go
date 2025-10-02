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
	nodeErrors := ""
	for _, host := range r.Hosts {
		if err := generateBootFile(&host, r); err != nil {
			nodeErrors = nodeErrors + err.Error() + ", "
		}
	}
	if nodeErrors != "" {
		nodeErrors = strings.TrimRight(nodeErrors, ", ")
		return fmt.Errorf("%s: error installing distro, please notify admin or check error logs for more details", nodeErrors)
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

	kernel_args := ""
	if r.Profile.Distro.KernelArgs != "" {
		kernel_args = fmt.Sprintf("%s %s", kernel_args, r.Profile.Distro.KernelArgs)
	}
	if r.Profile.KernelArgs != "" {
		kernel_args = fmt.Sprintf("%s %s", kernel_args, r.Profile.KernelArgs)
	}

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
		if kernel_args != "" {
			appendStmt = fmt.Sprintf("%s %s", appendStmt, kernel_args)
		}
		autoInstallPart := ""
		if autoInstallFilePath != "" {
			switch osType {
			case "redhat":
				appendStmt = "IPAPPEND 2\n" + appendStmt
				autoInstallPart = fmt.Sprintf(" inst.lang=  inst.kssendmac text inst.ksdevice=bootif inst.ks=%s ", autoInstallFilePath)
			case "ubuntu", "debian", "freebsd", "generic", "nexenta", "suse", "unix", "vmware", "windows", "xen":
				autoInstallPart = fmt.Sprintf(" lang=  netcfg/choose_interface=%s text  auto-install/enable=true priority=critical hostname=%s url=%s domain=local.lan", host.Mac, host.Name, autoInstallFilePath)
			default:
				logger.Error().Msgf("%s: generate boot file - unknown OS type: %s", host.Name, osType)
				return fmt.Errorf("%s", host.Name)
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
				logger.Error().Msgf("%s: generate boot file - unknown OS type: %s", host.Name, osType)
				return fmt.Errorf("%s", host.Name)
			}
		}
		content = fmt.Sprintf("set default=install-menu\nset timeout=6\n\nmenuentry %s --id install-menu {\n    linuxefi %s %s %s\n    initrdefi %s\n}\n", label, kernelPath, autoInstallPart, kernel_args, initrdPath)
		masterPath = filepath.Join(igor.TFTPPath, igor.PXEUEFIDir, "igor", host.Name)
	default:
		logger.Error().Msgf("%s: generate boot file - unknown boot mode: %s", host.Name, bootMode)
		return fmt.Errorf("%s", host.Name)
	}

	// Write master to backup
	if err := writeFile(masterPath, content); err != nil {
		logger.Error().Msgf("%s: res install - error writing master config to backup path - %s", host.Name, err.Error())
		return fmt.Errorf("%s", host.Name)
	}

	// Write the content to the file
	if err := writeFile(pxePath, content); err != nil {
		logger.Error().Msgf("%s: res install - error writing master config to main path - %s", host.Name, err.Error())
		return fmt.Errorf("%s", host.Name)
	}
	return nil
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
		label := fmt.Sprintf("\"Reservation: %s booting %s locally on host %s\"", r.Name, r.Profile.Distro.Name, host.Name)
		content = fmt.Sprintf(`
set default=install-menu
set timeout=6

menuentry %s --id install-menu {
	if search -n -s -f /efi/boot/bootx64.efi ; then
		if [ -f (${root})/efi/opensuse/grub.efi ] ; then
			chainloader (${root})/efi/opensuse/grub.efi
		elif [ -f (${root})/efi/sle/grub.efi ] ; then
			chainloader (${root})/efi/sle/grub.efi
		elif [ -f (${root})/efi/sles/grub.efi ] ; then
			chainloader (${root})/efi/sles/grub.efi
		elif [ -f (${root})/efi/grub/grub.efi ] ; then
			chainloader (${root})/efi/grub/grub.efi
		elif [ -f (${root})/efi/ubuntu/grubx64.efi ] ; then
			chainloader (${root})/efi/ubuntu/grubx64.efi
		elif [ -f (${root})/efi/redhat/grubx64.efi ] ; then
			chainloader (${root})/efi/redhat/grubx64.efi
		elif [ -f (${root})/efi/rocky/grubx64.efi ] ; then
			chainloader (${root})/efi/rocky/grubx64.efi
		elif [ -f (${root})/efi/almalinux/grubx64.efi ] ; then
			chainloader (${root})/efi/almalinux/grubx64.efi
		elif [ -f (${root})/efi/centos/grubx64.efi ] ; then
			chainloader (${root})/efi/centos/grubx64.efi
		else
			chainloader (${root})/efi/boot/bootx64.efi
		fi
		boot
	else
		insmod part_gpt
		insmod fat
		search --no-floppy --set=root --file +1
		chainloader +1
	fi
}
			`, label)
	default:
		return fmt.Errorf("unknown boot mode: %s", host.BootMode)
	}

	if err := writeFile(path, content); err != nil {
		return err
	}
	return nil
}
