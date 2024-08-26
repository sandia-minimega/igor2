// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"igor2/internal/pkg/api"

	"github.com/jedib0t/go-pretty/v6/table"

	"igor2/internal/pkg/common"

	"github.com/spf13/cobra"
)

func newDistroCmd() *cobra.Command {

	cmdDistro := &cobra.Command{
		Use:   "distro",
		Short: "Perform a distro command",
		Long: `
Distro primary command. A sub-command must be invoked to do anything.

Distros wrap bootable images like kernel initrd pairs, and can include addi-
tional information such as kernel args, a kickstart script, and a description.
Once created, a distro is associated with a profile, which is then used in a
reservation.
`,
	}

	cmdDistro.AddCommand(newDistroCreateCmd())
	cmdDistro.AddCommand(newDistroEditCmd())
	cmdDistro.AddCommand(newDistroShowCmd())
	cmdDistro.AddCommand(newDistroDelCmd())
	return cmdDistro
}

func newDistroCreateCmd() *cobra.Command {

	cmdCreateDistro := &cobra.Command{
		Use: "create NAME {--copy-distro DISTRO | --use-distro-image DISTRO |\n" +
			"              --kernel PATH/TO/KFILE.KERNEL --initrd PATH/TO/IFILE.INITRD |\n" +
			" 			   --kstaged FILENAME.KERNEL --istaged FILENAME.INITRD |\n" +
			" 			   -d FOLDER/PATH} | --image-ref IMAGEREF} \n" +
			"               [-g GRP1...] [--kickstart KICKSTART]\n" +
			"              [-k KARGS]  [-p PUBLIC] [--desc \"DESCRIPTION\"]",
		Short: "Create a distro",
		Long: `
Creates a new igor distro. A distro wraps an OS image (ex. KI-pair) and allows
optional attributes to be applied at boot time such as kernel arguments and/or
a kickstart script.

A new distro is private to its creator by default, but groups can be added to
allow others to use it. The creator of a distro can transfer ownership to
another igor user if desired.

` + requiredArgs + `

  NAME : distro name

` + requiredFlags + `

  The new distro's image must be specified in ONE of the following ways:

  --image-ref : The reference ID/name of a registered image. If an image was
      previously registered in a separate step, the refID that was returned can
      be used.
  --kernel/--initrd : The full path to the kernel and initrd files to be uploaded
      and registered to use in the new distro. This assumes the upload feature
      has been enabled in the configuration. Files must have extension names
      .kernel and .initrd respectively.
  --kstaged/--istaged : the file names of the kernel and initrd files
	  that have been placed in the igor_staged_images path by the admin.
  -d : path to the folder containing the distribution if local install
  --copy-distro : The name of an existing distro to base the new distro on.
      User must be the owner of the existing distro. New distro will inherit
      the description, image, kickstart script, and kernel args of the existing
      distro.
  --use-distro-image : The name of an existing distro to base the new distro's
      image on. User must be the owner of the existing distro. New distro will
      inherit only the image of the existing distro.

` + optionalFlags + `

Use the -k flag to add kernel arguments to this distro. These should be ` + sItalic("critical") + `
for the image to boot properly. Any additional or optional kernel args should
be associated to a profile created using the distro. Use 'igor profile create 
--help' for additional details.

Use the -g flag to set a list of groups that will have access to this distro.
Group members have the ability to use this distro for making profiles and
reservations.

Use the -p flag to specify that this distro is public, allowing anyone to use
it. The distro will be owned by igor-admin and can only be modified or deleted
by the admin team.

Use the --kickstart flag to assign a registered kickstart file to associate
with this distro. It is required when the image being used for the distro is
intended to be installed/boot locally. Otherwise it should not be used.

Use the --default flag (admin only) to designate this distro to overwrite an 
installed distro after its reservation ends. Only one distro at a time can be
marked as default. If a default distro already exists, it will revert to normal
status. A new default distro will change ownership to igor-admin and remove or
ignore any groups associated with it, making it privately owned by the admin
team.

` + descFlagText + `

` + sItalic("Due to similarities that can arise in naming a distro, use of the description\n"+
			"field is encouraged to make it easier to tell distros apart from each other.") + `
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flagset := cmd.Flags()
			kernel, _ := flagset.GetString("kernel")
			initrd, _ := flagset.GetString("initrd")
			kstaged, _ := flagset.GetString("kstaged")
			istaged, _ := flagset.GetString("istaged")
			dpath, _ := flagset.GetString("distro")
			copyDistro, _ := flagset.GetString("copy-distro")
			useDistroImage, _ := flagset.GetString("use-distro-image")
			imageRef, _ := flagset.GetString("image-ref")
			desc, _ := flagset.GetString("desc")
			groups, _ := flagset.GetStringSlice("groups")
			kargs, _ := flagset.GetString("kernel-args")
			public, _ := flagset.GetBool("public")
			isDefault, _ := flagset.GetBool("default")
			kickstart, _ := flagset.GetString("kickstart")
			res, err := doCreateDistro(args[0], kernel, initrd, kstaged, istaged, dpath, copyDistro, useDistroImage, imageRef, desc, groups, kargs, kickstart, public, isDefault)
			if err != nil {
				return err
			}
			printRespSimple(res)
			return nil
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	var kernel,
		initrd,
		kstaged,
		istaged,
		dpath,
		copyDistro,
		useDistroImage,
		imageRef,
		desc,
		kargs,
		kickstart string
	var groups []string

	cmdCreateDistro.Flags().StringVar(&kernel, "kernel", "", "full local path to a .kernel file")
	cmdCreateDistro.Flags().StringVar(&initrd, "initrd", "", "full local path to a .initrd file")
	cmdCreateDistro.Flags().StringVar(&kstaged, "kstaged", "", "name of the .kernel file already placed in the staged_images folder on the Igor server")
	cmdCreateDistro.Flags().StringVar(&istaged, "istaged", "", "name of the .initrd file already placed in the staged_images folder on the Igor server")
	cmdCreateDistro.Flags().StringVarP(&dpath, "distro", "d", "", "path to the distro folder to upload")
	// cmdCreateDistro.Flags().StringSlice("boot", boot, "the compatible boot system to use the image with ['bios','uefi']")
	cmdCreateDistro.Flags().StringVar(&copyDistro, "copy-distro", "", "name of an already existing distro to duplicate")
	cmdCreateDistro.Flags().StringVar(&useDistroImage, "use-distro-image", "", "name of an already existing distro to use image from")
	cmdCreateDistro.Flags().StringVar(&imageRef, "image-ref", "", "the image reference ID (provided by admin)")
	cmdCreateDistro.Flags().StringVar(&desc, "desc", "", "description of the distro")
	cmdCreateDistro.Flags().StringSliceVarP(&groups, "groups", "g", nil, "group(s) that can access the distro")
	cmdCreateDistro.Flags().StringVarP(&kargs, "kernel-args", "k", "", "string arguments to use when booting the image of this distro")
	cmdCreateDistro.Flags().StringVar(&kickstart, "kickstart", "", "the name of a registered kickstart file")
	cmdCreateDistro.Flags().BoolP("public", "p", false, "make this distro public (anyone can use, can't undo)")
	cmdCreateDistro.Flags().Bool("default", false, "make this distro default (used during post-reservation maintenance phase)")
	_ = cmdCreateDistro.MarkFlagFilename("kernel", "kernel")
	_ = cmdCreateDistro.MarkFlagFilename("initrd", "initrd")
	_ = registerFlagArgsFunc(cmdCreateDistro, "copy-distro", []string{"DIST"})
	_ = registerFlagArgsFunc(cmdCreateDistro, "use-distro-image", []string{"DIST"})
	_ = registerFlagArgsFunc(cmdCreateDistro, "image-ref", []string{"IMAGEREF"})
	_ = registerFlagArgsFunc(cmdCreateDistro, "desc", []string{"\"DESCRIPTION\""})
	_ = registerFlagArgsFunc(cmdCreateDistro, "groups", []string{"GRP1"})
	_ = registerFlagArgsFunc(cmdCreateDistro, "kernel-args", []string{"\"KARGS\""})
	_ = registerFlagArgsFunc(cmdCreateDistro, "copy-distro", []string{"USER1"})

	return cmdCreateDistro
}

func newDistroShowCmd() *cobra.Command {

	cmdShowDistros := &cobra.Command{
		Use: "show [-n NAME1,...] [-o OWNER1,...] [-g GRP1,...] [--image-ids ID1,...]\n" +
			"       [--kernels KERN1,...] [--initrds INIT1,...] [-x] [--default]",
		Short: "Show distro information",
		Long: `
Shows distro information, returning matches to specified parameters. If no
parameters are provided then all distros will be returned.

Output will provide the distro name, its owner, groups, associated image type,
names of bootable files, and whether the image is public.

` + optionalFlags + `

Use the -n, -o, -g, --image-ids, --kernel and --initrd flags to narrow results.
Multiple values for a given flag should be comma-delimited.

Use the -x flag to render screen output without pretty formatting.
`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			names, _ := flagset.GetStringSlice("names")
			owners, _ := flagset.GetStringSlice("owners")
			groups, _ := flagset.GetStringSlice("groups")
			imageIDs, _ := flagset.GetStringSlice("image-ids")
			kernels, _ := flagset.GetStringSlice("kernels")
			initrds, _ := flagset.GetStringSlice("initrds")
			byDefault, _ := flagset.GetBool("default")
			simplePrint = flagset.Changed("simple")
			printDistros(doShowDistros(names, owners, groups, imageIDs, kernels, initrds, byDefault))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNoArgs,
	}

	var names,
		owners,
		groups,
		imageIDs,
		kernels,
		initrds []string

	cmdShowDistros.Flags().StringSliceVarP(&names, "names", "n", nil, "search by distro name(s)")
	cmdShowDistros.Flags().StringSliceVarP(&owners, "owners", "o", nil, "search by owner name(s)")
	cmdShowDistros.Flags().StringSliceVarP(&groups, "groups", "g", nil, "search by group(s)")
	cmdShowDistros.Flags().StringSliceVar(&imageIDs, "image-ids", nil, "search by image ID(s)")
	cmdShowDistros.Flags().StringSliceVar(&kernels, "kernels", nil, "search by kernel file(s)")
	cmdShowDistros.Flags().StringSliceVar(&initrds, "initrds", nil, "search by initrd file(s)")
	cmdShowDistros.Flags().Bool("default", false, "show default distro")
	cmdShowDistros.Flags().BoolVarP(&simplePrint, "simple", "x", false, "use simple text output")
	_ = registerFlagArgsFunc(cmdShowDistros, "names", []string{"NAME1"})
	_ = registerFlagArgsFunc(cmdShowDistros, "owners", []string{"OWNER1"})
	_ = registerFlagArgsFunc(cmdShowDistros, "groups", []string{"GROUP1"})
	_ = registerFlagArgsFunc(cmdShowDistros, "image-ids", []string{"ID1"})
	_ = registerFlagArgsFunc(cmdShowDistros, "kernels", []string{"KERN1"})
	_ = registerFlagArgsFunc(cmdShowDistros, "initrds", []string{"INIT1"})

	return cmdShowDistros
}

func newDistroEditCmd() *cobra.Command {

	cmdEditDistro := &cobra.Command{
		Use: "edit NAME { [-n NEWNAME | -o OWNER | -a GRP1,... | -r GRP1,... |\n" +
			"       -k KARGS | --desc \"DESCRIPTION\" | -p ] }",
		Short: "Edit distro information",
		Long: `
Edits distro information. This can only be done by the distro owner or an admin.

` + requiredArgs + `

  NAME : distro name

` + optionalFlags + `

Use the -n flag to re-name the distro.

Use the -o flag to transfer ownership to another user. After this the original
owner can no longer edit or delete the distro.

Use the -k flag to replace the kernel arguments. Use caution when doing this as
distro kernel arguments are supposed to be critical to booting the underlying
OS image.

Use the -a and -r flags to add or remove groups from distro access respectively.
Separate multiple group names with commas.

Use the -p flag to change this distro to public, allowing anyone to use it. The
distro will be owned by igor-admin and can only be modified or deleted by the
admin team. This is a permanent change.

Use the --default flag (admin only) to designate this distro to overwrite an 
installed distro after its reservation ends. Only one distro at a time can be
marked as default. If a default distro already exists, it will revert to normal
status. A new default distro will change ownership to igor-admin and remove or
ignore any groups associated with it, making it privately owned by the admin
team.

Use the --default-remove flag (admin only) to remove the default designation
from the distro. Distro will no longer be marked as default to be used in the
maintenance phase when a reservation ends. No other changes are made to the
distro.

` + descFlagText + `
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			name, _ := flagset.GetString("name")
			desc, _ := flagset.GetString("desc")
			owner, _ := flagset.GetString("owner")
			add, _ := flagset.GetStringSlice("add")
			remove, _ := flagset.GetStringSlice("remove")
			kargs, _ := flagset.GetString("kernel-args")
			public, _ := flagset.GetBool("public")
			isDefault, _ := flagset.GetBool("default")
			defaultRemove, _ := flagset.GetBool("default-remove")
			printRespSimple(doEditDistro(args[0], name, owner, desc, add, remove, kargs, public, isDefault, defaultRemove))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}

	var name,
		owner,
		desc,
		kargs string
	var add,
		remove []string

	cmdEditDistro.Flags().StringVarP(&name, "name", "n", "", "update the name of the distro")
	cmdEditDistro.Flags().StringVarP(&owner, "owner", "o", "", "update the distro owner")
	cmdEditDistro.Flags().StringVar(&desc, "desc", "", "update the description of the distro")
	cmdEditDistro.Flags().StringSliceVarP(&add, "add", "a", nil, "group(s) to add to distro access")
	cmdEditDistro.Flags().StringSliceVarP(&remove, "remove", "r", nil, "group(s) to remove from distro access")
	cmdEditDistro.Flags().StringVarP(&kargs, "kernel-args", "k", "", "update the kernel arguments of the distro")
	cmdEditDistro.Flags().BoolP("public", "p", false, "make this distro public (anyone can use, can't undo)")
	cmdEditDistro.Flags().Bool("default", false, "make this distro default (used during post-reservation maintenance phase)")
	cmdEditDistro.Flags().Bool("default-remove", false, "remove the default designation from this distro")
	_ = registerFlagArgsFunc(cmdEditDistro, "name", []string{"NAME"})
	_ = registerFlagArgsFunc(cmdEditDistro, "owner", []string{"OWNER"})
	_ = registerFlagArgsFunc(cmdEditDistro, "desc", []string{"\"DESCRIPTION\""})
	_ = registerFlagArgsFunc(cmdEditDistro, "add", []string{"GRP1"})
	_ = registerFlagArgsFunc(cmdEditDistro, "remove", []string{"GRP1"})
	_ = registerFlagArgsFunc(cmdEditDistro, "kernel-args", []string{"\"KARGS\""})

	return cmdEditDistro
}

func newDistroDelCmd() *cobra.Command {

	return &cobra.Command{
		Use:   "del NAME",
		Short: "Delete a distro",
		Long: `
Deletes an igor distro. This can only be done by the distro owner or an admin.

` + requiredArgs + `

  NAME : distro name

` + notesOnUsage + `

A distro cannot be deleted if it is associated to an existing profile. Any 
profiles using the distro must be deleted first. If a distro is deleted and it
is the last to be using an image (ex. kernel/initrd pair), then the image will
also be destroyed automatically.
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printRespSimple(doDeleteDistro(args[0]))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     validateNameArg,
	}
}

func doCreateDistro(name, kfile, ifile, kstaged, istaged, dpath, eDistro, eKI, kiref, desc string, groups []string, kargs string, kickstart string, public, isDefault bool) (*common.ResponseBodyBasic, error) {

	params := map[string]interface{}{}
	params["name"] = name
	// params["boot"] = boot
	if kfile != "" && ifile != "" {
		params["kernelFile"] = openFile(kfile)
		params["initrdFile"] = openFile(ifile)
	} else if kstaged != "" && istaged != "" {
		params["kStaged"] = kstaged
		params["iStaged"] = istaged
	} else if eDistro != "" {
		params["copyDistro"] = eDistro
	} else if eKI != "" {
		params["useDistroImage"] = eKI
	} else if kiref != "" {
		params["imageRef"] = kiref
	} else {
		return nil, fmt.Errorf("error - one of the following is required: kernel and initrd OR kstaged and istaged OR copy-distro OR use-distro-image OR image-ref")
	}
	if dpath != "" {
		params["dPath"] = dpath
	}
	if desc != "" {
		params["description"] = desc
	}
	if kargs != "" {
		params["kernelArgs"] = kargs
	}
	if kickstart != "" {
		params["kickstart"] = kickstart
	}
	if public {
		params["public"] = "true"
	}
	if isDefault {
		params["default"] = "true"
	}
	if len(groups) > 0 {
		params["distroGroups"] = groups
	}

	if len(params) > 0 {
		body := doSendMultiform(http.MethodPost, api.Distros, params)
		return unmarshalBasicResponse(body), nil
	} else {
		return nil, fmt.Errorf("error loading params: %v", params)
	}
}

func doShowDistros(names []string, owners []string, groups []string, imageIDs []string, kernels []string, initrds []string, byDefault bool) *common.ResponseBodyDistros {

	var params string
	if len(names) > 0 {
		for _, n := range names {
			params += "name=" + n + "&"
		}
	}
	if len(owners) > 0 {
		for _, o := range owners {
			params += "owner=" + o + "&"
		}
	}
	if len(groups) > 0 {
		for _, o := range groups {
			params += "group=" + o + "&"
		}
	}
	if len(imageIDs) > 0 {
		for _, o := range imageIDs {
			params += "imageID=" + o + "&"
		}
	}
	if len(kernels) > 0 {
		for _, o := range kernels {
			params += "kernel=" + o + "&"
		}
	}
	if len(initrds) > 0 {
		for _, o := range initrds {
			params += "initrd=" + o + "&"
		}
	}
	if byDefault {
		params += "default=true&"
	}
	if params != "" {
		params = strings.TrimSuffix(params, "&")
		params = "?" + params
	}
	apiPath := api.Distros + params
	body := doSend(http.MethodGet, apiPath, nil)
	rb := common.ResponseBodyDistros{}
	err := json.Unmarshal(*body, &rb)
	checkUnmarshalErr(err)
	return &rb
}

func doEditDistro(name string, newName string, owner string, desc string, add []string, remove []string, kargs string, public, isDefault, defaultRemove bool) *common.ResponseBodyBasic {
	apiPath := api.Distros + "/" + name
	params := make(map[string]interface{})
	if newName != "" {
		params["name"] = newName
	}
	if owner != "" {
		params["owner"] = owner
	}
	if desc != "" {
		params["description"] = desc
	}
	if len(add) > 0 {
		params["addGroup"] = add
	}
	if len(remove) > 0 {
		params["removeGroup"] = remove
	}
	if kargs != "" {
		params["kernelArgs"] = kargs
	}
	if public {
		params["public"] = "true"
	}
	if isDefault {
		params["default"] = "true"
	}
	if defaultRemove {
		params["default_remove"] = "true"
	}
	body := doSendMultiform(http.MethodPatch, apiPath, params)
	return unmarshalBasicResponse(body)
}

func doDeleteDistro(name string) *common.ResponseBodyBasic {
	apiPath := api.Distros + "/" + name
	body := doSend(http.MethodDelete, apiPath, nil)
	return unmarshalBasicResponse(body)
}

func printDistros(rb *common.ResponseBodyDistros) {

	checkAndSetColorLevel(rb)

	distroList := rb.Data["distros"]
	if len(distroList) == 0 {
		printSimple("no distros to show (yet) or no matches based on search criteria", cRespWarn)
	}

	sort.Slice(distroList, func(i, j int) bool {
		return strings.ToLower(distroList[i].Name) < strings.ToLower(distroList[j].Name)
	})

	if simplePrint {

		var distroInfo string
		for _, d := range distroList {

			distroInfo = "DISTRO: " + d.Name + "\n"
			distroInfo += "  -DESCRIPTION: " + d.Description + "\n"
			distroInfo += "  -OWNER:       " + d.Owner + "\n"
			distroInfo += "  -PUBLIC:      " + strconv.FormatBool(d.IsPublic) + "\n"
			distroInfo += "  -GROUPS:      " + strings.Join(d.Groups, ",") + "\n"
			distroInfo += "  -IMAGE-TYPE:  " + d.ImageType + "\n"
			distroInfo += "  -KERNEL:      " + d.Kernel + "\n"
			distroInfo += "  -INITRD:      " + d.Initrd + "\n"
			distroInfo += "  -ISO:         " + d.Iso + "\n"
			distroInfo += "  -KERNEL-ARGS: " + d.KernelArgs + "\n"
			if d.Kickstart != "" {
				distroInfo += "  -KICKSTART:   " + d.Kickstart + "\n"
			}
			fmt.Print(distroInfo + "\n\n")
		}

	} else {

		tw := table.NewWriter()
		tw.AppendHeader(table.Row{"NAME", "DESCRIPTION", "OWNER", "PUBLIC?", "GROUPS", "IMAGE-TYPE", "KERNEL", "INITRD", "ISO", "KICKSTART", "KERNEL-ARGS"})
		tw.AppendSeparator()

		for _, d := range distroList {

			tw.AppendRow([]interface{}{
				d.Name,
				d.Description,
				d.Owner,
				d.IsPublic,
				strings.Join(d.Groups, "\n"),
				d.ImageType,
				d.Kernel,
				d.Initrd,
				d.Iso,
				d.Kickstart,
				d.KernelArgs,
			})
		}

		tw.SetColumnConfigs([]table.ColumnConfig{
			{
				Name:     "KERNEL-ARGS",
				WidthMax: 40,
			},
		})

		tw.SetStyle(igorTableStyle)
		fmt.Printf("\n" + tw.Render() + "\n\n")
	}

}
