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
		Use: "register -k FILENAME.KERNEL -i FILENAME.INITRD " +
			" 		[-l --localBoot {true|false} -b --breed BREED]",
		Short: "Register image files " + adminOnly,
		Long: `
Registers bootable file(s) (ex. a kernel/initrd file pair) with igor. This
command is used when uploading is not enabled for users.

` + requiredFlags + `

  -k : name of the kernel file in the staged image directory
  -i : name of the initrd file in the staged image directory

` + optionalFlags + `

  -l : true if included, designate the image for local boot
  -b : breed of image. (Required if localBoot = true) Available values are:
  		debian, freebsd, generic, nexenta,
		redhat, suse, ubuntu, unix, vmware
		windows, xen

On success, the admin will receive a reference ID. It can be used by anyone to
create a distro:

'igor distro create NAME --image-ref REFID'

When registering an image using this action, the image files must have already
been placed in igor server's designated staged-images directory. See the
server.imageStageDir setting in the server config for directory path.

` + adminOnlyBanner + `
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			flagset := cmd.Flags()
			kstaged, _ := flagset.GetString("kernel")
			istaged, _ := flagset.GetString("initrd")
			localBoot, _ := flagset.GetBool("localBoot")
			breed, _ := flagset.GetString("breed")
			res, err := doRegisterImage(kstaged, istaged, breed, localBoot)
			if err != nil {
				return err
			}
			printRespSimple(res)
			return nil
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	var kstaged, istaged, breed string
	var localBoot bool
	cmdRegisterImage.Flags().StringVarP(&kstaged, "kernel", "k", "", "name of the already staged .kernel file")
	cmdRegisterImage.Flags().StringVarP(&istaged, "initrd", "i", "", "name of the already staged .initrd file")
	cmdRegisterImage.Flags().StringVarP(&breed, "breed", "b", "", "name of the OS breed")
	cmdRegisterImage.Flags().BoolVarP(&localBoot, "localBoot", "l", false, "true = image is intended for local install/boot")
	_ = cmdRegisterImage.MarkFlagRequired("kernel")
	_ = cmdRegisterImage.MarkFlagRequired("initrd")
	_ = registerFlagArgsFunc(cmdRegisterImage, "kernel", []string{"FILENAME.kernel"})
	_ = registerFlagArgsFunc(cmdRegisterImage, "initrd", []string{"FILENAME.initrd"})

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

func doRegisterImage(kstaged string, istaged string, breed string, localBoot bool) (*common.ResponseBodyBasic, error) {

	params := map[string]interface{}{}
	params["kernelStaged"] = kstaged
	params["initrdStaged"] = istaged
	if localBoot {
		params["localBoot"] = "true"
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
	tw.AppendHeader(table.Row{"NAME", "ID", "TYPE", "KERNEL", "INITRD", "ISO", "BREED", "LOCAL", "DISTROS"})

	for _, di := range imageList {
		tw.AppendRow([]interface{}{
			di.Name,
			di.ImageID,
			di.ImageType,
			di.Kernel,
			di.Initrd,
			di.Iso,
			di.Breed,
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
