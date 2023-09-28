# forklift

A GitOps-inspired software distribution and configuration system for operating customizable scientific instruments at scale.

Note: this is still an experimental prototype and is not yet ready for general use.

## Introduction

An industrial forklift is a tool for moving pallets of packages:

![Photograph of a forklift being used to move stacks of cardboard boxes on a pallet](https://images.rawpixel.com/image_1000/cHJpdmF0ZS9sci9pbWFnZXMvd2Vic2l0ZS8yMDIyLTExL2ZsNDk3OTgwOTA2MjctaW1hZ2UuanBn.jpg)

This repository provides a [reference specification](reference.md) of _Forklift_, a declarative software package management system for uniformly distributing, deploying, and configuring software as Docker Compose applications, mean to simplify deployment of software distributions on PlanktoScopes and other networked scientific instruments which may be operated as cattle (i.e. interchangeable units from a large, uniform pool of machines) or pets (i.e. with a custom unique configuration per machine). This repository also provides `forklift`, a Git-based command-line tool for applying, removing, upgrading, and downgrading Forklift package deployments on a Docker host.

## Usage

### Download/install forklift

First, you will need to download forklift, which is available as a single self-contained executable file. You should visit this repository's [releases page](https://github.com/PlanktoScope/forklift/releases/latest) and download an archive file for your platform and CPU architecture; for example, on a Raspberry Pi 4, you should download the archive named `forklift_{version number}_linux_arm.tar.gz` (where the version number should be substituted). You can extract the forklift binary from the archive using a command like:
```
tar -xzf forklift_{version number}_{os}_{cpu architecture}.tar.gz forklift
```

Then you may need to move the forklift binary into a directory in your system path, or you can just run the forklift binary in your current directory (in which case you should replace `forklift` with `./forklift` in the commands listed below), or you can just run the forklift binary by its absolute/relative path (in which case you should replace `forklift` with the absolute/relative path of the binary in the commands listed below).

### Deploy a published environment

Once you have forklift, you will need to clone a Pallet environment to your local environment. For example, you can clone the latest unstable version (on the `edge` branch) of the [`github.com/PlanktoScope/pallets-env`](https://github.com/PlanktoScope/pallets-env) environment using the command:
```
forklift env clone github.com/PlanktoScope/pallets-env@edge
```

Then you will need to download the Pallet repositories specified by your local environment into your local cache, so that you can deploy packages provided by those repositories. You can download the necessary repositories using the command:
```
forklift env cache-repo
```

Then you will need to apply the package deployments as configured by your local environment, into your Docker Swarm. You can apply the deployments using the command (note that you need `sudo -E` unless you are running the Docker in [rootless mode](https://docs.docker.com/engine/security/rootless/) or your user is in [the `docker` group](https://docs.docker.com/engine/install/linux-postinstall/#manage-docker-as-a-non-root-user)):
```
sudo -E forklift env apply
```

### Work on a development environment

First, you will need to make/download a Pallet environment somewhere on your local file system. For example, you can clone the latest unstable version (on the `edge` branch) of the [`github.com/PlanktoScope/pallets-env`](https://github.com/PlanktoScope/pallets-env) environment using the command:

```
git clone https://github.com/PlanktoScope/pallets-env
```

Then you will need to download/install forklift (see instructions in the "Download/install forklift" section above). Once you have forklift, you can run commands using the `dev env` subcommand; if forklift is in your system path, you can simply run commands within the directory containing your development environment, or any subdirectory of it. For example, if your development environment is at `/home/pi/dev/pallets-env`, you can run the following commands to see some information about your development environment:

```
cd /home/pi/dev/pallets-env
forklift dev env show
```

You can also run the command from anywhere else on your filesystem by specifying the path of your development environment. For example, if your forklift binary is in `/home/pi`, you can run any the following sets of commands to see the same information about your development environment:

```
cd /home/pi/
./forklift dev --cwd ./dev/pallets-env env show
```

```
cd /etc/
/home/pi/forklift dev --cwd /home/pi/dev/pallets-env env show
```

You can also use the `forklift dev env add-repo` command to add additional Pallet repositories to your development environment, and to change the versions of Pallet repositories already added to your development environment.

You can also run commands like `forklift dev env cache-repo` and `sudo -E forklift dev env apply` (with appropriate values in the `--cwd` flag if necessary) to download the Pallet repositories specified by your development environment into your local cache and deploy the packages provided by those repositories according to the configuration in your development environment. This is useful if, for example, you want to make some experimental changes to your development environment and test them on your local machine before committing and pushing those changes onto GitHub.

Finally, you can run the `forklift dev env check` command to check the environment for any problems, such as resource constraint violations.

You can also override cached repos with repos from your filesystem by specifying one or more directories containing one or more repos; then the repos in those directories will be used instead of the respective repos from the cache, regardless of repo version. For example:

```
cd /home/pi/
/home/pi/forklift dev --cwd /home/pi/dev/pallets-env env --repo /home/pi/forklift/dev/pallets check
```

## Similar projects

The following projects solve related problems with containers, though they make different trade-offs compared to Forklift and Pallets:

- poco enables Git-based management of Docker Compose projects and collections (*catalogs*) of projects and repositories and provides some similar functionalities to forklift: https://github.com/shiwaforce/poco
- Terraform (an inspiration for this project) has a Docker Provider which enables declarative management of Docker hosts and Docker Swarms from a Terraform configuration: https://registry.terraform.io/providers/kreuzwerker/docker/latest/docs
- swarm-pack (an inspiration for this project) uses collections of packages from user-specified Git repositories and enables templated configuration of Docker Compose files, with imperative deployments of packages: https://github.com/swarm-pack/swarm-pack
- SwarmManagement uses a single YAML file for declarative configuration of an entire Docker Swarm: https://github.com/hansehe/SwarmManagement
- Podman Quadlets enable management of containers, volumes, and networks using declarative systemd units: https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html
- FetchIt enables Git-based management of containers in Podman: https://github.com/containers/fetchit
- Projects developing GitOps tools such as ArgoCD, Flux, etc., store container environment configurations as Git repositories but are generally designed for Kubernetes: https://www.gitops.tech/

## Licensing

Except where otherwise indicated, source code provided here is covered by the following information:

Copyright Ethan Li and PlanktoScope project contributors

SPDX-License-Identifier: Apache-2.0 OR BlueOak-1.0.0

You can use the source code provided here either under the [Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0) or under the [Blue Oak Model License 1.0.0](https://blueoakcouncil.org/license/1.0.0); you get to decide. We are making the software available under the Apache license because it's [OSI-approved](https://writing.kemitchell.com/2019/05/05/Rely-on-OSI.html), but we like the Blue Oak Model License more because it's easier to read and understand.
