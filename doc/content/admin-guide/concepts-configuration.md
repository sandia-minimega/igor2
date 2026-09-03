# Concepts and Configuration Options

At its most basic level, Igor helps the user reserve cluster nodes for a period of time and launch their chosen OS image on those nodes. But for people who run the cluster, Igor provides a much greater service -- handing off much of the work of that job to an automated service. Igor's new design is centered around both a richer user experience and a plethora of new administrative capabilities that allow cluster teams to tailor it to their specific needs.

## User Roles and Permissions

In the Igor realm, there are two roles: _users_ and _admins_.

As mentioned in the setup guide, Igor administrators are normal users that have been added to the **_admins_** group and a user can only be added to that group by another admin.

This design gives the cluster admin team the freedom to give other members of the cluster user community the ability to run privileged commands. There can be many reasons for doing this. For example the cluster hardware team might want to leave the management of cluster scheduling up to people who will take on that role and assist users when special requests are made.

The roles grants certain permissions.

### Users can:

  * view all or most information about
```shell
* reservations
* cluster nodes
* users
* groups
```

  * create, edit and delete their own
```shell
* reservations
* groups
* distros
* profiles
```

  * update their own
```shell
* email address
* password (when Igor uses local authentication)
```

  * edit or delete
```shell
* reservations they can access via group membership
```

  * run power commands on nodes they have reserved
  * get select helpful pieces of configuration details

### Admins can do all the above, plus:

  * edit the cluster configuration
  * create, edit and delete Igor users
  * edit the membership of the admins group
  * reset user passwords (when Igor uses local authentication)
  * manage OS images used to make distros and profiles
  * edit and delete other user's reservations, groups, distros and profiles
  * create and edit reservations in ways that exceed normal user limitations
  * create, edit and delete node reservation policies
  * block (and unblock) cluster nodes from being reserved
  * run power commands on any cluster node
  * run special-purpose commands (stats, sync and auth-reset)

### The 'elevate' command

For admins to run admin-level commands, they must first get elevated status by running:

```shell
$ igor elevate
```

This status remains in effect for a period of time (set in the server config) before it needs to be run again. An admin can add the `**-s**` flag to see how much time is left in elevated status or `**-c**` to cancel out of it. If an admin attempts to run a privileged command without elevated status Igor will send a message explaining the command can be retried after elevating.

Running a password modification command for a user when LDAP is configured for auth will result in an error message, but read the following section regarding the igor-admin account.

### Choosing Your Authentication Method

**_It is strongly recommended to use LDAP if it is available._** This will reduce the need for users to remember another password and naturally align Igor account names with login names for your LDAP-enabled network.

If local authentication is chosen, Igor stores user passwords in its database as bcrypt hashes. Using local offers an advantage in being able to have user account names that are different from OS login names, for instance if there is a reason a single user would want multiple user accounts they can log into the service with.

#### Password Resets (Local Auth Only)

If local authentication is enabled, users will occasionally need assistance with a forgotten password. Igor admins can reset any user's password with the command:

`igor user reset USERNAME`

This will set the user's password back the default (specified in the config file) and send an email informing the user of the reset.

It is strongly advised that any password reset is followed by the affected user changing their password as soon as possible to a more secure one.

#### Resetting the igor-admin account

###

The igor-admin account is always local even if LDAP is enabled.

Use the same command as above for resetting the password of this account. It will be reset to the default, "igor-admin". Note that it is not possible to specify a different default.

All members of the admins group will be cc'd on the email that the igor-admin password has been reset.

### Adding, Updating and Removing Users

#### Manual Account Management

Since Igor maintains its own user accounts regardless of authentication method, it is necessary for the admin team to maintain this list accordingly. Even if a user has an OS account on a machine where Igor is accessible they will not be able to use Igor until an application-level account is created for them. Igor does not currently support external group mechanisms (offered by the OS or LDAP) to determine who has the ability to login.

**_To create a new Igor user_** execute the following command as an elevated admin:

`igor user create USERNAME EMAIL [-f "FULL NAME"]`

In most cases, USERNAME will match the machine login account name, although it is possible to use other names with local authentication configured. EMAIL should be the primary email where the user expects to receive notifications. The full name field is optional but highly useful as a recognizable proper name that will make them easier to identify to the user community.

If local authentication is configured, the new user alert sent by email will contain the default Igor user password. If LDAP is configured it will instruct the user to use that password for access.

Once created, users can edit their own email and full-name fields. Local auth passwords can also be changed. The edit command by default operates on the logged-in user's account, so the Igor username doesn't have to be provided.

`igor user edit {[-n USERNAME]` `-e EMAIL -f "FULLNAME" | --password }`

**_To update an Igor user 's information_** the optional `**-n**` flag can be used by an elevated admin to specify which user account is being changed.

**Example:**

```shell
$ igor user edit -n srogers -e CaptAmerica@avengers.team -f "Steven Rogers"
```

The option for password change cannot be used with any other flag and _cannot_ be performed by an admin on someone else's user account. However, admins _can_ reset a user's password to the default. This is covered in the Password Resets section below.

**_To remove a user from Igor_** execute the following command as an elevated admin:

`igor user del USERNAME`

Users cannot own any reservations at the time of deletion. Ownership of existing reservations must be transferred to another user or the reservation deleted first. Once deleted, the user will no longer be able to run Igor commands. Their account is dropped from the Igor database but can be re-created at any time.

#### Automated Account Management

Igor can automatically create and remove accounts via LDAP. This feature completely replaces manual account creation. It also assumes you have set up your Igor server to use LDAP authentication.

**Important Note:**_You must change**igor-admin** 's default account password in order for this feature to work. Without this change, account syncing will not take place and an error will be recorded in the log. The **igor-admin** account is always a local Igor account. It does not map to a LDAP user._

To use automated account management, fill in the necessary sections of the Igor server config file under the _ldap_ -> _sync_ settings.

Specifically:

  * set **enableUserSync** to _true_
  * set **syncFrequency** to how often LDAP will perform a sync (in minutes)
  * set **groupFilters** to a list of LDAP group names that contain users who need Igor accounts
  * set **userListAttribute** , **userEmailAttribute** , **userDisplayNameAttribute** and **groupOwnerAttributes** in accordance with your LDAP system's naming convention for each piece of information.

Note that Igor does not distinguish between owners and delegates of an LDAP group. It is recommended to provide the attributes for each in the config so Igor treats each one as an owner.

Once the server is running with the above configuration, Igor will automatically create user accounts for any user that is referenced by **groupFilters** and remove them if they are dropped. If a user is present in more than one group, they need to be removed from all of them in order for account deletion to occur.

New in v2.2 Should a user that is automatically removed have ownership of reservations or distros that are shared with others, two possible situations can occur. If the reservation is shared with a group, ownership will revert to igor-admin and an alert with notify administrators that the reservation's ownership needs to be investigated for re-assignment to another group member or deleted. In the case of a distro, if ownership is shared with another user(s), then no action is taken other than removing the user from the owner list. If the user is the sole owner but the distro is shared, a similar transfer of control to igor-admin will occur for follow-up by the admin team.

The timing interval of account creation/deletion is dependent on the value of **syncFrequency**. A sync will always be performed when the server starts.

**Note:** Even when LDAP is used to add/remove accounts, users still have the ability to update their email and full name in their account details. These changes are not specifically maintained by Igor account management, but if the user is removed and then restored to Igor, it will always use LDAP source information when re-creating the account.

#### Automated Group Management New in v2.1

LDAP can also be used to manage Igor groups. This is also configured in the LDAP sync settings of the config file. Specifically:

  * set **enableGroupSync** to _true_

When enabled, Igor will allow the creation of groups that sync periodically with the LDAP groups of the same name, maintaining their membership list by comparing them to the source. Such groups can only be created by a group owner or delegate as specified by LDAP, or an Igor admin. Once created a group's membership is controlled by LDAP and is not accessible through Igor. Owners, delegates and admins can delete a synced group when it is no longer needed.

See the "Groups" documentation for more information.

## Cluster Nodes

An Igor cluster is defined in the _igor-clusters.yaml_ configuration file. The cluster itself simply requires a name and a prefix and represents the nodes being used. Within the cluster settings, you also need to specify the display width and height. This refers to how the nodes are displayed in the CLI. These values should be based on the number of nodes you intend to include, but does not need to correspond to any real-world rack configuration.

### Adding, Editing and Removing Nodes

The _igor-clusters.yaml_ file can be edited and the changes picked up by Igor server when you run a command.

#### Adding Nodes

If you need to add a new node, you should do the following:

```shell
$ igor cluster show --dump  # backs up the current config

# At this point, edit the current igor-cluster.yaml config file in a text editor

$ igor cluster config       # reloads the edited config
```

#### Editing Nodes

Editing nodes allows the admin to change existing information in the config file without having to edit it directly.

`igor host edit NODENAME {[-p POLICY] [-d HOSTNAME] [-b BOOT] [-e ETH] [-i IP] [-m MACID]}`

Each flag except for policy represents a field in the config that can be changed. Using this command will back up the current config and apply the changes to the new config.

#### Deleting Nodes

Deleting a node is simple. Running the command below will back up the existing cluster config and remove the node from its list. A node cannot be removed if it has reservations attached to it.

`igor host del NODENAME `

Note that removing a node will create a gap in the node-numbering sequence. This can cause reservations that might include that node's name to fail. For example:

```shell
$ igor host del kn37
$ igor res create CoolStuff -d SomeDistro -e 5h -n kn[30-40]
igor: host(s) 'kn37' not found
$
```

## Dealing With Node Downtime

Cluster node downtime happens for many reasons, and sometimes -- usually in a failure scenario -- admins will not know how long it will take to restore the node back to service. During this time cluster admins don't want to remove the node from cluster, just prevent users from accessing it.

Igor supports this condition with the host `**block**` and **`unblock`** commands. Blocking a node removes it from Igor's reservation pool until it is unblocked. During this time Igor's node map display will highlight the node's background color in amber and the node's ID will be added to the cluster detail line for blocked nodes.

`igor host {block|unblock} NODES`

**Examples:**

```shell
$ igor host block kn[4-10,65,90]

$ igor host unblock kn55,kn23
```

### Users cannot take reservation action on blocked nodes

Users will receive an error message if they attempt to reserve a node while it is blocked, or if they try to extend a reservation that has one or more blocked nodes. . If a block is removed while the node is still reserved, it resumes it's normal schedule. And if a reserved node's reservation expires, is deleted or the node is dropped, the block remains.

Nodes can be blocked even if they are part of an active reservation. When this happens, Igor will send an email to affected reservation owner(s) to let them know what has happened. For a currently running reservation and assuming the node hasn't crashed, users can do what they need to stop and save any important work.

During the block of a node in an active reservation, users can still issue power commands to it.

Dropping affected node(s) from the reservation is an easy way for users to maintain their existing reservation if a node block looks like it will last for an extended period. Dropping a node is a permanent change since adding nodes after creation is not allowed. If the user needs an additional node to replace the dropped one, a potential workaround would be to make a new reservation with the existing reservation's name as its VLAN parameter.

There may also be scenarios where admins want planned downtime of nodes. This can be done ad-hoc with either blocking, having an admin reserve the nodes in question for the given time period, or creating a policy that restricts the designated nodes to a group of admins.

## Reservation Management Configuration

Reservation management config parameters can be set to reflect the needs of the cluster admin teams and community. These setting are found in the igor-server.yaml config file.

### NodeReserveLimit

This the maximum number of nodes a user can have in a reservation. If more are desired then the user must make another reservation or request an admin to make a single reservation in their name with the requested number. This setting can promote fairness in handing out resources, however there is no setting to limit the number of reservations a user can make. Therefore, this setting cannot be used to impose a hard cap on the amount of cluster resources a given user can hold.

### MaxScheduleDays

This defines Igor's scheduling window. The window starts at now (according to the current system time of the _igor-server_ host) plus however many days are specified by this setting where a day is defined as 24 hours. It is within this window that Igor will allow reservations made by users. For example, if now is 8 AM on Jan 1 and _MaxScheduleDays_ is set to 365, then Igor will allow reservations to be made through the entire year whether they start immediately or at some time in future before the cutoff.

The default for this setting is its max value of **1457** days, which is equivalent to 4 years plus 1 leap day. Most if not all cluster teams will choose to set their scheduling window much shorter. It is recommended for the team to discuss how far in advance users should be able to schedule cluster time and change this parameter accordingly.

### MinReserveTime

This is the smallest amount of time that a user can schedule a reservation. The value is expressed in minutes. Igor will not allow reservations using a smaller time value unless the reservation is made by an elevated admin. _There is a hard minimum cap on this value of 10 minutes._ Going lower increases the risk that some hardware nodes may take more than half the total time of the reservation just to boot their OS image.

The default for this setting is **30** minutes.

### MaxReserveTime

This is the largest amount of time that a user can schedule a reservation. The value is expressed in minutes. Igor will not allow reservations using a larger time value unless the reservation is made by an elevated admin.

The default for this setting is **43200** minutes, which is equivalent to 30 days.

### DefaultReserveTime

This is the amount of time a reservation will last if its length or end datetime is not specified when the reservation is made. The value is expressed in minutes. It can be equal to _MinReserveTime_ or _MaxReserveTime_ but cannot go below or above those values.

The default for this setting is **60** minutes.

### ExtendWithin

This is a period (in minutes) during which the reservation edit `**--extend**` and `**--extend-max**` flags can be used to push out a reservation's expiration time. The period counts backward from a given reservation's end timestamp. For example, 8640 minutes is equivalent to 14 days, meaning that a user is only allowed to extend her reservation when there is two weeks or less of time remaining on it.

The rationale behind this setting is that to keep things fair on a busy cluster, users must be forced to wait before extending reservations in order to give other users the chance to sign up for nodes. Otherwise users could run scripts that continually push their reservations out every few minutes and hog near-future availability of resources. During this allowed window a user can attempt to extend their reservation and if no one else has reserved one or more of the reservation's nodes during the desired extension period, Igor will grant the new end time.

A successful extension cannot make the reservation last longer than current server time plus _MaxReserveTime_ minutes. Therefore, the max extension time that can be asked for is _MaxReserveTime_ minutes minus how much time the reservation has left when the command is executed. If the maximum amount of time is desired, the `**--extend-max**` flag lets Igor automatically calculate the new expiration datetime based on the above conditions.

###

If the cluster admin team does not want to allow extensions, then set this value to 0. Elevated admins can still use the extend flags. This lets the admin team allow extensions based on external requests, although it then requires an admin to execute an approved one.

The default value is **4320** minutes, the equivalent of 7 days. Changing this value to align with the first notification of approaching reservation end time is usually a good idea. (See section on email notifications.)

## Policies

Among the new features that Igor provides, policies allow for more fine-grained scheduling options of cluster nodes that are dynamically managed by the admin team while Igor is running.

Policies change the scheduling behavior of cluster nodes at the node level. They allow:

  * changing the maximum amount of time a reservation can last on a node
  * allowing one or more groups to have the exclusive right to reserve the node
  * defining a period of time when a node is unavailable to be reserved
  * any combination of the above

To see a list of all existing policies, use:

```shell
$ igor policy show
```

Each policy will display its name, any hosts it's been assigned to, max reservation time, any access-groups assigned to it, and time periods when its assigned hosts become unavailable.

### Creating Policies

To create a new policy, the syntax is:

`igor policy create NAME {-g [GROUP1,...] | -t MAXTIME | -u NOTAVAIL}`

Policies can be applied at any time, even if reservations exist on nodes that would not be allowed under the new policy. Any existing reservations are allowed to run their course and expire as normal. The only caveat is they can't be extended. Admins can, of course, encourage users to end reservations prematurely or forcibly end them should enforcement of a new policy require timely action.

#### Setting a Different MaxTime for Selected Hosts

Using the `**-t**` flag allows for creating a policy where the _MaxReserveTime_ is different from other nodes. Specify a duration value using days, hours and/or minutes.

**Example:**

```shell
$ igor policy create 3Months -t 90d
```

#### Making Hosts Only Available to Certain Groups

Using the `-g` flag allows for creating a policy composed of user groups. Any host with a group policy can only be reserved by the members of said group(s).

**Example:**

```shell
$ igor policy create HackersOnly -g TheCollective,FinalFive
```

###

Nodes can be held by policy for single users, but that can also be achieved by an admin extending their reservation window for a lengthy period of time. Using a policy to achieve this requires creating a group with only the user in it.

#### Scheduling Host Unavailability

This kind of policy produces the same effect as the host `**block**` command, but instead of being open-ended it is a scheduled interval, allowing the admin team to specify periods of time in advance when hosts are not available to users.

Using the `-u` flag, an unavailable value takes the format of a _start:duration_ string composed of a cron expression, followed by a colon, then a duration value:

`**"* * * * *:dur"**`

For more information on cron expressions see: <https://en.wikipedia.org/wiki/Cron>. The duration is a standard Igor duration expression.

**Example:**

**`"0 0 * * 6:2d5h"`** _- > __from Saturday at 12:00 AM to Monday at 5:00 AM every week._

```shell
$ igor policy create NoWeekends -u "0 0 * * 6:2d5h"
```

#### Combined Policy Types

You may mix any number of policy type flags together to suit the needs of users.

**Example:**

```shell
$ igor policy create WeekdayHacking -t 4d19h -g TheCollective,FinalFive -u "0 0 * * 6:2d5h"
```

### Applying Policies

To assign a policy to one or more hosts, use the syntax:

`igor policy apply POLICYNAME NODES`

The NODES parameter uses the same range notation (ex. kn[3,7-9,22-35,47]) as other commands.

**Example:**

```shell
$ igor policy apply Rack1Maint pp[1-20]
```

Applying a policy that modifies host access by group or scheduled unavailability will cause the clients to display those hosts as **restricted** to users who can't access them.

###

Assigning a policy to a host does not affect existing reservations (current or future) that include that host. It only applies to new reservations made after the policy goes into effect. If a policy change would not allow an existing reservation to be made, such reservations lose the ability to be extended.

### The Default Policy

Even if policies are not used, every node on the cluster has a default policy that makes it available 24/7 to all users with the maximum reservable time equal to the `MaxReserveTime` setting in the _igor-server.yaml_ configuration file. Any time a node's policy needs to be canceled, the default policy can be applied to restore this behavior. Also, whenever the `MaxReserveTime` setting is changed in the config file, Igor will update the default policy to reflect the new max time.

**Example:**

```shell
$ igor policy apply default cr[54-67]
```

### Editing Policies

To modify an existing policy, use the syntax:

`igor policy edit NAME { [-n NEWNAME] [-t MAXTIME] [-g GRP1,…] [-r GRP1,…][-u "EXP1",…] [-x "EXP1",…] }`

where `**-g**` will add the specified groups to the policy while `**-r**` will remove the specified groups from the policy and `**-u**` will add the specified not-available instance(s) to the policy while `**-x**` will remove the specified not-available instance(s) from the policy.

Modifying an existing policy _will not_ affect a reservation that is currently active on the host that receives the update. For example, adding an unavailable window to the policy of a host that would not allow its currently running reservation will not terminate the reservation, but it will prevent it from being extended if the period of unavailability extends beyond the reservation's current end time.

### Deleting Policies

To delete a policy, use the syntax:

`igor policy del POLICYNAME`

**Example:**

```shell
$ igor policy del Rack1Maint
```

The policy cannot be associated to any hosts when deleting. If necessary, reset those hosts back to the default (or another policy) before performing this action.

## Image Management

Igor allows for the management of boot images and their delivery to the nodes of the clusters it manages. At this time, only kernel and initrd (KI) pairs are maintained, though they can be either for netbooting or for local installation. There are three main components Igor uses to maintain and install a boot image, where each builds on top of the previous:

  * **Image:** The kernel/initrd pair is registered to Igor as an image object that contains paths to the files' storage location on disk and metadata describing the OS breed and whether the image is intended for installation or not. Standard users do not see or use image objects directly.
  * **Distro:** A distro object refers to a single image object. It provides additional boot data such as kernel arguments (if a netboot image) or a kickstart/preseed script (if an installed, or localboot, image). Distros also contain ownership and group data, defining which Igor users can deploy it in a reservation. Some distros are set to _public_ access, making them available for anyone to use.
  * **Profile:** A profile object is a wrapper around a single distro for the purpose of including additional kernel arguments at boot time. Profiles handle more specialized use-cases where booting the same OS image under different setups is required. Users can create as many profiles as they need for a given distro.

When creating a new reservation, a user can specify either a profile or a distro. If a user chooses a distro, Igor will create a temporary profile under the hood (with no added kernel args) using the selected distro to install. When the reservation ends, the temporary profile is destroyed.

There are several ways a distro can be created, though some are only available to the admin. For example, if _allowImageUpload_ is set to true in the igor-server configuration, users can upload a KI pair directly when creating a new distro (which registers the KI pair as an image object in the same process). Otherwise, the KI pair must be registered as a separate step first, which can only be done by an elevated admin.

In the case of images intended for installation, Igor provides additional endpoints the nodes must call once finished with initial installation in order to trigger the PXE install process to configure the boot instructions so the nodes will boot locally on subsequent power cycles. Other optional endpoints are available to act as a file server for the node to retrieve additional packages, libraries and scripts for further customization. These must be configured in the kickstart script that is registered to Igor to use in a distro. Details on how to configure the kickstart script with these features are covered below.

### Images

An image consists of:

  * **name:** in the format of a set prefix (currently 'ki' only) followed by 8 characters
  * **ID:** a hash value generated from the image file(s)
  * **type** : currently ki only
  * **image file names**
  * **local:** a boolean indicating if the image is intended to be installed
  * **breed:** OS type. Must be one of:
```shell
* debian
* freebsd
* generic
* nexenta
* redhat
* suse
* ubuntu
* unix
* vmware
* windows
* xen
```

To see a list of all registered images, use

```shell
$ igor image show
```

This is the easiest way to grab an image name if you want to use it to create a new distro.

**_To register an image_** copy the image files into the _imageStagePath_ as specified in the igor-server configuration. (Igor will remove these files from the staging folder once they are successfully registered.) Then in the command line use:

`igor image register -k KERNEL_FILENAME -i INITRD_FILENAME [-l -b BREED]`

If an image will be used to create a distro intended to be locally booted from disk, you are required to add the `**-l**` flag for local and the `**-b**` flag with one of the breed options listed above.

**Examples:**

```shell
$ igor image register -k tinyos.kernel -i tinyos.initrd
**ki4daa05be**

$ igor image register -k ubu20.kernel -i ubu20.initrd -l -b ubuntu
**ki73c822f1**
```

Successful registration of the image will return an image reference identifier. If this image is for private use only, the id should be communicated to the user(s) who will create distros from the image.

```shell
$ igor distro create TinyOSPublic --image-ref ki4daa05be --public

$ igor distro create FaeRealm --image-ref ki73c822f1 -g TheFae --kickstart fae.ks
```

**_If an image needs to be deleted_** (before it's associated with a distro) use:

`igor image del -n IMAGEREF`

### Kickstart

When creating a distro using an image intended for installation and local booting, a kickstart script is required to be specified in the distro creation parameters. A kickstart script contains all the "answers" to the questions that are asked during the installation process, as well as the Igor endpoints that need to be called either to trigger local booting or pull additional files. A kickstart file must be first registered to Igor before it can be referenced in the distro creation process.

To see a list of all registered kickstart scripts, use

```shell
$ igor kickstart show
```

To register a new kickstart script, use:

`igor kickstart register -k /absolute/path/to/kickstart.ks`

```shell
$ igor kickstart register -k /srv/scripts/hackathon-playground.ks
```

#### Igor Endpoint Callbacks

Igor provides three endpoints that can be used to assist in the automated installation process. These endpoints serve as a file server, local-boot trigger, and to provide user and reservation information.

Base Callback URL: `http://{server-host:callback-port}/igor`

This is the base URL that starts all three calls. Values can be found from the _igor-server.yaml_ file. Note the callback port is different from the primary igor-server port. The endpoints that follow are given as a relative path to append to this.

File server: `_base_cb_url_ /cb/svc/scripts/`

This endpoint will serve files specified after this path. The true path on the server is specified under scriptDir in the igor-server configuration.

Switch to local booting: `_base_cb_url_ /cb/svc/local`

This endpoint triggers the change in configuration to cause the node to boot locally on restart

Get reservation info: `_base_cb_url_ /cb/svc/info`

This endpoint responds with a string which contains the reservation name, user name, and reservations nodes related to the node this endpoint is being called from. You can potentially use this information to set up a user on the system, coordinate with the other reservation nodes, etc.

#### Customizing a Kickstart File

Information on using kickstart templates for PXEbooting will depend on the OS intended for use. For example:

  * Debian-based installations use "preseed" kickstart scripts. Information, including templates, can be found at:
<https://wiki.debian.org/DebianInstaller/Preseed>
  * RHEL-based installations use kickstart scripts. Information on automated installations can be found here:
<https://docs.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/performing_an_advanced_rhel_8_installation/index>
  * CentOS-based installations use the same kickstart format; the pykickstart
    reference documents the available commands:
<https://pykickstart.readthedocs.io/en/latest/kickstart-docs.html>

While the details of these kickstart files will vary by OS, some pieces are consistent across them all where customization for Igor is essential, namely specifying a URL where the process can pull installation files.

Kickstarts will have language to add where the user can specify a URL where the installation can download files to install. For example, Debian-based kickstarts may look like the following:

#### Setup the Installation Source

```shell
d-i mirror/http/hostname string kn-mc6:8444

d-i mirror/http/directory string /igor/cb/svc/scripts/ks_mirror/ubuntu-18.04-server

d-i mirror/http/proxy string
```

Igor can act as a local repository for install files using the file server endpoint above.

  * "late_command" or "post" action sections where post-install commands can be specified

The critical addition to the kickstart script is the inclusion of the call back URL to trigger local booting. This must be specified in the section of the kickstart that is executed _after_ the OS is installed to disk.

### Distros

Basic usage is [covered](../user-guide/distros.md) in the User Guide. However the admin can perform additional actions or see enhanced results when elevated:

  * The distro `show` command will return a full list of all distros known to Igor for all owners.
  * Change the ownership of a distro to another user using the `-o` flag. (See help in the CLI distro edit command for details.)
  * Edit or delete any distro (subject to current use conditions).

### Profiles

Basic usage is [covered](../user-guide/profiles.md) in the user guide. However the admin can perform additional actions or see enhanced results when elevated:

  * The profile `show` command will return a full list of all profiles known to Igor for all owners. This includes any temporary/default profiles created for current reservations.
  * Edit or delete any profile (subject to current use conditions).

## Hardware Switch Sync

Igor connects to optional external services to expand its capabilities. The primary example of this is facilitating VLAN through switches. If configured to do so, Igor can send commands out to supported switches to associate or disassociate hosts to VLANs. In doing so, Igor maintains a record which tracks these assignments or, in general, data about actions taken when communicating with external services.

Sync serves as method for checking the local state against that of the related external service, if available, and report back. Additionally, sync can be used to force Igor's state to match that of the service.

To use sync for Arista:

```shell
$ igor sync arista
```

Igor will respond with a list of all nodes, the VLAN value assigned per Igor, and the VLAN value assigned per Arista.

Add the `-q` flag for Igor to report on only those hosts where the VLAN values between Igor and Arista do not match.

Add the `**-f**` flag to conform Arista's VLAN values to Igor's (authoritative). If there is a mismatch of values, Igor will send a command to Arista to assign the VLAN value Igor has on record to the respective host.

New in v2.2 Use the `-s` flag to specify a specific set of hosts on which to sync. This can be either a hostname list or range of hosts or one or more reservation names as a list.

## Stats

Igor provides statistics based on usage which, when elevated, can be evoked using:

```shell
$ igor stats
```

By default, Igor uses the start date as the moment the command was entered with a duration of seven days from the start point.

Adding the `-s` flag with a formatted date will specify a start point for the stats period.

Adding the `-d` flag with an integer value representing the number of days will specify a duration. Specifying 0 will include all history.

Igor will report back global counts for Reservations, Nodes Used (non-unique), Reservations Cancelled early, Extensions used, and Total Reservation Time

Adding the `-v` flag will additionally include information for each user with reservation activity within the given time window. This includes all reservation details made by each user along with a summary of statistics:

  * Reservation Name
  * Reservation ID
  * Nodes
  * Start time
  * Original end time
  * Actual end time
  * Number of extensions

This command can be used to track usage trends for Igor, particularly if its data is forwarded to service such as Splunk.
