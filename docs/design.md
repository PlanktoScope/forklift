# Forklift design

Forklift was created to enable the [PlanktoScope](https://www.planktoscope.org/) project's operating system (based on the Raspberry Pi OS) to gain various benefits of atomic operating systems - traditionally referred to as [immutable operating systems](https://ceur-ws.org/Vol-3386/paper9.pdf) - without requiring the PlanktoScope project to migrate all its legacy software (which, unfortunately, is quite closely-coupled to the Raspberry Pi OS) to an immutable Linux distro. Our resulting approach to modular composition and decentralized distribution is heavily inspired by the simple, transparent, and powerful [design of the Go programming language](https://cacm.acm.org/magazines/2022/5/260357-the-go-programming-language-and-environment/fulltext) and [Go’s Modules system](https://go.dev/doc/modules/developing).

## Values

Forklift's design is guided by the following values for infrastructural software in the PlanktoScope software:

- Autonomy: It must empower people to make their own decisions specific to their needs and contexts, and to exercise full control over their operations of the PlanktoScope, independent of the PlanktoScope project’s longevity.
- Compatibility: It must work well together with legacy systems such as the [PlanktoScope OS](https://docs-edge.planktoscope.community/reference/software/architecture/os/), with the diverse programs which might be managed by it, and with the variety of operational contexts for PlanktoScope deployment. When compatibility is infeasible, incremental migration must be feasible.
- Integrity: It must be trustworthy and reliable in its behavior. It must not corrupt the state of systems built around it. It must be honest to users about what it is doing.
- Productivity: It must help people, teams, and communities to efficiently develop, operate, and maintain their projects; including both the PlanktoScope project and novel extensions and uses for the PlanktoScope. It must be easy to learn, fast enough to use in iterative prototyping, and reliable enough to use in production. It must minimize any complexity and novelty which would distract people from their higher-level goals.
- Thoughtfulness: Its design must be rigorous, deliberate, and considerate of how it will impact people. We must not commit to new features or changes in system behavior until we thoroughly understand their consequences.
- Transparency: Its architecture and behavior must be sufficiently simple and easy to observe, fully explain, fully understand, troubleshoot, and learn from.

## High-level architecture

As a software deployment & configuration system, Forklift consists of the following technical components:

- The `forklift` tool, a single self-contained executable file which provides user interfaces (currently only a command-line interface, though a browser app is also planned) for managing the deployment of apps and system configurations on a Docker host.
- The Forklift specifications, which describe the syntax, structure, and semantics of files and Git repositories used by the `forklift` tool.
- Publicly-hosted Git repositories (e.g. on GitHub) complying with the Forklift specifications.
- Container image registries (e.g. [Docker Hub](https://hub.docker.com/) or the [GitHub Container Registry](https://github.com/features/packages)) complying with the [OCI Distribution Specification](https://github.com/opencontainers/distribution-spec).

Together, the Forklift specifications and the `forklift` tool can be used by project maintainers, contributors, and users to:

1. Specify one or more modules, each of which may consist of a Docker Compose app definition and/or a set of files to export to a directory which can be made available on the operating system (e.g. with overlay mounts and/or bind mounts), and specify how the module will integrate with other modules.
2. Distribute app specifications as *packages* in one or more *repositories*, each of which is a Git repository hosted online by a hosting service such as - but not limited to - GitHub.
3. Specify the exact and complete configuration of a set of packages to be deployed as apps on a computer; the complete configuration for a computer is called a *pallet*, which is a Git repository which may be hosted online.
4. Run continuous integration checks on a pallet - including first-class support for GitHub Actions - to automatically ensure that many kinds of changes to a pallet will not break compatibility or introduce conflicts between deployed apps.
5. Apply a specified version of a pallet to a computer, replacing any previous deployments of apps on that computer.
6. Manually or automatically upgrade a pallet whenever new versions are released.
7. Compose one or more pallets together into a new pallet by *layering*.

## Forklift specifications

Below, we summarize key design decisions in the Forklift specifications as implemented by the `forklift` tool; the full specifications can be found in the [`specs` subdirectory](./specs).

### App packaging and distribution

Forklift uses software containers according to the [OCI specifications](https://github.com/opencontainers) for packaging, distributing, and running containerized software. Thus, Forklift is compatible with the vast collection of open-source applications distributed through container registries such as [Docker Hub](https://hub.docker.com/) and the [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry).

Containers are composed into applications using the [Compose specification](https://github.com/compose-spec/compose-spec) as implemented by [Docker Compose](https://github.com/docker/compose). Each application’s Compose files, configuration files, and additional metadata - including information used for mechanisms to customize and verify app configurations, as discussed below - are organized into a *package*. In a package, a program's executable files (i.e. binaries or scripts, whether or not they are distributed in OCI containers) can be specified together with that program's configuration files, so that all files are bound together in the same lifecycle for deployment, upgrading, and removal. Package metadata is specified in the [YAML format](https://github.com/yaml/yaml-spec), chosen for familiarity with Compose file syntax.

Related packages are organized into a *repository*, which is just a Git repository with a specific metadata file and some packages. Each repository must be published to a Git host such as GitHub, [GitLab](https://gitlab.com/gitlab-org/gitlab), [Codeberg](https://codeberg.org/), or [Gitea](https://github.com/go-gitea/gitea), with a stable URL for identifying and downloading the repository. Repository releases are versioned using Git tags, and all packages in a repository are versioned, released, and updated together.

[fig-pkg-repo]: (add a nested-boxes-and-lines figure with an example directory structure, repository paths, package paths, and a schematic representation of resource interface & feature flags)

Each package can specify a *resource interface*, including the system resources which a deployment of that package provides - such as files, network port listeners, and service APIs - and the resources which a deployment of that package requires. Resource interfaces are used for verifying correctness of app deployment configurations.

Each package can also specify particular files which should be *exported* (i.e. copied) into a path. This enables tools to make certain files (e.g. systemd service definitions) available on the operating system via an overlay filesystem.

To make a package customizable, maintainers can define *feature flags* for it, each with an optional list of provided and required resources (such as exported files) and an optional list of additional Compose files to [merge](https://github.com/compose-spec/compose-spec/blob/master/13-merge.md) into the app. These lists are only active in a package deployment for which the pallet (defined below) has enabled the corresponding feature flag.

### App deployment configuration

Each package can be deployed on a device as one or more running instances of the app provided by the package. Each instance is a *package deployment*, and the configuration of a package deployment includes the enabled feature flags and a name unique to that deployment. Package deployment configurations for packages from one or more Forklift repositories are organized into a *pallet*, which is just a Git repository with a specific metadata file and some configuration files with which a pallet declaratively specifies the complete configuration of every app to be deployed on a device; a pallet is *applied* to a device by updating the device’s state to match the state declared by the pallet. Pallet releases are versioned using Git tags.

[fig-pallet]: (add a figure with an example directory structure, showing how repos are locked and package deployments are specified; add a panel with an example of resource constraint relationships among package deployments, and an example of two alternative HALs for different underlying hardware being swapped underneath a device controller)

A pallet can only be applied when all *resource constraints* are satisfied, namely if every package deployment’s resource requirements are fulfilled and no package deployments provide conflicting resources.

An end-user can initialize a custom pallet from a published pallet, without having to maintain the custom pallet as a fork, by defining a new pallet which references the original base pallet at a particular version and includes all of its configurations; in the new pallet, package deployment configurations can be added to extend or override configurations from the base pallet. Pallets can define their own *feature flags*, each of which specifies a set of configuration files for easy inclusion by other pallets, and other pallets can include multiple base pallets to include specific sets of configuration files from each pallet. This mechanism enables granular composition of configurations among pallets, as the Forklift specification currently prohibits inclusion of conflicting configurations from different base pallets due to the complexity of [diamond dependency conflicts](https://research.swtch.com/vgo-import) which could result from such inclusion.

[fig-customization]: (include a diagram showing how packages across repositories can be recombined in pallets, and how pallets can be overridden and layered can be composed for customization)

## `forklift` tool

The `forklift` tool also provides tool-specific behaviors not described in the Forklift configuration specification:

### App deployment configuration

(talk about resolving branches & tags into commits for repositories & pallets)

(talk about the workspace & cache - caching of repositories & pallets, and switching between pallets, and assembling files into a cached location, and any A/B design of exporting files)

### Configuration reconciliation

(talk about how forklift plans changes and uses Docker Compose to apply the changes; talk about how forklift just requires a Docker host)

(talk about automatic updates, version upgrades/downgrades/pinning; using Docker compose to get GitOps-style reconciliation)

(talk about staged apply, and how in the PlanktoScope OS we need to run forklift apply upon every boot, due to Docker Compose’s design and the PlanktoScope OS's use of forklift to manage various systemd units which only run at boot)

Resource requirement constraints are used in planning the order of changes needed for applying a pallet to a device ([fig-reconciliation-planning]).

[fig-reconciliation-planning]: show an example of resource constraint relationships and the resulting partial order with some example state

### File exporting

(talk about how we export files, and how OSes can use the export directory for bind mounts, overlay mounts, systemd-sysexts/confexts, etc.)

### User interfacing

(talk about the CLI, including the way we organize and name subcommands)
