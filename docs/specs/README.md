# Forklift specifications

This directory provides the specifications for *Forklift*, the package management system used by the PlanktoScope project to manage deployment and configuration of software apps on the computers embedded in PlanktoScopes.

Forklift is a GitOps software configuration management system for uniformly distributing, deploying, and configuring software as [Docker Compose applications](https://docs.docker.com/compose/) on Docker hosts. Its design is heavily inspired by the Go programming language's module system.

Forklift's specifications are organized as follows:

1. For app packaging and distribution: the Forklift [package specification](00-package.md)
2. For app deployment configuration: the Forklift [pallet specification](01-pallet.md)
