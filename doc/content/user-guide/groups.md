# Groups

An Igor group is a collection of Igor users. It's that simple. Igor maintains its own process of creating and maintaining groups since this feature is dynamic and can't rely on an outside service like LDAP. Any user can create a group and add members to it. The group owner maintains the power of modifying a group, but they can also transfer ownership to another Igor user, if needed.

Groups are used throughout Igor to provide command and control access to reservations, distros, and other resources for multiple people.

## Viewing Groups

To get started with groups, you can view groups with a normal show command. The base command hides member names for brevity. You can display members by adding the `**-m**` flag.

```shell
$ igor group show
$ igor group show -m
```

## Creating an LDAP-synced Group New in v2.1

Igor has the ability to create a group that is managed by LDAP. This is a simple command that can only be issued by the owner or a delegate of the LDAP group itself.

`igor group create NAME -L`

Any group created in this manner will automatically sync its membership with those in the LDAP group, including making any owners and delegates the owner(s) of the Igor copy. It will skip any member, owner or delegate of the LDAP group who does not have an Igor account. From this point forward the only way the group membership and ownership can be edited is through the organization's LDAP utility. Igor cannot alter it.

LDAP-synced Igor groups can be removed with a normal delete command (covered below).

To use this feature, the cluster admin team will have to enable it in the server configuration.

## Creating a new Group

To create a new group, issue the following command with list of usernames that you want to include. The list doesn't have to include yourself; the group creator will automatically be added. An optional description can help clarify what the group is being used for. Groups along with their descriptions are visible to all Igor users, but membership in groups is only visible to members and admins.

`igor group create NAME [-o USER1,USER2,`…`][-m USER1,USER2,…] [--desc "optional description"]`

#### Multiple Owners New in v2.1

Of particular note is the optional `**-o**` flag that can be used to add more than one owner to the group. The user issuing the command is automatically made the group owner. Use this flag to name additional owners who will have owner-level edit privileges.

**Examples:**

```shell
$ igor group create Avengers -m ironman,hawkeye,thor,blackwidow --desc "The world's mightiest heroes"
$ igor group create ApertureStaff -o Wheatley -m SpaceCore --desc "This group was created by GlaDOS, but Wheatley is also an owner"
```

Of course you may not know the usernames you wish to populate your group with. If you need to look someone up you can grab a list of all Igor users by running the command:

```shell
$ igor user show -a
```

If the full name field is populated it can be helpful in identifying usernames for your group creation or modification commands.

## Modifying a Group

Sometimes a group needs to be modified when members join and leave or an owner wants to hand over the group to another Igor user.

`igor group edit NAME [-n NEWNAME] [-o OWN1,OWN2,`…`] `[-w OWN1,OWN2,`…`] [-a USER1,USER2,…]`[-r USER1,USER2,…] [--desc "optional description"]`

### Adding/Removing Users

Use the `**-a**` and the `**-r**` flags to add or remove members from a group you own. Multiple users are declared with comma-delimited lists.

```shell
$ igor group edit Avengers -a hulk,antman
$ igor group edit Avengers -r vision,wanda
```

### Adding/Removing Owners New in v2.1

Use the `**-o**` and `**-w**` flags to add or remove owners from a group. New owners don't have to be current group members. If they are not, they will be automatically added to the group when the command is executed. Likewise, removing an owner from a group simply changes their status to a member. To remove an owner completely, use the `**-r**` flag as if removing a normal member.

```shell
$ igor group edit Avengers -o minime
$ igor group edit Avengers -w ironman
```

It is not possible to remove all owners of a group. Before removing a single owner, a new owner must be added first. Or use both flags in the same edit command.

## Deleting a Group

Deleting a group is simple. The command works on either standard or LDAP-synced groups.

`igor group del NAME`

```shell
$ igor group del Avengers
```

###

If you delete a group that is attached to reservations or distros, then that group will be removed from the access list of those resources. In each case the resource will only be accessible by the owner unless, for example, a distro has more than one group.
