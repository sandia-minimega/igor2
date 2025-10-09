# Release Notes

## v2.2.1

9-October-2025

### Fixes
- Fixed a bug where profile-switching was failing. (Thanks to user mabbonda for catching this!)
- Fixed search filtering bug in 'igor res show'.


## v2.2.0

2-October-2025

### New Features

- Igor now probes K/I pair files for kernel and OS metadata and publishes its results in the 'kernel' and 'initrd' columns displayed by the igor distro show command.
- Reservations and distros for auto-removed owner accounts are re-assigned to administrators for possible owner re-assignment to other users or manual deletion.
- Users can now add hosts to existing reservations.
- Administrators can make a public distro private and owned by the admin group using the --deprecate flag when editing a distro.

### Updates


- More detailed line-wrap formatting within table cells in CLI output for better readability of dense information.
- Removed some CLI output columns in information tables that had little impact for the amount of space used. All information is still retained in non-table formatted version (using -x flag).
- More detailed server logging that always specifies the user associated with a given command action.
- The igor sync command can now operate on a specific set of hosts.
- Updated db-migrate tool to migrate Igor databases from previous versions to this one.
- Bump minimum required Golang to v1.23.x
- Bump minimum nodeJS to v22.x and npm to v10.x to build Igorweb.
- Security package updates.

### Fixes

- Group-restricted nodes can be extended if the same group applied to the reservation matches the one declared by the restriction policy.
- Default to returning only the user's reservations when using igor res show. Add the --all flag to get everything.
- Host blocking now correctly handles hosts in maintenance mode.
- Fixed some ASCII terminal formatting issues.

## v2.1.3

03-April-2025

### Security Update

- Updated axios package for Igorweb.

## v2.1.2

17-January-2025

### Critical Patch

- Updated kernel arguments for UEFI boot and installed RedHat images with BIOS.

## v2.1.1

30-September-2025

### Critical Patch

- BIOS installed images specify MAC address for interface, not auto.

## v2.1.0

16-September-2024

### New Features

- Added support for UEFI hardware boot configuration.
- Igor can now automatically sync account creation and removal with one or more LDAP user groups.
- Igor user group membership can now be synchronized with LDAP user group membership. An LDAP-synced group can only be created by the owner or delegate of the corresponding LDAP group. Only members with an Igor account are synced, so the LDAP group does not need to have a strict 1:1 match with Igor account holders.
- Igor groups (both internal and LDAP-synced) now support multiple owners.
- The admin host block feature can now be used on reserved nodes. Users will not lose access to blocked nodes, but reservation extensions will not work until the node is no longer blocked.
- Igor now includes a database migration feature for upgrades that require changes to tables.
- Added an explicit igor login command for the CLI.
- The igor res show command now defaults to displaying only the user’s reservations, similar to igor show. Use the --all flag to see all reservations.

### Updates

- Bump the minimum Go language compiler requirement to version 1.21.
- Update go crypto package to address CVE vulnerability.

### Fixes

- The single-node selection in Igorweb when making a new reservation by CTRL-clicking on the node map now works correctly.
- Fixed the issue where maintenance-mode reservations were not getting the correct end time if the server had been off for an extended period.
- Resolved the issue where some email reservation expiry warnings were not being sent.
- Fixed the issue where some node power permissions were being dropped while reservations were still active.
- Corrected the issue where some power commands were sometimes using non-existent hostnames.
- The igor res show -d flag now correctly filters on distributions.
- The igor show -f flag now correctly indicates if the user has no future reservations.

## v2.0.0

20-April-2023

Initial release!
