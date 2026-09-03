# Operational Requirements

## Deployment Environment

### Cluster

Igor assumes you have a cluster with uniformly-named or aliased nodes (e.g. "kn1", "kn2", and so on) with the head node running RedHat or Debian/Ubuntu. We assume the cluster nodes are configured for netboot using PXE and TFTP with the head node handing out DHCP addresses.

### Hardware

Igor utilizes network switch communication to gather information about the status of cluster nodes. **_At this time only Arista network switches are supported for enabling VLAN segmentation._** We hope to support more alternatives in the future.

### Additional Software

The following OS packages must be installed on the node hosting igor-server (check package name equivalencies for yum or apt):

  * tftp, tftp-server
  * dhcp-server, dhcp-libs, dhcp-common
  * freeipmi, ipmitool, or some other utility that can send power commands to nodes through a command-line interface
  * nmap (no longer needed in v2.3+)
  * sqlite 3.x+

The installation script that comes with Igor will attempt to install/update nmap and sqlite since those applications are used directly by the server application. It is beyond the scope of this document to install and configure the other packages. It is assumed they have already been set up according to the needs of the cluster environment.

The Igor CLI and web-app binaries do not require additional OS-provided packages to run.

### The Head Node

Igor is designed for distributed deployment in order to minimize security risks depending on the policies of the organization's cluster network. While it is simple to deploy all three executables onto the cluster head node and give users access rights to log into that node, many organizations will want to restrict normal users from head node access.

Our recommended configuration for the head node is to install at least the server and CLI client on it, allowing logged in admins to be able to use the application. The head node must also have access to enough space to hold however many OS images that Igor users are expected to deploy.

Igor-web can also be hosted here if desired, simplifying setup by co-locating it on the same node as igor-server.

### User Login Node(s)

Our recommended configuration for Igor user access is one or more login or edge nodes that has cluster network access and space for users to do whatever they need. Any such node should have the Igor CLI installed on it at minimum.

This is also an ideal place to host Igor-web. Depending on how you want to scale out your user access points, Igor-web can be installed on as many user login nodes as desired. Each web-app instance will need to be referenced in the igor-server config.

### Other User Access Points

Finally the team should decide if they will allow users to access Igor from other machines, possibly users' own desktops or laptops. The Igor CLI client can be run on any Linux instance as long as it can reach the server.

Limiting access to some types of connections -- for example, only allowing access on a company or campus network -- is best handled by network firewall policies.

## Security Certificates

Igor's server, web and CLI communicate using HTTPS. With the exception of the callback port (see below), there is no unencrypted option. It is highly recommended to purchase certificates for igor-server and igor-web or use existing wildcard certificates that cover all machines in the cluster domain. The folder _/etc/igor/certs_ is provided as a place to store them. See the included README in that folder after installation for more information.

The callback port can run either over HTTP or HTTPS based on a setting in the igor-server configuration file. If running over HTTPS (default option), the same certs are used as igor-server.

### Self-signed certificates

Self-signed certificates can be used if acceptable. To generate a certificate, use the following example command on a Linux instance with OpenSSL installed. When prompted make sure the CN parameter matches the hostname you have configured.

`openssl req -newkey rsa:4096 -x509 -sha256 -days <number> -nodes -out <hostname>.crt -keyout <hostname>.key`

Due to the CORS restrictions implemented on web browsers, you will need to register the igor-server self-signed certificate as trusted on the same node where igor-web is running. Consult your OS documentation for how to do this. If the certificate is not trusted, igor-web will load the login page but fail when login is attempted. As a temporary workaround you can point your browser at the igor-server address and accept the unsigned certificate before attempting to access the web app. _Be aware that some browser and platform combinations having been removing the ability to accept unsigned certificates._

`https://{server-host:port}/igor`

### Using a CLI Client Certificate

The Igor CLI doesn't require a certificate to operate, but sysadmins can require one in order to restrict which clients can access the server. In practice this shouldn't be necessary since firewall rules and required user authentication will be sufficient, however the option exists if desired.
