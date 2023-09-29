# Forklift pallet specification

This specification defines Forklift pallets.


## Introduction

This specification's design is heavily inspired by the design of the Go programming language and its module system, and this reference document tries to echo the [reference document for Go modules](https://go.dev/ref/mod) for familiarity. This specification builds upon concepts such as *Forklift packages* and *Forklift repositories* which are introduced in the Forklift [package specification](00-package.md).


## Pallets

A Forklift *pallet* is a fully-specified declarative configuration of the configurations of package deployments which should be active on a Docker host; a pallet is applied to a Docker host by updating the host’s state to match the pallet’s configuration. Only one pallet may be applied to a Docker host at any time. Pallets are how Forklift makes distros easy to customize, extend, and deploy. A Forklift pallet is just a Git repository which may be hosted at a stable location on the internet (e.g. on GitHub), with a special configuration file declaring the pallet. A pallet is identified by a [*pallet path*](#pallet-paths), which is declared in a `forklift-pallet.yml` file at the root of the pallet.

A pallet can also include configurations defined in other pallets. Refer to TODO for more details.

### Pallet paths
An optional *pallet path* is the canonical name for a Forklift pallet, declared with the `path` field in the repository's `forklift-pallet.yml` file. A pallet's path, if it exists, is the prefix for the config files provided by the pallet which can be imported by other pallets.

If defined, a pallet path should communicate both what the pallet does and where to find it. A pallet's path is just the path of the Git repository, if it exists, which contains the pallet. `github.com/PlanktoScope/pallet-standard` is an example of a pallet path.

### Versions
Forklift pallets use the same [versioning system](00-package.md#versions) used for Forklift repositories and packages.

### Configuration file paths
Certain kinds of configuration files and subdirectories in a pallet can be imported by other pallets. For these purposes, the path of a configuration file or subdirectory in the pallet is the pallet path joined with the subdirectory path (relative to the pallet repository's root) of the file or subdirectory.  For example, the pallet `github.com/PlanktoScope/pallet-standard` contains a configuration file at the subdirectory path `requirements/repositories/github.com/PlanktoScope/device-pkgs/forklift-version-lock.yml`. Note that the configuration file path is not necessarily resolveable as a web page URL (so for example <https://github.com/PlanktoScope/pallet-standard/requirements/repositories/github.com/PlanktoScope/device-pkgs/forklift-version-lock.yml> gives a HTTP 404 Not Found error), because the configuration file path is only resolvable in the context of a specific GitHub repository version.


## Package deployments

A typical Forklift pallet defines a set of [*package deployments*](00-package.md#package-deployments-and-constraints); each package deployment specifies a package to deploy as an app, a unique name which should be assigned to the app, and any [package features](00-package.md#package-features) to enable with the package deployment.

Each package deployment should be defined by a file, named with a `.deploy.yml` extension, which is located either in the pallet's `deployments` directory or in a subdirectory of that directory. For any package deployment included in a pallet, the repository which provides that package must be specified with a particular version in a `forklift-version-lock.yml` file at a subdirectory path corresponding to the repository's path, under the `requirements/repositories` directory; for example, a pallet which deploys the `github.com/PlanktoScope/device-pkgs/core/infra/caddy-ingress` package (provided by the `github.com/PlanktoScope/device-pkgs` repository) must also include a file at `requirements/repositories/github.com/PlanktoScope/device-pkgs/forklift-version-lock.yml`.

## Pallet layering

TODO

## Bundles

## Pallet definition

TODO

## Package deployment definition

TODO

## Repository requirement definition

TODO

## Pallet requirement definition

TODO

## Bunde definition

TODO
