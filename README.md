<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://github.com/PlanktoScope/forklift/assets/180370/a4317864-a75e-4717-99e5-402d07e109fb">
  <source media="(prefers-color-scheme: light)" srcset="https://github.com/PlanktoScope/forklift/assets/180370/c915ca65-aabf-4721-894e-81f002100004">
  <img src="https://github.com/PlanktoScope/forklift/assets/180370/c915ca65-aabf-4721-894e-81f002100004" alt="Forklift logo" height="60">
</picture>

<hr>

A simpler, easier, and safer way to manage application/system services on embedded Linux systems.

Note: this is still an experimental prototype and is not yet ready for general use.

## Introduction

Forklift is a software deployment and configuration system providing a simpler, easier, and safer mechanism for updating, reconfiguring, recomposing, and extending browser apps, network services, system services, and operating system configuration files on single-computer systems (such as a Raspberry Pi or a laptop), especially computers embedded in open-source scientific instruments. While Forklift can also be used in other contexts, it makes tradeoffs specific to the ways in which many open-source scientific instruments need to be deployed and operated (e.g. intermittent internet access, independent administration by individual people, decentralized management & customization).

For end-users operating open-source instruments with application services (e.g. network APIs or browser-based interfaces) and/or system services (for e.g. data backups/transfer, hardware support, computer networking, monitoring, etc.), Forklift aims to provide an experience for installing and uninstalling software similar what is achieved by app stores for mobile phones - but with more user control. Forklift also simplifies the process of keeping software up-to-date and the process of rolling software back to older versions if needed; this reduces the need to (for example) re-flash a Raspberry Pi's SD card with a new OS image just to update the application software running on the instrument while ensuring the validity of the resulting state of the system.

For open-hardware project developers, Forklift enables Linux-based devices and [appliances](https://en.wikipedia.org/wiki/Computer_appliance) to be retrofitted and extended with an open ecosystem of containerized software - device-specific or general-purpose, project-maintained or third-party. Forklift also provides an incremental path for migrating project-specific application/system services and OS configs into management by Forklift so that they can be configured, distributed, installed, and replaced by users just like any other app managed by Forklift - i.e. with version control and easy upgrades/rollbacks. The [PlanktoScope](https://www.planktoscope.org/), an open-source microscope for quantitative imaging of plankton, uses Forklift as foundational infrastructure for software releases, deployment, and extensibility in the [PlanktoScope OS](https://docs-edge.planktoscope.community/reference/software/architecture/os/), a hardware-specific operating system based on the Raspberry Pi OS; and Forklift was designed specifically to solve the software maintenance and operations challenges experienced in the PlanktoScope project.

For indie software developers and sysadmins familiar with DevOps and cloud-native patterns, Forklift is just a GitOps-inspired system which is small and simple enough to work beyond the cloud - using Docker Compose to avoid the architectural complexity, operational overhead, and memory usage of even minimal Kubernetes distributions like k0s; and bundling app deployment with the deployment of system files, executables, and systemd units from configuration files version-controlled in Git repositories. Thus, Forklift allows hassle-free management of software configurations on one or more machines with only occasional internet access (or, in the future, no internet access at all!) and no specialized ops or platform team.

For information about the design of Forklift, please refer to the [design document](./docs/design.md).


### Project Governance

Currently, design and development of Forklift prioritizes the needs of the PlanktoScope community and the PlanktoScope project's [values for its infrastructural software](./docs/design.md#values). Thus, for now decisions will be made by the PlanktoScope software's lead maintainer (currently [@ethanjli](https://github.com/ethanjli)) as a "benevolent dictator"/"mad scientist" in consultation with the PlanktoScope community in online meetings and discussion channels open to the entire community. This will remain the governance model of Forklift while it's still an experimental tool and still only used for the standard/default version of the PlanktoScope's operating system, in order to ensure that Forklift develops in a cohesive way consistent with the values mentioned above and with intended use of Forklift for the PlanktoScope community. Once Forklift starts being used for delivering/maintaining variants of the PlanktoScope's operating system, for integration of third-party apps from the PlanktoScope community, or for software configuration by ordinary users, then governance of the [github.com/PlanktoScope/forklift](https://github.com/PlanktoScope/forklift) repository will transition from benevolent dictatorship into the PlanktoScope project's [consensus-based proposals process](https://github.com/PlanktoScope/proposals). In the meantime, we encourage anyone who is interested in using/adapting Forklift to fork this repository for experimentation and/or to [create new discussion posts in this repository](https://github.com/PlanktoScope/forklift/discussions/new/choose), though we can't make any guarantees about the stability of any APIs or about our capacity to address any external code contributions or feature requests.

If other projects beyond the PlanktoScope community decide to use Forklift as part of their software delivery/deployment infrastructure, we can talk about expanding governance of Forklift beyond the PlanktoScope community - feel free to start a discussion in this repository's GitHub Discussions forum.

## Usage

### Download/install `forklift`

First, you will need to download the `forklift` tool, which is available as a single self-contained executable file. You should visit this repository's [releases page](https://github.com/PlanktoScope/forklift/releases/latest) and download an archive file for your platform and CPU architecture; for example, on a Raspberry Pi 4, you should download the archive named `forklift_{version number}_linux_arm.tar.gz` (where the version number should be substituted). You can extract the `forklift` binary from the archive using a command like:
```
tar -xzf forklift_{version number}_{os}_{cpu architecture}.tar.gz forklift
```

Then you may need to move the `forklift` binary into a directory in your system path, or you can just run the `forklift` binary in your current directory (in which case you should replace `forklift` with `./forklift` in the example commands listed below), or you can just run the `forklift` binary by its absolute/relative path (in which case you should replace `forklift` with the absolute/relative path of the binary in the example commands listed below).

### Deploy a published pallet

To deploy a particular version of a published pallet to your computer, you will need to clone a pallet and stage it to be applied, and then you will need to apply the staged pallet. Pallets are identified by the path of their Git repository and a version query (which can be a Git branch name, a Git tag name, or an abbreviated or full Git commit hash). For example, the most recent commit on the `main` branch of the [`github.com/ethanjli/pallet-example-minimal`](https://github.com/ethanjli/pallet-example-minimal) can be identified as `github.com/ethanjli/pallet-example-minimal@main` - this is what we use in the example commands in this section.

If you are running Docker in [rootless mode](https://docs.docker.com/engine/security/rootless/) or your user is in [the `docker` group](https://docs.docker.com/engine/install/linux-postinstall/#manage-docker-as-a-non-root-user)):

- If you want to apply the pallet immediately, you can run `forklift pallet switch --apply` with your specified pallet. For example:

  ```
  forklift pallet switch --apply github.com/ethanjli/pallet-example-minimal@main
  ```

- If you want to apply the pallet later, you can first stage the pallet, and then later apply the staged pallet using a separate `forklift stage apply` command. For example:

  ```
  # Run now:
  forklift pallet switch github.com/ethanjli/pallet-example-minimal@main
  # Run when you want to apply the pallet:
  forklift stage apply
  ```

If you aren't running Docker in rootless mode and your user isn't in a `docker` group, we recommend a slightly different set of commands:

- If you want to apply the pallet immediately, you can run `forklift pallet switch --no-cache` as a regular user, and then run `forklift stage apply` as root; the `--no-cache` flag prevents `forklift pallet switch` from attempting to make Docker pre-download all container images required by the pallet, as doing so would require root permissions for Forklift to talk to Docker. For example:

  ```
  forklift pallet switch --no-cache-img github.com/ethanjli/pallet-example-minimal@main
  sudo -E forklift stage apply
  ```

- If you want to apply the pallet later, you can first stage the pallet and pre-download all container images required by the pallet, and then later apply the staged pallet using a separate `forklift stage apply` command (which you can then run even when you don't have internet access). For example:

  ```
  # Run now:
  forklift pallet switch --no-cache-img github.com/ethanjli/pallet-example-minimal@main
  sudo -E forklift stage cache-img
  # Run when you want to apply the pallet:
  sudo -E forklift stage apply
  ```

Note: in the above commands, you can replace `forklift pallet` with `forklift plt` if you want to type fewer characters when running those commands.

### Work on a development pallet

First, you will need to make/download a pallet somewhere on your local file system. For example, you can use `git` to clone the latest unstable version (on the `main` branch) of the [`github.com/ethanjli/pallet-example-minimal`](https://github.com/ethanjli/pallet-example-minimal) pallet using the command:

```
git clone https://github.com/ethanjli/pallet-example-minimal
```

Then you will need to download/install the `forklift` tool (see instructions in the ["Download/install forklift"](#downloadinstall-forklift) section above). Once you have `forklift`, you can run commands using the `dev plt` subcommand; if `forklift` is in your system path, you can simply run commands within the directory containing your development pallet, or any subdirectory of it. For example, if your development pallet is at `/home/pi/dev/pallet-example-minimal`, you can run the following commands to see some information about your development pallet:

```
cd /home/pi/dev/pallet-example-minimal
forklift dev plt show
```

You can also run the command from anywhere else on your filesystem by specifying the path of your development pallet. For example, if your forklift binary is in `/home/pi`, you can run any the following sets of commands to see the same information about your development pallet:

```
cd /home/pi/
./forklift dev --cwd ./dev/pallet-example-minimal plt show
```

```
cd /etc/
/home/pi/forklift dev --cwd /home/pi/dev/pallet-example-minimal plt show
```

You can also use the `forklift dev plt require-repo` command to require additional Forklift repositories for use in your development pallet, and/or to change the versions of Forklift repositories already required by your development pallet.

You can also run commands like `forklift dev plt cache-all` and `forklift dev plt stage --no-cache-img` (with appropriate values in the `--cwd` flag if necessary) to download the Forklift repositories specified by your development pallet into your local cache and stage your development pallet to be applied with `sudo -E forklift stage apply`. This is useful if, for example, you want to make some experimental changes to your development pallet and test them on your local machine before committing and pushing those changes onto GitHub.

Finally, you can run the `forklift dev plt check` command to check the pallet for any problems, such as violations of resource constraints between package deployments.

You can also override cached repos with repos from your filesystem by specifying one or more directories containing one or more repos; then the repos in those directories will be used instead of the respective repos from the cache, regardless of repo version. For example:

```
cd /home/pi/
/home/pi/forklift dev --cwd /home/pi/dev/pallet-example-minimal plt --repos /home/pi/forklift/dev/device-pkgs check
```

## Similar projects

The following projects solve related problems with containers for application software, though they make different trade-offs compared to Forklift:

- [poco](https://github.com/shiwaforce/poco) enables Git-based management of Docker Compose projects and collections (*catalogs*) of projects and repositories and provides some similar functionalities to forklift
- [Terraform](https://registry.terraform.io/providers/kreuzwerker/docker/latest/docs) (an early inspiration for this project) has a Docker Provider which enables declarative management of Docker hosts and Docker Swarms from a Terraform configuration
- [swarm-pack](https://github.com/swarm-pack/swarm-pack) (an early inspiration for this project) uses collections of packages from user-specified Git repositories and enables templated configuration of Docker Compose files, with imperative deployments of packages to a Docker Swarm
- [SwarmManagement](https://github.com/hansehe/SwarmManagement) uses a single YAML file for declarative configuration of an entire Docker Swarm
- Podman [Quadlets](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html) enable management of containers, volumes, and networks using declarative systemd units
- [FetchIt](https://github.com/containers/fetchit) enables Git-based management of containers in Podman
- Projects developing [GitOps](https://www.gitops.tech/) tools such as ArgoCD, Flux, etc., store container environment configurations as Git repositories but are generally designed for Kubernetes

The following projects solve related problems in the base OS, though they make different trade-offs compared to Forklift (especially because of the PlanktoScope project's legacy software):

- [systemd-sysext and systemd-confext](https://www.freedesktop.org/software/systemd/man/latest/systemd-sysext.html) provide a more structured/constrained way (compared to Forklift) to atomically overlay system files onto the base OS; however, Forklift can also be used as a way to deploy sysexts/confexts onto an OS (see [this demo](https://github.com/ethanjli/ublue-forklift-sysext-demo?tab=readme-ov-file#explanation))
- systemd's [Portable Services](https://systemd.io/PORTABLE_SERVICES/) pattern and `portablectl` tool provide a more structured/constrained/sandboxed way (compared to Forklift) to atomically add system services
- [ostree](https://ostreedev.github.io/ostree/) enables atomic updates of the base OS, but [it is not supported by Raspberry Pi OS](https://github.com/ostreedev/ostree/issues/2223)
- The [bootc](https://containers.github.io/bootc/) project enables the entire operating system to be delivered as a bootable OCI container image, but currently it relies on bootupd, which [currently only works on RPM-based distros](https://github.com/coreos/bootupd/issues/468)
- [gokrazy](https://gokrazy.org/) enables atomic deployment of Go programs (and also of software containers!), but it has a very different architecture compared to traditional Linux distros

Other related OS-level projects can be found at [github.com/castrojo/awesome-immutable](https://github.com/castrojo/awesome-immutable).

## Licensing

Except where otherwise indicated, source code provided here is covered by the following information:

Copyright Ethan Li and PlanktoScope project contributors

SPDX-License-Identifier: Apache-2.0 OR BlueOak-1.0.0

You can use the source code provided here either under the [Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0) or under the [Blue Oak Model License 1.0.0](https://blueoakcouncil.org/license/1.0.0); you get to decide. We are making the software available under the Apache license because it's [OSI-approved](https://writing.kemitchell.com/2019/05/05/Rely-on-OSI.html), but we like the Blue Oak Model License more because it's easier to read and understand.
