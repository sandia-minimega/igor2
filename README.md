# IGOR

-------
Igor is a node reservation manager for clusters that host ad-hoc multi-user/multi-project work.

Users can reserve and provision cluster nodes with OS images of their choice. Users can also
create groups that share access to reservation management.

Administrators can fine-tune reservation creation and resource access across a wide variety
of settings from open and permissive to regulated and tightly controlled.

The first version of Igor began as part of the [minimega](https://github.com/sandia-minimega/minimega) 
toolset. Version 2 is a sibling project and a significant upgrade from the original.

## Documentation

Please visit: https://www.sandia.gov/igor

## Architecture
Igor has evolved from a simple scheduling app that updated its entries with a cron job and
JSON files to a full-blown client-server architecture communicating through a REST API over
HTTPS and backed by a database.

An Igor server can be contacted from anywhere by an Igor client (web-app and CLI versions are
included), as long as the server's URL can be reached from the calling computer. From a
security standpoint this means users don't have to be given login privileges on the cluster
head node to run Igor commands.

## User Accounts and Groups
Igor tracks its own registered users. Even if the client is installed on a general node, login
credentials are required to verify the user can access Igor.

The Igor account system can be configured to use a network's LDAP service for authentication
and account management. However, Igor can also track and manage its own local passwords if
LDAP isn't available or the cluster team wants to stand up Igor without it.

Groups have been enhanced with Igor either internally creating and maintaining its own or
syncing with LDAP-created groups. This provides flexibility for any user to create a group
of users with whom they can share resources. The group owner maintains the power of modifying
a group, but they can also transfer ownership to another Igor user if needed. Groups are used
throughout Igor to provide command and control access to reservations, distros, and other
resources needed by multiple users.

## OS Image Management
Igor manages all OS images internally without the relying on third-party apps. Users can upload
kernel/initrd file pairs to the server for their exclusive use or to share them with the
community. Alternatively, an admin team can turn off user uploading and manage OS's themselves
if they wish to control which images are allowed to run on cluster resources.

### Net or Local Boot
Igor's leverages PXE and TFTP to netboot images by default. However, it can also boot an image
locally on an assigned node. Working with an admin to provide proper pre- and post-boot scripts
is required. Both BIOS and UEFI hardware boot configurations are supported.

## Timely Reminders
Igor sends out a wide variety of email notices when current or impending status changes merit
getting a user's attention.

## Greater Administrative Control
Igor comes with a plethora of administrative options for cluster admin teams, both in
external configuration and within Igor itself.

On the inside, admins get access to many privileged commands that help them manage Igor and
cluster scheduling interactively. For example, host policies allow admins to customize how
long a host can run a single reservation, can fence off hosts for exclusive use by select
user groups, or shut off reservation scheduling for particular nodes during planned downtime
like routine maintenance periods or holidays.

On the configuration side there are many settings that set down limits of node usage,
reservation scheduling windows, user file upload capabilities and more.
