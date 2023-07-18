# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

- `env check` and `dev env check` now checks resource constraints againt all provided resources and resource requirements among all package deployments in the environment, and reports any identified resource constraint violations.

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
