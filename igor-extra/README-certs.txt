This folder is for storing certificates and keys needed by igor applications
for HTTPS connections.

If you'd rather use key/cert files stored elsewhere, then either:
  1) reference them directly in the paths defined in the YAML config files.
  2) symlink the proper files from this folder.
  3) delete this folder and re-create it as a symlink to the correct folder.

In any case, you must fill in the appropriate location in to these files in
their respective YAML configs.

You may use self-signed certificates if you are willing to accept the security
risks. If your Linux distro has openssl installed you can generate the needed
files with this command:

openssl req -newkey rsa:4096 -x509 -sha256 -days 1457 -nodes -out <name>.crt -keyout <name>.key

Examples:
openssl req -newkey rsa:4096 -x509 -sha256 -days 1457 -nodes -out x509-igor.crt -keyout x509-igor.key

(using localhost)
openssl req -newkey rsa:4096 -x509 -sha256 -days 1457 -nodes -out localhost.crt -keyout localhost.key