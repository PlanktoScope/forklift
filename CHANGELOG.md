# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

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
