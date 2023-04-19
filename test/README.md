# Testing Resources

The files in this folder can be used by developers to test Igor functionality.

The kernel/init files in this folder are dummy text files. If you wish to test files that match the size of a typical k/i pair, use your OS shell to generate random bytes to a file named with the appropriate extension.

Example:

```
head -c 25MB /dev/urandom > test.kernel
``` 

The `igor-clusters.yaml` file is an example configuration for setting up a mock cluster instance.