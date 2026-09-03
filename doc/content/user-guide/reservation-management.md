# Reservation Management

## Power Commands

Once you reservation is active (i.e., it has a green background on your _igor show_ list), you can issue power commands to its nodes.

```shell
$ igor power {on|off|cycle} {-r RESNAME | -n NODES}
```

_Unless otherwise specified, nodes will immediately begin a**power cycle** operation when the reservation becomes active,_ thus no power command is necessary for start up. Future reservations cannot accept power commands because you don't own the hosts until your reservation is running.

Power cycling will turn node power off and back on again, ensuring that hosts boot for the first time with the image specified by the reservation distro.

Optionally, you can use the `-n` flag to specify _any_ nodes for which you belong to an active reservation. This not only allows you to run power commands on a subset of hosts, but also on hosts spanning multiple reservations. For example, you might have two reservations connected to the same VLAN and you want to reboot a selection of nodes on each one at the same time.

**Examples:**

```shell
$ igor power cycle -r MyCoolRes

$ igor power cycle -n kn[1,5,27]
```

### Did My Power Command Work?

### Be Patient!

If your host is showing up in blue coloration on the CLI or Igorweb, then you know the host at least is powered on. Hosts can take some time to boot. The amount of time varies depending on the cluster node hardware and the size of the OS image being deployed. Igor will report power is _on_ (in the show command) when the boot cycle has progressed far enough to send an affirmative response.

Power commands are routed through Igor to an external IPMI service that tells Igor immediately if the command was successfully submitted. The actual booting of a host can fail for many other reasons. Attempts to power command a node should therefore be followed up with close monitoring to check that the boot completed, sometimes taking as long as a few minutes before the network status changes to show the node is ready to be accessed.

If your node gets stuck in powered-on (blue) mode and never reaches a ready state (white), contact an admin to see if the node is having hardware trouble.

### Group Member Power Commands

If a reservation is attached to a group, then all members of the group will have the ability to use power commands on nodes in the reservation.

## Early Termination

If you are finished with a reservation before it expires, it is recommended that once you have saved off any data or other files you intend to keep that you delete your reservation early. This allows other users to take advantage of nodes that are no longer being used.

Group members also have the ability to delete a reservation.
