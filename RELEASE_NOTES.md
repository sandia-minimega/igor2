# Release Notes

## v2.3.4

17-September-2026

***This release includes a change to the igor-server.yaml config file. See full documentation for details.***

### Updates

- Add forcePXE setting to external commands section of server config (see description in config for usage notes).

### Security Updates

- Bump minimum required Golang to 1.25.x
- Updated x/crypto package to 0.52.0
- Updated x/net package to 0.54.0

## v2.3.3

11-September-2026

***This release includes a change to the igor-server.yaml config file. See full documentation for details.***

### Updates

- Added a configurable timeout to arista RPC calls. This can be changed from the default 30 seconds in the igor-server.yaml file.
- Added GOTRACEBACK setting to the server systemd service file to assist diagnosis of problems.

### Fixes

- Fixed an issue to stop the standard panic handler from re-panicking and respond instead with the intended HTTP 500 code.
- Fixed an issue where every multipart file upload would strand its spooled file in /tmp instead of cleaning it up when finished.
- Fixed an issue where queuing email notices inside database access calls could lead to a deadlock state that prevented emails from being sent.
- Fixed an issue where image registration could deadlock against the initrd worker.
- Added a timeout to server shutdown and better handling and resolution for shutting down workers if they are stuck.
- Replaced database write operation calls with a unified mutex architecture that properly defers releasing the database lock until it is safe to do so.
- Arista RPC calls now put credentials into an HTTP Basic Auth header instead of fields in the URL.
- Fixed an issue where trying to access a missing email template for removing a group owner or when reservation notifications are turned off would cause a server panic.
- Fixed an issue where the help link was being substituted as a sender address when the igor-admin email field was blank. A warning is now sent to the igor log if igor-admin doesn't have an email address to send to.
- Fixed an issue where queued mail for a removed user was re-queued every minute indefinitely instead of dropped.
- Fixed an issue where use of the percent-sign '%' character in format strings could result in mangled output for the MOTD and console messages.


## v2.3.2

02-April-2026

### Updates

- Public endpoint extended with group membership query.

### Fixes

- Took out sections in CLI help text that were no longer relevant.

## v2.3.1

17-February-2026

### Updates

- Igorweb has a dark mode! Toggle setting is remembered by the browser.
- CLI calls out nodes with no power status in break-out node summary.
- Security updates to crypto, net and xz package dependencies.
- Bump minimum required Golang to v1.24

### Fixes

- Fix proper display of on and ping statuses in group res, blocked and restricted nodes.
- Minor color and wording fixes to igorweb banner and legend text.

## v2.3.0

13-February-2026

***This release requires changes to the igor-server.yaml config file. See full documentation for details.***

### Updates

- Host network status uses internal go TCP alive requests to determine if a host is reachable. This removes the linux utility `nmap` as an external dependency called by the server for the same purpose. 
- The entire power + connectivity interface has been re-designed to display additional statuses. Nodes now have indicators for powered-on but no network availability and ping-able status but no TCP connectivity. This is covered more extensively in the documentation.
- Distro names have been appended to the public endpoint for getting a list of active reservations.

### Fixes

- Fixed an interval timer issue where the notification manager could fill up the log with DEBUG messages when the clock shifted back to Standard Time.
- Fixed an issue where adding a new node to the cluster would freeze the node status as unknown and would require a server restart to get the status unstuck.
- Numerous small fixes to igorweb.


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
- Bump minimum Node.js to v22.x and npm to v10.x to build Igorweb.
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
