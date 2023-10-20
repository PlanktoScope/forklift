# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

### Added

- The `forklift-package.yml` files now have an optional `compose-files` field in feature flags to define Compose files which should be merged into the Compose app for every package deployment which enables that feature.
- The `forklift-pallet.yml` file can now optionally specify a README file and a pallet path. When specified, those fields are displayed by the `plt show` and `dev plt show` commands.
- The `.deploy.yml` files now have a `disabled` boolean flag which specifies whether that deployment definition should be ignored (so that it is excluded from `plt plan`, `plt check`, and `plt apply` commands).
- The `show-pkg` subcommand now shows a list of Compose files associated with each package feature flag.
- The `show-depl` subcommand now shows a list of the Compose files used to define the Compose app for the deployment.
- The `show-depl` subcommand now shows more details about the Compose app specified by the deployment.
- The `forklift-repository.yml` and `forklift-pallet.yml` now have a `forklift-version` field which indicates that the repository/pallet was written assuming the semantics of a given version of Forklift, and which sets the minimum version of Forklift required to use the repository/pallet. The Forklift version of a pallet cannot be less than the Forklift version of any repo required by the pallet. The Forklift tool also checks version compatibility - an older version of the Forklift tool is incompatible with repositories/pallets with newer Forklift versions, and the Forklift tool is also sets the minimum Forklift version of any repository/pallet it is compatible with (so for example v0.4.0 of the Forklift tool is incompatible with any repositories/pallets with Forklift version below v0.4.0, due to other breaking changes made for Forklift v0.4.0).

### Changed

- (Breaking change) The `definition-files` field in `forklift-package.yml` files has been renamed to `compose-files`, for unambiguity and future-proofing (so that we can add other definition types, such as for regular files rather than Docker apps).
- (Breaking change) The `forklift-version-lock.yml` file now requires a `type` field which specifies whether the version lock is to be interpreted as a tagged version or as a pseudoversion. The `commit` and `timestamp` fields are now required for all types, instead of being used to determine whether the version lock is for a tagged version or a pseudoversion.
- (Breaking change) The `DefinesApp` method has been removed from `PkgDeplSpec`, since now a Compose App may be defined purely by feature flags.

### Fixed

- Now the `dev plt add-repo` command correctly specifies version-locking information when locking a repo at a tagged version.

## 0.3.1 - 2023-08-24

### Removed

- Removed builds for Darwin and Windows targets, because v0.3.0 couldn't be released due to the CI workflow running out of disk space.

## 0.3.0 - 2023-08-23

### Added

- Now Git repositories providing Forklift packages can be hosted anywhere with a URL, not just on GitHub.

### Changed

- (Breaking change) Now only a single Forklift repository is permitted per Git repository, and the root of the Forklift repository must be the root of the Git repository. This means that the path of the Forklift repository is just the path of the Git repository corresponding to that Forklift repository, and thus the repository definition file must be located at the root of the Git repository.
- (Breaking change) Renamed "Forklift pallet"/"pallet" to "Forklift repository"/"repository". All commands now use `repo` instead of `plt`. This partially reverts a change made in 0.2.0.
- (Breaking change) Renamed "environment"/"env" to "pallet"/"plt". All commands now use `plt` instead of `env`.
- (Breaking change) Changed the name of repository definition files from `forklift-pallet.yml` to `forklift-repository.yml`. This partially reverts a change made in 0.2.0.
- (Breaking change) Changed the name of the repository specification section in the repository definition file from `pallet` to `repository`. This reverts a change made in 0.2.0.
- (Breaking change) Changed the name of the repository requirements directory in environments from `requirements/pallets` to `requirements/repositories`. This partially reverts a change made in 0.2.0.
- (Breaking change) Changed the name of the repository cache directory in the workspace from `cache` to `cache/repositories`. This partially reverts a change made in 0.2.0.
- (Breaking change) Changed the name of the pallet directory in the workspace from `env` to `pallet`.
- (Breaking change) Renamed the `depl` subcommand to `host`.
- (Breaking change) Renamed the `env` subcommand to `plt`, and the `dev env` subcommand to `dev plt`.

## 0.2.2 - 2023-08-11

### Fixed

- Updated the Makefile for the `make release` target to provide the `GITHUB_TOKEN` environment variable to the `goreleaser-cross` Docker container used for that Makefile target.

## 0.2.1 - 2023-08-11

### Fixed

- Updated the Makefile for the `make release` target to also use the `goreleaser-cross` Docker image which is used for the `make build` target.

## 0.2.0 - 2023-08-11

### Added

- Added `depl ls-con` command which either lists the containers for the specified package deployment or (if no package deployment name is specified) lists all containers.

### Changed

- (Breaking change) Forklift now manages Docker Compose applications instead of Docker Stacks, due to Docker Swarm Mode's lack of support for [devices](https://github.com/moby/swarmkit/issues/1244) (important for talking to hardware) and privileged containers (very useful for gradually migrating non-containerized applications into containers). Note that this causes the compiled binaries to approximately double in size, from ~20-25 MB (on linux_amd64) to ~50-60 MB, because of all the unnecessary dependencies pulled in by the `github.comdocker/compose/v2` package; similarly, the compressed archives for the binaries double in size, from ~8 MB to ~17 MB. Hopefully we can return to more reasonable uncompressed binary sizes in the future.
- (Breaking change) Renamed "Pallet repository" to "Forklift pallet"/"pallet". All commands now use `plt` instead of `repo`.
- (Breaking change) Changed the name of pallet definition files from `pallet-repository.yml` to `forklift-pallet.yml`.
- (Breaking change) Changed the name of the pallet specification section in the pallet definition file from `repository` to `pallet`.
- (Breaking change) Changed the name of package definition files from `pallet-package.yml` to `forklift-package.yml`.
- (Breaking change) Changed the way the Docker Compose application is specified in the `deployment` section of a Pallet package definition, from a `definition-file` field for a single string to a `definition-files` field for a list of strings.
- (Breaking change) Changed the name of the pallet requirements directory in environments from `repositories` to `requirements/pallets`.
- (Breaking change) Changed the name of pallet version lock files in environments from `forklift-repo.yml` to `forklift-version-lock.yml`.
- (Breaking change) Changed the name of the pallet cache directory in the workspace from `cache` to `cache/pallets`.

### Removed

- (Breaking change) Removed one-letter abbreviations in all aliases.

## 0.1.10 - 2023-08-03

### Added

- `env plan`, `dev env plan`, `env apply`, and `dev env apply` now return an error (and report the problems) if there are resource conflicts or missing resource dependencies (the same problems which would be reported by `env check` and `dev env check`).
- `cache show-repo`, `env show-repo`, and `dev env show-repo` now print the Pallet repository's readme file, hard-wrapped to a max line length of 100 characters.

### Fixed

- `env plan`, `dev env plan`, `env apply`, and `dev env apply` now account for the resource dependency relationships among package deployments when planning the sequence of changes to make to the Docker host, so that a Docker stack which requires a network provided by another Docker stack won't be deployed before that other Docker stack (since such a deployment would fail).
- The `dev env add-repo` subcommand now makes any directories it needs to make in order to write repository requirement definition files to the appropriate locations.
- File path separators should no longer be obviously incorrect on Windows systems (though they may still be incorrect, since Forklift is not tested on Windows).

## 0.1.9 - 2023-07-29

### Fixed

- Fixed regression from v0.1.8 where the `dev env add-repo` would not properly print out the resolved versions of repo version queries.

## 0.1.8 - 2023-07-29

### Added

- `env check` and `dev env check` now checks resource constraints againt all provided resources and resource requirements among all package deployments in the environment, and reports any identified resource constraint violations.
- `dev env` now allows specifying one or more directories containing Pallet repositories to replace any corresponding cached repositories, using the `--repo` flag (which can be specified repeatedly).
- `env plan` and `dev env plan` now show the changes which will be made by `env apply` and `dev env apply`, respectively.
- The (draft) implementation of the (draft) specification for the Pallets package management system is now available in the `/pkg/pallets` directory of this repository. Note that the specification and implementation will be changed to simplify terminology, so the API will definitely change.
- Pallet package deployment specifications can now be defined in subdirectories under the `deployments` directory of a Forklift environment, instead of having to be defined only in the `deployments` directory.

### Changed

- A major internal refactoring was done. No breaking changes are expected, but breakage is still possible.

### 0.1.7 - 2023-06-01

- `cache ls-img` can now search for locally-downloaded Docker container images matching a provided repository and tag.
- `cache ls-img` now shows the first tag of each Docker container image together with the image repository name, if the tag exists

### Fixed

- Previously, `env apply` and `dev env apply` would always explicitly attempt to pull the image needed for each Docker Stack service, leading them to fail with an error if the computer had no internet connection. Now, they will only explicitly attempt to pull the image if it is not found among the locally-downloaded Docker container images.

## 0.1.6 - 2023-05-31

### Added

- `env cache-img` and `dev env cache-img` commands are now provided to download all Docker container images ahead-of-time before applying the specified environment.
- `cache ls-img` and `cache show-img` commands are now provided to show information about the Docker container images available in the Docker environment.
- `cache rm` now also removes all unused Docker container images which had been locally downloaded.

### Changed

- (Breaking change) Renamed the `env deploy` and `dev env deploy` commands to `env apply` and `dev env apply`, respectively. This is meant to make the mental model for forklift slightly more familiar to people who have used HashiCorp Terraform.
- (Breaking change) Renamed the `env cache` and `dev env cache` commands to `env cache-repo` and `dev env cache-repo`, respectively. This disambiguates the commands for caching Pallets-related data and for caching Docker container images, while allowing them to be run separately (useful on Docker environments where root permissions are required to talk to the Docker daemon).

### Fixed

- When the `env apply` and `dev env apply` commands pull images as part of the process of deploying Docker stacks, they now pull images before creating the stack services with proper image tags, since the Docker API client pulling images without any tags.

## 0.1.5 - 2023-05-26

### Fixed

- Previously, the `dev env add` command did not correctly update the local cache's mirrors of remote Git repositories. Now it should (probably) update them correctly.

## 0.1.4 - 2023-05-24

### Added

- A `dev env add` subcommand to add one or more Pallet repositories to the development environment, and/or to update their configured versions in the development environment

### Changed

- (Breaking change) Renamed the `version` field of `forklift-repo.yml` files to `base-version`.

## 0.1.3 - 2023-05-23

### Added

- An `env info` subcommand to display info about the local environment as a Git repository
- A `dev env` command with subcommands to display info about a development environment (at a user-set path), with the same subcommand structure as the `env` command

### Changed

- Changed the "info" verb in subcommands to "show".
- Standardized abbreviations and expansions of verbs for subcommands (e.g. "d"/"deploy", "ls"/"list", or "s"/"show").
- Standardized abbreviations and expansions of nouns for subcommands (e.g. "d"/"depl"/"deployment"/"deployments" or "r"/"repo"/"repository"/"repositories"). Now the longest alias of a noun-verb subcommand always makes grammatical sense (e.g. "list-repositories" instead of "list-repository", "show-repository" instead of "information-repository"), and the shortest alias of a subcommand always has a one-to-three-letter verb and a one-to-three-letter noun, and the main name of a subcommand is of intermediate length (e.g. "ls-repo", "show-pkg", "show-depl").

### Removed

- Release channels are no longer tracked for each Pallet repository within a Forklift environment, for simplicity.
- (Breaking change) The `forklift-repo-lock.yml` file has been renamed to `forklift-repo.yml`, for simplicity.

### Fixed

- The `depl rm` subcommand now waits until all deleted networks actually disappear before it finishes. This is to help prevent the `env deploy` and `dev env deploy` subcommands from being run while the state of the Docker Swarm is still changing as a result of a previous `depl rm` subcommand.

## 0.1.2 - 2023-04-26

### Fixed

- Set correct file permission flags when making the forklift workspace if it doesn't already exist.

## 0.1.1 - 2023-04-26

### Added

- Handling of stacks which need to be removed as part of the `forklift env deploy` command

### Fixed

- Order of deleting resources (services, networks, secrets, configs) in the `forklift depl rm` command, so that it does not error out when one of the stacks to be deleted provides a resource (e.g. a network) used by other stacks as an external resource.

## 0.1.0 - 2023-04-25

### Added

- Basic commands for cloning and tracking a Pallet environment from a remote Git repository
- A basic command for downloading (into a local cache of Pallet repositories/packages) all Pallet repositories specified by the local Pallet environment
- A basic command with minimal functionality for deploying the local Pallet environment into the local Docker Swarm; this does not fully implement the Pallet specification (notably, all package features are always enabled)
- Basic commands for displaying information about the local Pallet environment and the local cache
- A basic command for deleting the local Pallet environment
- A basic command for deleting the local cache of Pallet repositories/packages
- A basic command for checking what Docker stacks are running in the local Docker Swarm
- A basic command with minimal functionality for deleting all stacks from the local Docker Swarm; this is not fully correct in deleting resources created by Pallet packages (for example, it can't properly delete a network created by one package which is used as an external network by other packages)
