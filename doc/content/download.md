# Obtaining Igor

Igor's code is hosted on GitHub. You can also view the site to see issues, pull requests and the latest check-ins.

[Igor on GitHib](https://github.com/sandia-minimega/igor2)

## Building from source

Igor has been built and tested on Debian/Ubuntu and RedHat Enterprise distributions. _It is not intended to be built or installed on a Windows system._

To build from source you will need [Go](http://golang.org/) (version 1.23 or greater) and [NodeJS](https://nodejs.org/) (version 22 or greater).

Having installed these onto your build environment, grab the Igor source code:

```shell
$ git clone git@github.com:sandia-minimega/igor2.git
$ cd igor2
```

Next, check out the latest release. If you wish to build from the most recent commit that is not officially tagged with a version number, skip this command.

```shell
$ git checkout <version-tag>
```

Finally, you can run the included _build-igor_ script with the help flag to see instructions and options.

```shell
$ ./build-igor -h
```

You can build any number of Igor's parts (server, web-app, and/or CLI) with a single command. All executables and relevant files will be placed in the _build/_ sub-directory. Use the **-z** flag to output a tar archive which can be run in conjunction with the _install-igor_ script on your production nodes to get Igor apps and config files installed in the right places.

See the Administration Guide for more detailed information on hardware requirements and installation.

## Binary distribution

Igor currently doesn't offer pre-built binaries, but we are looking into this as part of a future update.
