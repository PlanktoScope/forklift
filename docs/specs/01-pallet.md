# Forklift pallet specification

This specification defines Forklift pallets.

## Introduction

This specification's design is heavily inspired by the design of the Go programming language and its module system, and this reference document tries to echo the [reference document for Go modules](https://go.dev/ref/mod) for familiarity. This specification builds upon concepts such as *Forklift packages* and *Forklift repositories* which are introduced in the Forklift [package specification](00-package.md).

## Pallets

A Forklift *pallet* is a fully-specified declaration of the configurations of [package deployments](./00-package.md#package-deployments-and-constraints) which should be active on the operating system. A pallet can be *applied* to a host operating system, which means that the operating system is modified according to the contents of the pallet. Only one pallet may be applied to the operating system at any time.

A pallet is a Git repository hosted at a stable location on the internet (e.g. on GitHub), with a special `forklift-pallet.yml` configuration file declaring the pallet. The `forklift-pallet.yml` configuration file is expected to exist at the pallet's root directory, which is exactly the root of the Git repository. A pallet is identified by a [*pallet path*](./00-package.md#pallet-paths), which is declared in the `forklift-pallet.yml` file.

Since a pallet is just a Git repository, there will typically be multiple [*versions*](#versions) of a pallet. Usually we will speak of the pallet as the Git repository itself (e.g. "openUC2 has a pallet at `github.com/openUC2/rpi-imswitch-os`"), but sometimes we may speak of a pallet as the set of configurations specified by the files at a particular commit of the Git repository (e.g. "we will now apply the pallet on the `beta` branch instead of the pallet on the `main` branch"); the exact meaning will depend on the context.

Pallets are how Forklift makes operating systems easy to customize, extend, and deploy. A Forklift pallet is just a Git repository which may be hosted at a stable location on the internet (e.g. on GitHub), with a special configuration file declaring the pallet. A pallet is identified by a [*pallet path*](#pallet-paths), which is declared in a `forklift-pallet.yml` file at the root of the pallet.

A pallet can also include configurations defined in other pallets. Refer to TODO for more details.

### Pallet paths
An optional *pallet path* is the canonical name for a Forklift pallet, declared with the `path` field in the repository's `forklift-pallet.yml` file. A pallet's path, if it exists, is the prefix for the config files provided by the pallet which can be imported by other pallets.

If defined, a pallet path should communicate both what the pallet does and where to find it. A pallet's path is just the path of the Git repository, if it exists, which contains the pallet. `github.com/openUC2/rpi-imswitch-os` is an example of a pallet path.

### Versions

A *version* is a Git tag which identifies an immutable snapshot of a pallet and all packages in the pallet; thus, all packages in any single commit of a pallet will have always have identical versions, and all packages in a pallet will always have the same version for a given Git commit. A version may be either a release or a pre-release. Once a Git tag is created, it should not be deleted or changed to a different revision. Versions should be authenticated to ensure safe, repeatable deployments. If a tag is modified, clients may see a security error when downloading it.

Each version starts with the letter `v`, followed by either a semantic version or a calendar version. The [Semantic Versioning 2.0.0 specification](https://semver.org/spec/v2.0.0.html) expains how semantic versions should be formatted, interpreted, and compared; the [Calendar Versioning reference](https://calver.org/) describes a variety of ways that calendar versions may be constructed, but any calendar versioning scheme used must meet the following requirements:

- The calendar version must have three parts (major, minor, and micro), and it may have additional labels for pre-release and build metadata following the semantic versioning specification.
- No version part may be zero-padded (so e.g. `2022.4.0` and `22.4.0` are allowed, while `2022.04.0` and `22.04.0` are not allowed).
- The calendar version must conform to the semantic versioning specifications for precedence, so that versions can be compared and sequentially ordered.

### Configuration file paths
Certain kinds of configuration files and subdirectories in a pallet can be imported by other pallets. For these purposes, the path of a configuration file or subdirectory in the pallet is the pallet path joined with the subdirectory path (relative to the pallet repository's root) of the file or subdirectory.  For example, the pallet `github.com/PlanktoScope/pallet-standard` contains a configuration file at the subdirectory path `requirements/repositories/github.com/PlanktoScope/device-pkgs/forklift-version-lock.yml`. Note that the configuration file path is not necessarily resolveable as a web page URL (so for example <https://github.com/PlanktoScope/pallet-standard/requirements/repositories/github.com/PlanktoScope/device-pkgs/forklift-version-lock.yml> gives a HTTP 404 Not Found error), because the configuration file path is only resolvable in the context of a specific GitHub repository version.

## Package deployments

A typical Forklift pallet defines a set of [*package deployments*](00-package.md#package-deployments-and-constraints); each package deployment specifies a package to deploy as an app, a unique name which should be assigned to the app, and any [package features](00-package.md#package-features) to enable with the package deployment.

Each package deployment should be defined by a file, named with a `.deploy.yml` extension, which is located either in the pallet's `deployments` directory or in a subdirectory of that directory. For any package deployment included in a pallet, the repository which provides that package must be specified with a particular version in a `forklift-version-lock.yml` file at a subdirectory path corresponding to the repository's path, under the `requirements/repositories` directory; for example, a pallet which deploys the `github.com/PlanktoScope/device-pkgs/core/infra/caddy-ingress` package (provided by the `github.com/PlanktoScope/device-pkgs` repository) must also include a file at `requirements/repositories/github.com/PlanktoScope/device-pkgs/forklift-version-lock.yml`.

## Pallet layering

TODO

## Bundles

## Pallet definition

The definition of a pallet is stored in a YAML file named `forklift-pallet.yml` in the pallet's root directory. Here is an example of a `forklift-pallet.yml` file:

```yaml
forklift-version: v0.4.0

pallet:
  path: github.com/PlanktoScope/device-pkgs
  description: Packages for the PlanktoScope software distribution
  readme-file: README.md
```

### `forklift-version` field

This field of the `forklift-pallet.yml` file declares that the pallet was written assuming the semantics of a given version of Forklift.

- This field is required.

- The version must be a valid version of the Forklift tool or the Forklift specification.

- The version sets the minimum version of the Forklift tool required to use the pallet. The Forklift tool refuses to use repositories declaring newer Forklift versions (or excessively old Forklift versions) for any operations beyond printing information.

- Example:
  
  ```yaml
  forklift-version: v0.4.0
  ```

All other fields in the pallet metadata file are under a `pallet` section.

### `pallet` section

This section of the `forklift-pallet.yml` file contains some basic metadata to help describe and identify the pallet. Here is an example of a `pallet` section:

```yaml
pallet:
  path: github.com/PlanktoScope/device-pkgs
  description: Packages for the PlanktoScope software distribution
  readme-file: README.md
```

#### `path` field

This field of the `pallet` section is the pallet path.

- This field is required.

- Example:
  
  ```yaml
  path: github.com/PlanktoScope/device-pkgs
  ```

#### `description` field

This field of the `pallet` section is a short (one-sentence) description of the pallet to be shown to users.

- This field is required.

- Example:
  
  ```yaml
  description: Packages for the PlanktoScope software distribution
  ```

#### `readme-file` field

This field of the `pallet` section is the filename of a readme file to be shown to users.

- This field is required.

- The file must be located in the same directory as the `forklift-repository.yml` file.

- The file must be a text file.

- It is recommended for this file to be named `README.md` and to be formatted in [GitHub-flavored Markdown](https://github.github.com/gfm/).

- Example:
  
  ```yaml
  readme-file: README.md
  ```


## Package deployment definition

TODO

## Pallet requirement definition

TODO

## Bundle definition

TODO

# Versioning with constraints and features

Usually, the following changes to a package are backwards-incompatible, in which case they will require incrementing the major component of the semantic version of the repository providing that package, if the repository follows semantic versioning and the major component of the semantic version was already nonzero:

- Changing the name to use for deploying the package
- Making changes which may introduce conflicts between provided resources, for certain combinations of package deployments:
   - In the host specification or the deployment specification:
      - Adding a provided resource
      - Modifying the identification criteria of a provided resource
   - In any optional feature:
      - Adding a new provided resource
      - Modifying the identification criteria of a provided resource
- Making changes which may make dependencies between provided and required resources unresolvable, for certain combinations of package deployments:
   - In the host specification or the deployment specification:
      - Removing a provided resource
      - Modifying the identification criteria of a provided resource
   - In the deployment specification:
      - Adding a resource requirement
      - Modifying the identification criteria of a resource requirement
   - In any optional feature:
      - Adding a new resource requirement
      - Removing a provided resource
      - Modifying the identification criteria of a provided resource
      - Modifying the identification criteria of a resource requirement
- Making changes which may make a package deployment declaration invalid:
   - In the list of optional features offered by the package:
      - Removing any feature
      - Renaming any feature
- Making a backwards-incompatible change to any external technical interfaces (protocols, APIs, schemas, data formats, etc.) provided or required by that package: this may break compatibility with other deployed packages which interact with those interfaces. Backwards-incompatible changes include:
   - Adding a new requirement as part of the interface
   - Removing a previously-provided functionality from the interface
- Making a change to any user interfaces provided or by that package which would probably break users' existing mental models of how to use the interface.

The following changes to a package are usually backwards-compatible, in which case they would only require incrementing the minor component of the semantic version of the repository providing that package, if the repository follows semantic versioning:

- Adding a new optional feature
- Removing a resource requirement from any optional feature
- Making a backwards-compatible change to any external technical or user interfaces provided or required by that package. Backwards-compatible changes in an interface include:
   - Removing a requirement from the interface
   - Adding new optional functionality to the interface

It is the reponsibility of the package maintainer to document the package's external interfaces, to increment the relevant components of a repository's semantic or calendar version as needed, and to help users migrate smoothly across version upgrades and downgrades.