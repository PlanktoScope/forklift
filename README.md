# forklift
Experimental prototype of tooling to manage local installations of Pallet packages

## Introduction

A forklift is a tool for moving pallets of packages:

![Photograph of a forklift being used to move stacks of cardboard boxes on a pallet](https://images.rawpixel.com/image_1000/cHJpdmF0ZS9sci9pbWFnZXMvd2Vic2l0ZS8yMDIyLTExL2ZsNDk3OTgwOTA2MjctaW1hZ2UuanBn.jpg)

This repository provides `forklift`, a Git-based command-line tool for installing, uninstalling, upgrading, and downgrading [Pallet](https://github.com/PlanktoScope/pallets) packages on a Docker Swarm Mode environment.

## Usage

First, you will need to download forklift, which is available as a single self-contained executable file. You should visit this repository's [releases page](https://github.com/PlanktoScope/forklift/releases/latest) and download an archive file for your platform and CPU architecture; for example, on a Raspberry Pi 4, you should download the archive named `forklift_{version number}_linux_arm.tar.gz` (where the version number should be substituted). You can extract the forklift binary from the archive using a command like:
```
tar -xzf forklift_{version number}_{os}_{cpu architecture}.tar.gz forklift
```

Then you may need to move the forklift binary into a directory in your system path, or you can just run the forklift binary in your current directory (in which case you should replace `forklift` with `./forklift` in the commands listed below).

Once you have forklift, you will need to clone a Pallet environment to your local environment. For example, you can clone the latest unstable version (on the `edge` branch) of the [`github.com/PlanktoScope/pallets-env`](https://github.com/PlanktoScope/pallets-env) environment using the command:
```
forklift env clone github.com/PlanktoScope/pallets-env@edge
```

Then you will need to download the Pallet repositories specified by your local environment into your local cache, so that you can deploy packages provided by those repositories. You can download the necessary repositories using the command:
```
forklift env cache
```

Then you will need to apply the package deployments as configured by your local environment, into your Docker Swarm. You can apply the deployments using the command:
```
forklift env deploy
```

## Licensing

Except where otherwise indicated, source code provided here is covered by the following information:

Copyright Ethan Li and PlanktoScope project contributors

SPDX-License-Identifier: Apache-2.0 OR BlueOak-1.0.0

You can use the source code provided here either under the [Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0) or under the [Blue Oak Model License 1.0.0](https://blueoakcouncil.org/license/1.0.0); you get to decide. We are making the software available under the Apache license because it's [OSI-approved](https://writing.kemitchell.com/2019/05/05/Rely-on-OSI.html), but we like the Blue Oak Model License more because it's easier to read and understand.
