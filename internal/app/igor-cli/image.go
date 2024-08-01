// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"igor2/internal/pkg/api"

	"github.com/jedib0t/go-pretty/v6/table"

	"igor2/internal/pkg/common"

	"github.com/spf13/cobra"
)

func newImageCmd() *cobra.Command {

	cmdImage := &cobra.Command{
		Use:   "image",
		Short: "Perform an image command " + adminOnly,
		Long: `
Image primary command. A sub-command must be invoked to do anything.

Images represent bootable files (currently limited to kernel/initrd pairs).
Based on igor's configuration, a user can upload bootable files while creating
a distro. If uploading is not enabled, an image must be registered as a 
separate step first, which requires an administrative role in handling the 
addition of new images to igor for private and public use.

Images intended to be installed and booted locally must include both the 
parameter localBoot = true and a breed.  

` + sBold("All image commands are admin-only.") + `
`,
	}

	cmdImage.AddCommand(newImageRegisterCmd())
	cmdImage.AddCommand(newImageShowCmd())
	cmdImage.AddCommand(newImageDelCmd())
	return cmdImage
}

func newImageRegisterCmd() *cobra.Command {

	cmdRegisterImage := &cobra.Command{
		Use: "register {-k FILENAME.KERNEL -i FILENAME.INITRD |\n" +
			" 		--kstaged FILENAME.KERNEL --istaged FILENAME.INITRD |\n" +
			" 		-d FOLDER/PATH} --boot {bios,uefi}\n" +
			"		[-l --localBoot {true|false} -b --breed BREED]\n",
		Short: "Register image files or distro",
		Long: `
Registers bootable file(s) (ex. a kernel/initrd file pair) with igor. This
command is used when uploading is not enabled for users.

` + requiredFlags + `

  -k : name/path to the kernel file. If including a distro for local boot,
		include the kernel file name if using a custom name. Otherwise,
		Igor will look for a default name based on OS breed.
  -i : name/path to the initrd file. If including a distro for local boot,
  		include the initrd file name if using a custom name. Otherwise,
  		Igor will look for a default name based on OS breed.
  -d : path to the folder containing the distribution if local install
  --boot: at least one or more comma-separated strings incidicating this 
  		image's compatible boot methods. Available values are: bios,uefi

` + optionalFlags + `

  -l : true if included, designate the image for local boot
  -b : breed of image. (Required if local boot flag -l is included)
  		Available values are:
  		debian, freebsd, generic, nexenta,
		redhat, suse, ubuntu, unix, vmware
		windows, xen

On success, the admin will receive a reference ID. It can be used by anyone to
create a distro:

'igor distro create NAME --image-ref REFID'

When registering an image using this action, the image files must have already
been placed in igor server's designated staged-images directory. See the
server.imageStageDir setting in the server config for directory path.
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			flagset := cmd.Flags()
			kstaged, _ := flagset.GetString("kstaged")
			istaged, _ := flagset.GetString("istaged")
			kpath, _ := flagset.GetString("kernel")
			ipath, _ := flagset.GetString("initrd")
			dpath, _ := flagset.GetString("distro")
			boot, _ := flagset.GetStringSlice("boot")
			localBoot, _ := flagset.GetBool("localBoot")
			breed, _ := flagset.GetString("breed")
			res, err := doRegisterImage(kstaged, istaged, kpath, ipath, dpath, boot, breed, localBoot)
			if err != nil {
				return err
			}
			printRespSimple(res)
			return nil
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	var kstaged, istaged, kpath, ipath, dpath, breed string
	var boot []string
	var localBoot bool
	cmdRegisterImage.Flags().StringVar(&kstaged, "kstaged", "", "name of the .kernel file already staged in the staged_images folder on the Igor server")
	cmdRegisterImage.Flags().StringVar(&istaged, "istaged", "", "name of the .initrd file already staged in the staged_images folder on the Igor server")
	cmdRegisterImage.Flags().StringVarP(&kpath, "kernel", "k", "", "name/path of the .kernel file to upload")
	cmdRegisterImage.Flags().StringVarP(&ipath, "initrd", "i", "", "name/path of the .initrd file to upload")
	cmdRegisterImage.Flags().StringVarP(&dpath, "distro", "d", "", "path to the distro folder to upload")
	cmdRegisterImage.Flags().StringSlice("boot", boot, "the compatible boot system to use the image with")
	cmdRegisterImage.Flags().StringVarP(&breed, "breed", "b", "", "name of the OS breed")
	cmdRegisterImage.Flags().BoolVarP(&localBoot, "localBoot", "l", false, "true = image is intended for local install/boot")
	// _ = cmdRegisterImage.MarkFlagRequired("kernel")
	// _ = cmdRegisterImage.MarkFlagRequired("initrd")
	// _ = registerFlagArgsFunc(cmdRegisterImage, "kernel", []string{"FILENAME.kernel"})
	// _ = registerFlagArgsFunc(cmdRegisterImage, "initrd", []string{"FILENAME.initrd"})

	return cmdRegisterImage
}

func newImageShowCmd() *cobra.Command {

	cmdShowImages := &cobra.Command{
		Use:   "show [-x]",
		Short: "Show image information " + adminOnly,
		Long: `
Shows all image information known to igor's image store. No parameters are
accepted. Full list of images are always returned.

` + optionalFlags + `

Use the -x flag to render screen output without pretty formatting.

` + adminOnlyBanner + `
`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			simplePrint = flagset.Changed("simple")
			printImages(doShowImages())
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	cmdShowImages.Flags().BoolVarP(&simplePrint, "simple", "x", false, "use simple text output")
	return cmdShowImages

}

func newImageDelCmd() *cobra.Command {

	return &cobra.Command{
		Use:   "del NAME",
		Short: "Delete an image " + adminOnly,
		Long: `
Deletes an igor image from the image store.

` + requiredArgs + `

  NAME : image name (ref-ID)

An image cannot be deleted if it is currently associated to an existing distro.
Any distros using the image must be deleted first. If a distro is deleted and
it is the last to be using an image (ex. kernel/initrd pair), then the image
will also be destroyed automatically.

` + adminOnlyBanner + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printRespSimple(doDeleteImage(args[0]))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}
}

func doRegisterImage(kstaged, istaged, kpath, ipath, dpath string, boot []string, breed string, localBoot bool) (*common.ResponseBodyBasic, error) {

	params := map[string]interface{}{}
	params["boot"] = boot
	if localBoot {
		if dpath != "" {
			tarPath := "output.tar.gz"
			// an OS repo is being uploaded
			err := compressFolderToTarGz(dpath, tarPath)
			if err != nil {
				return nil, fmt.Errorf("Unable to process given distribution path: %v", err.Error())
			}
			params["distro"] = openFile(tarPath)
			// might need to remove the tar file
			// os.Remove(tarPath)
		} else if kstaged != "" && istaged != "" {
			params["kstaged"] = kstaged
			params["istaged"] = istaged
		}
		if kpath != "" {
			params["kpath"] = kpath
		}
		if ipath != "" {
			params["ipath"] = ipath
		}
		params["localBoot"] = "true"
	} else {
		if kstaged != "" && istaged != "" {
			params["kstaged"] = kstaged
			params["istaged"] = istaged
		} else if kpath != "" && ipath != "" {
			params["kernelFile"] = openFile(kpath)
			params["initrdFile"] = openFile(ipath)
		} else {
			return nil, fmt.Errorf("paths to either uploadable kernel/initrd files or staged files names are required for image registration")
		}

	}
	if breed != "" {
		params["breed"] = breed
	}

	body := doSendMultiform(http.MethodPost, api.ImageRegister, params)
	return unmarshalBasicResponse(body), nil
}

func doShowImages() *common.ResponseBodyImages {
	var params string
	apiPath := api.Images + params
	body := doSend(http.MethodGet, apiPath, nil)
	rb := common.ResponseBodyImages{}
	err := json.Unmarshal(*body, &rb)
	checkUnmarshalErr(err)
	return &rb
}

func doDeleteImage(name string) *common.ResponseBodyBasic {
	apiPath := api.Images + "/" + name
	body := doSend(http.MethodDelete, apiPath, nil)
	return unmarshalBasicResponse(body)
}

func printImages(rb *common.ResponseBodyImages) {

	checkAndSetColorLevel(rb)

	imageList := rb.Data["distroImages"]
	if len(imageList) == 0 {
		printSimple("no images to show (yet) or no matches based on search criteria", cRespWarn)
	}

	sort.Slice(imageList, func(i, j int) bool {
		return strings.ToLower(imageList[i].Name) < strings.ToLower(imageList[j].Name)
	})

	tw := table.NewWriter()
	tw.AppendHeader(table.Row{"NAME", "ID", "TYPE", "KERNEL", "INITRD", "ISO", "BREED", "BOOT TYPE", "LOCAL", "DISTROS"})

	for _, di := range imageList {
		tw.AppendRow([]interface{}{
			di.Name,
			di.ImageID,
			di.ImageType,
			di.Kernel,
			di.Initrd,
			di.Iso,
			di.Breed,
			di.Boot,
			di.Local,
			strings.Join(di.Distros, "\n"),
		})
	}

	if simplePrint {
		tw.Style().Options.SeparateRows = false
		tw.Style().Options.SeparateColumns = true
		tw.Style().Options.DrawBorder = false
	} else {
		tw.SetStyle(igorTableStyle)
	}

	fmt.Printf("\n" + tw.Render() + "\n\n")

}
