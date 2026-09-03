# Distros

In Igor 1.x, OS images in the form of kernel/initrd (KI) pairs were transferred to a reservation's node(s) to boot. Igor 2.x still uses KI pairs but now wrapped in a **_distro_**. A distro includes the KI pair and other attributes that tell Igor how to install to a reservation's node(s). If the image is intended for net-boot, you can add kernel arguments to the distro to enhance or supplement the boot process. If the image is intended for local (disk) boot, you can reference a kickstart (or seed) script that has been registered with Igor to allow for automated installation. You can additionally specify groups to the distro that will restrict availability of the distro only to members of the assigned groups.

In order to use a KI pair you've created, you must first register the pair and create a new distro so you can use it in a reservation. New distros are created private by default. Only the owner of the new distro can use it, though you can set a distro for public use.

## Viewing Distros

To see what distros are available to you, use:

```shell
$ igor distro show
```

Any distros that are public (available to anyone), private (owned by you), or shared (groups assigned to them of which you're a member), will be included in the list and can be used to create a reservation.

New in v2.2 The kernel and initrd columns for distro information will now display the kernel and OS versions respectively, if Igor can determine this information from internal files in the KI pair. OS version may not be available immediately upon distro creation because it will take time to scan the compressed OS file. If the information cannot be found, these fields will default to "unknown".

## Creating a Distro

You can create a new distro in different ways. The usage syntax is:

`igor distro create NAME {--copy-distro DISTRO | --use-distro-image DISTRO | --image-ref IMAGEREF | `
`--kernel PATH/TO/KFILE.KERNEL --initrd PATH/TO/IFILE.INITRD}`

Distro names must be globally unique.

There are multiple options for where the KI pair (image) can come from:

  1. `--copy-distro DISTRO `: References the name of an existing distro. If the user owns the existing distro, the new distro inherits the kernel file, initrd file, kernel arguments and description from existing distro, if they are populated. A User can still modify these fields if included in parameters.
  2. `--use-distro-image DISTRO `: References the name of an existing distro. If the user owns the existing distro, then only the KI pair will be used to create the new distro. It will not inherit any other fields.
  3. `--image-ref IMAGEREF` : This is the reference ID a user receives from an admin after registering a KI pair in a separate step. Igor uses the ref ID to identify the registered KI pair and associate to the new distro. This creation method is used when an admin registers OS image files for the user.
  4. `--kernel PATH/TO/KFILE.KERNEL --initrd PATH/TO/IFILE.INITRD `: If enabled, Igor allows a user to create a new distro by uploading a KI pair directly from the local file system where the CLI is being used. Specify the path to each file required.

The optional `--desc` flag is used in the examples below to highlight differences between each creation method.

```shell
$ igor distro create MyUbu --copy-distro TheirUbu --desc "Cloned their distro"

$ igor distro create AllMine --use-distro-image TheirUbu --desc "Only used their KI pair"

$ igor distro create MyApprovedOS --image-ref ki5da2f0d2 --desc "The admin uploaded this for me"

$ igor distro create MyTinyOS --kernel ./tinyOS.kernel --initrd ./tinyOS.initrd --desc "I uploaded this TinyOS"
```

In order for option #4 to work, you will need to verify if uploading KI pairs is allowed on your Igor cluster, use:

```shell
$ igor settings
```

and check for the entry **canUploadImages.**  If _true_ , the including paths to the KI pair will work. Otherwise contact your cluster admin team to find out the process for getting your OS image registered with Igor. Once it is complete you can use the`--image-ref` flag along with the reference ID supplied by the admin to create your distro.

### Net-boot vs. Local-boot

Local-boot images require the assistance of a cluster admin team member to be set up correctly. Contact your admin for instructions on where to send your image files.

## Optional Creation Parameters

### Adding Groups

You can share a distro you own with groups you belong to using the **`-g | --group`**  flag. Any group member can then use it to make a reservation.

`-g myGroup1,myGroup2`

### Creating a Distro for Public Use

You can make a distro for public use using the `**-p | --public**  `flag. When you include this flag, Igor will set its ownership to the admin team and thereafter any user can make a reservation with it. You will not be able to modify a public distro once created. Public distros are a great resource for users to have one or more standardized OS images for general purpose use.

```shell
$ igor distro create TinyOS --kernel ./tinyOS.kernel --initrd ./tinyOS.initrd --description "Standard TinyOS" --public
```

### Adding Kernel Arguments

You can add kernel arguments to the distro using the `-k`  flag. These will be used in the boot process for the associated OS image.

`-k "--append=KEY=VALUE, ..."`

As a best practice, you should include only kernel arguments that _must_  be run with this image every time it's used by anyone. Optional or test kernel arguments should instead be used in a profile (see the Profile section below).

### Adding a Description

As shown in the first example, you can add a description to the distro using `--desc`. This may be helpful to distinguish between distros, especially if they have similar names.

`--desc "Ubuntu18 for Genesis Project dev"`
`--desc "Ubuntu18 with boot args for Genesis testing"`

## Modifying a Distro

You can modify an existing distro to change its name, add or remove groups, make it public, add or replace kernel arguments, or update the description.

```shell
$ igor distro edit NAME { [-n NEWNAME | -o OWNER | -a GRP1,... | -r GRP1,... | -k KARGS | --desc "DESCRIPTION" | -p ] }
```

### Change the Distro's name

You can change the name of the Distro using **`-n | --name`**. Distro names must be globally unique.

```shell
$ igor distro edit MyOldName -n MyNewName
```

### Adding/Removing Groups

You can share a distro you own with groups you belong to using the **`-a | --add`**  flag. Any group member can then use it to make a reservation. Likewise you can also remove assigned groups using the **`-r | --remove`**  flag.

```shell
$ igor distro edit ForceOS -a TheRebellion,JediKnights

$ igor distro edit ForceOS -r StormTroopers,TuskenRaiders
```

If you own a Distro with an assigned group and you are subsequently removed from the group's membership at some point, the group is automatically removed from the Distro.

### Change a Distro to Public Use

You can make your distro public using the **`-p | --public`**  flag. When you include this flag, Igor will change ownership of the distro to the admin team and set the distro to be available for all users. **_This change is permanent._**  The only way to reverse this action is to have an admin delete the public distro, then re-create your original distro as a private one.

```shell
$ igor distro edit MyTinyOS -p
```

### Revert a Public Distro to Private New in v2.2

This is an administrator-only feature. Sometimes public distros need to be recalled or de-commissioned, even though they might still be in use.

You can revert a public distro to private using the `--deprecate` flag. This causes a few changes to occur.

  * The distro owner is changed to the admins group (from igor-admin).
  * The distro is no longer listed as public and won't show up when non-admin users run `igor distro show`.
  * Any existing reservations using the distro will still be honored, but non-admin users cannot extend existing reservations or make new reservations with the distro.

The last point gives the admin team leeway to not interrupt current work being done with the distro while ultimately deciding is they want to phase out its remaining reservations (before deleting it) or re-assigning ownership to a private group for continued use by a subset of users.

If ownership is re-assigned, the distro will continue as a private distro and can be seen and used normally by member(s) of the new owner group.

```shell
$ igor distro edit ThisOldPublicDistro --deprecate
```

### Add or Replace Kernel Arguments

You can add kernel arguments to the distro using the `-k`  flag. If the distro already has kernel arguments, they will be replaced.

```shell
$ igor distro edit MyDistro -k "--append=debug=true"
```

As a best practice, you should include only arguments that _must_  be run with this image every time it's used by anyone. Optional or test kernel arguments should instead be used in a Profile (see Profile section below).

### Add or Replace the Description

You can add a description to the Distro using `--desc`. If the Distro already had a description, it will be replaced.

```shell
$ igor distro edit MyDistro --desc "This is my new description"
```

## Deleting a Distro

To delete a distro you own, use:

`igor distro del NAME`

```shell
$ igor distro del MyUbu20
```

###

A distro cannot be deleted if it is being used by a reservation. The reservation must end or be deleted first. Additionally, if the underlying KI pair for the distro you delete is not used by another distro, the KI pair will also be deleted.
