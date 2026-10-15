# Profiles

A profile is a simple wrapper that allows you to add additional kernel arguments to a distro. You create profiles when you want to try out different kernel arguments with a distro without changing the distro itself. If the distro associated to the profile also has kernel arguments, the profile kernel arguments are appended after them during installation. Profiles can then be used as a part of reservation creation.

Profiles are specific to users and cannot be shared or made public.

Whenever the distro argument is used to create a reservation, a default temporary profile is created for it that has no added kernel arguments. This profile can be edited. It will also be deleted once the reservation ends.

## Viewing Profiles

To see the profiles you've created, use

```shell
$ igor profile show
```

You'll see a list of all your profiles for distros currently available to you.

## Creating a Profile

To create a custom profile for an existing distro:

`igor profile create NAME DISTRO [-k "KARGS"] [--desc "DESCRIPTION"]`

```shell
$ igor profile create MyProfile2 MyDistro -k "…" --desc "different mounts"
```

## Modifying a Profile

The name, kernel arguments and description can all be changed on a profile using the appropriate flags.

`igor profile edit NAME { [-n NEWNAME] [-k "KARGS"] [--desc "DESCRIPTION"] }`

```shell
$ igor profile edit ThisProfile -n ThatProfile

$ igor profile edit ThatProfile -k "…" --desc "changed the boot args again"
```

## Deleting a Profile

To delete a profile, use:

`igor profile del NAME`

```shell
$ igor profile del UsedUpProfile
```
