# Forklift pallet specification

## Introduction

Pallets are how Forklift manages packages and how Forklift makes operating systems easy to customize, extend, and deploy. This document is a detailed reference manual for Forklift pallets.

This specification's design is heavily inspired by the design of the Go programming language and its "module" system, and this reference document tries to echo the [reference document for Go modules](https://go.dev/ref/mod) for familiarity. This specification builds upon concepts such as *Forklift packages* and *Forklift package deployments* described in the Forklift [package specification](00-package.md).

## Pallets

A Forklift *pallet* is a collection of [packages and package deployments](./00-package.md#packages-and-deployments) which are configured, tested, versioned, released, distributed, deployed, and upgraded together. A pallet is just a directory which contains a [`forklift-pallet.yml`](#pallet-definition) file and one or more packages and/or package deployments; that directory is the *pallet root directory*.

A pallet can be *applied* to a host operating system, which means that the operating system will be modified according to the contents of the pallet. A pallet should fully specify **all** Forklift package deployments which should be active on the operating system after the pallet is applied; only one pallet may be applied to the operating system at any time. When functionalities from multiple pallets are simultaneously required on the operating system, those pallets should instead be combined together through [pallet layering](#pallet-layering) to create a new pallet which provides all required functionalities from the combined pallets.

### Pallet paths
A *pallet path* is the canonical name for a Forklift pallet, declared in the pallet's [`forklift-pallet.yml`](#pallet-definition) file. If defined, a pallet path should convey both what the pallet does and where to find it (e.g. `github.com/openUC2/rpi-imswitch-os`).

If a pallet path is defined, then:

- The *package path* of each package in the pallet is simply the pallet path joined with the subdirectory containing the package, relative to the pallet root. For example, the pallet `github.com/openUC2/rpi-imswitch-os` contains a package in the subdirectory `deployments/infra/caddy.pkg`; that package's path is `github.com/openUC2/rpi-imswitch-os/deployments/infra/caddy.pkg`.
- Any other file in the pallet can also be specified with a path which is simply the pallet path joined with the path of the file, relative to the pallet root. For example, the pallet `github.com/openUC2/rpi-imswitch-os` contains a file at `deployments/infra/caddy.deploy.yml`; that file's full path is `github.com/openUC2/rpi-imswitch-os/deployments/infra/caddy.deploy.yml`.

Note that package paths and the paths of files in the pallet are not necessarily resolveable as web page URLs.

By convention, package paths should end in the suffix `.pkg`, e.g. `github.com/openUC2/rpi-imswitch-os/deployments/infra/caddy-ingress.pkg` instead of `github.com/openUC2/rpi-imswitch-os/deployments/infra/caddy-ingress`. This convention makes it easy to determine that, for example, `github.com/openUC2/rpi-imswitch-os/deployments/infra` isn't a package, but rather a directory which contains packages.

### Version control
A pallet may be published for reuse by being hosted as a Git repository at a stable location on the internet (e.g. on GitHub); each such pallet should be identified by a pallet path unique to it (e.g. `github.com/openUC2/rpi-imswitch-os`). Pallets may be downloaded directly as Git repositories. If the pallet is a Git repository, then the pallet root directory should be the root directory of the Git repository.

Usually we will speak of the pallet as the Git repository published online at the pallet path (e.g. "openUC2 has just released version v2026.0.0 of the pallet `github.com/openUC2/rpi-imswitch-os`"), but sometimes we may instead speak of a pallet as a local directory which is a clone of that Git repository (e.g. "your computer's local pallet is now `github.com/openUC2/rpi-imswitch-os`"); the exact meaning will depend on the context.

#### Versions
In a pallet which is a Git repository, a *version* is a Git tag which identifies an immutable snapshot of a pallet and all packages in the pallet. Thus, all packages in any single Git commit of a pallet will have always have identical versions. A version may be either a release (e.g. `v1.0.0`) or a pre-release (e.g. `v1.0.0-beta`). Once a Git tag is created, it should not be deleted or changed to a different revision. Versions should be authenticated to ensure safe, repeatable deployments. If a tag is modified, clients should be shown a security error when downloading it.

Each version starts with the letter `v`, followed by either a semantic version or a calendar version. The [Semantic Versioning 2.0.0 specification](https://semver.org/spec/v2.0.0.html) expains how semantic versions should be formatted, interpreted, and compared; the [Calendar Versioning reference](https://calver.org/) describes a variety of ways that calendar versions may be constructed, but any calendar versioning scheme used must meet the following requirements:

- The calendar version must have three parts (major, minor, and micro) separated by `.` characters, and it may have additional labels for pre-release metadata following the semantic versioning specification (so e.g. `v2022.1.1` and `v2022.0.2-alpha.1` are allowed, while `v2022-1-1` and `v2022-alpha.1` are not allowed).
- No version part may be zero-padded (so e.g. `v2022.4.0` and `v22.4.0` are allowed, while `v2022.04.0` and `v22.04.0` are not allowed).
- The calendar version must conform to the semantic versioning specifications for precedence, so that versions can be compared and sequentially ordered.

#### Pseudo-versions
A *pseudo-version* is a specially formatted pre-release version that encodes information about a specific commit in a Git repository. For example, `v0.0.0-20191109021931-daa7c04131f5` is a pseudo-version.

Pseudo-versions may refer to commits for which no semantic version tags are available. They may be used to test commits before creating version tags, for example, on a development branch.

Each pseudo-version has three parts:

1. A base version prefix (`vX.0.0` or `vX.Y.Z-0`), which is either derived from a semantic (or calendar) version tag that precedes the commit or `v0.0.0` if there is no such tag.
2. A timesetamp (`yyyymmddhhmmss`), which is the UTC time the commit was created. In Git, this is the commit time, not the author time.
3. A revision identifier (`abcdefabcdef`), which is the 12-character prefix of the commit hash.

Each pseudo-version may be in one of three forms, depending on the base version. These forms ensure that a pseudo-version compares higher than its base version, but lower than the next tagged version.

- `v0.0.0-yyyymmddhhmmss-abcdefabcdef` is used when there is no known base version.
- `vX.Y.Z-pre.0.yyyymmddhhmmss-abcdefabcdef` is used when the base version is a pre-release version like `vX.Y.Z-pre`. For example, if the base version is `v1.2.3-alpha.0`, a pseudo-version might be `v1.2.3-alpha.0.0.20191109021931-daa7c04131f5`.
- `vX.Y.(Z+1)-0.yyyymmddhhmmss-abcdefabcdef` is used when the base version is a release version like `vX.Y.Z`. For example, if the base version is `v1.2.3`, a pseudo-version might be `v1.2.4-0.20191109021931-daa7c04131f5`.

More than one pseudo-version may refer to the same commit by using different base versions. This happens naturally when a lower version is tagged after a pseudo-version is written.

These forms give pseudo-versions two useful properties:

- Pseudo-versions with known base versions sort higher than those versions but lower than other pre-releases for later versions.
- Pseudo-versions with the same base version prefix sort chronologically.

When pseudo-versions are used, several checks should be performed to ensure that pallet authors have control over how pseudo-versions are compared with other versions and that pseudo-versions refer to revisions that are actually part of a pallet's commit history:

- If a base version is specified, there should be a corresponding semantic (or calendar) version tag that is an ancestor of the commit described by the pseudo-version.
- The timestamp must match the commit's timestamp. This prevents pallet consumers from changing the relative ordering of versions.
- The commit must be an ancestor of one of the pallet repository's branches or tags. This prevents attackers from referring to unapproved changes or pull requests.

Pseudo-versions never need to be typed by hand. `forklift` commands accept a commit hash or a branch name as a *version query* and will automatically resolve the version query into a pseudo-version (or tagged version if available). For example:

```
forklift pallet switch github.com/openUC2/rpi-imswitch-os@main
# log output includes:
# Resolved github.com/openUC2/rpi-imswitch-os@main as v2026.0.0-alpha.0.0.20260122192233-9cb4541b2f69

forklift pallet switch github.com/openUC2/rpi-imswitch-os@9cb454
# log output includes:
# Resolved github.com/openUC2/rpi-imswitch-os@9cb454 as v2026.0.0-alpha.0.0.20260122192233-9cb4541b2f69

forklift inspector resolve-git-repo github.com/openUC2/rpi-imswitch-os@d126804
# output is: v2026.0.0-alpha.0
```

### Package deployments

A typical Forklift pallet defines a set of [*package deployments*](00-package.md#packages-and-deployments). Each package deployment in a pallet should be defined by a YAML file, named with a `.deploy.yml` extension, which is located either in the pallet's `deployments` directory or in a subdirectory of that directory.

Each package deployment also has a unique name corresponding to its path within the pallet; for example, in the pallet `github.com/openUC2/rpi-imswitch-os`, the package deployment declared by the file `/deployments/infra/caddy-ingress.deploy.yml` can be unambiguously identified with the abbreviated name `infra/caddy-ingress` in user interfaces; or it can be unambiguously identified with the name `github.com/openUC2/rpi-imswitch-os/deployments/infra/caddy-ingress.deploy.yml` from other pallets.

A package deployment in a pallet should specify the package it will deploy in one of the following ways:

1. If the package to be deployed is defined in a different pallet: the package deployment should refer to the package path, which includes the pallet path of the package's pallet (if the package to be deployed is defined in a different pallet. For example, a package deployment in some pallet which is not `github.com/openUC2/rpi-imswitch-os` would use the package path `github.com/openUC2/rpi-imswitch-os/deployments/infra/caddy-ingress.pkg` in order to refer to that package.
2. If the package to be deployed is defined in the same pallet: the package deployment should refer to the subdirectory path of the package (relative to the pallet's root directory), with a leading `/` character prepended to the path. For example, in the pallet `github.com/openUC2/rpi-imswitch-os`, the package deployment at `deployments/infra/caddy-ingress.deploy.yml` uses the path `/deployments/infra/caddy-ingress.pkg` in order to refer to the package defined at `deployments/infra/caddy-ingress.pkg` in that same pallet.

By convention, each package deployment file should usually be placed in the same directory as the package which it deploys. For example, the pallet `github.com/openUC2/rpi-imswitch-os` contains a package deployment for `github.com/openUC2/rpi-imswitch-os/deployments/infra/caddy-ingress.pkg` at `/deployments/infra/caddy-ingress.deploy.yml`.

### Resolving package paths

Because a pallet can exist at multiple [versions](#versions), a package path alone is not sufficient to unambiguously determine which version of a package is being referred to by a package deployment when the package and the package deployment are defined in different pallets. Thus, the package deployment's pallet must provide additional information about the required version of the pallet which provides the package to be deployed.

For any package deployment in pallet A which deploys a package provided by pallet B, pallet B must be specified with a particular version (or [pseudo-version](#pseudo-version)) in a [`forklift-version-lock.yml`](#pallet-requirement-definition) file in pallet A's `requirements/pallets` subdirectory, in a sub-subdirectory path identical to pallet B's path. That sub-subdirectory is pallet A's *pallet requirement* directory for pallet B. For example, a pallet A which includes a deployment of the `github.com/openUC2/rpi-imswitch-os/deployments/infra/caddy-ingress.pkg` package (provided by a pallet B whose pallet path is `github.com/openUC2/rpi-imswitch-os`, which is different from pallet B) must also include a file at `requirements/pallets/github.com/openUC2/rpi-imswitch-os/forklift-version-lock.yml`. Then pallet A's subdirectory `requirements/pallets/github.com/openUC2/rpi-imswitch-os` is its pallet requirement directory for `github.com/openUC2/rpi-imswitch-os`.

When a package deployment refers to a package by its package path, the pallet which provides that package is determined by checking for the existence of pallet requirement directories for pallet paths starting from the package path its parents at increasing levels. For example, the package path `github.com/openUC2/rpi-imswitch-os/deployments/infra/caddy-ingress.pkg` would be resolved in the following way:

1. If a pallet requirement directory exists at `requirements/pallets/github.com/openUC2/rpi-imswitch-os/deployments/infra/caddy-ingress.pkg`, then the pallet is determined to be `github.com/openUC2/rpi-imswitch-os/deployments/infra/caddy-ingress.pkg`, and a specific version can be loaded from the `forklift-version-lock.yml` file. Otherwise,
2. If a pallet requirement directory exists at `requirements/pallets/github.com/openUC2/rpi-imswitch-os/deployments/infra`, then the pallet is determined to be `github.com/openUC2/rpi-imswitch-os/deployments/infra`. Otherwise,
3. If a pallet requirement directory exists at `requirements/pallets/github.com/openUC2/rpi-imswitch-os/deployments`, then the pallet is determined to be `github.com/openUC2/rpi-imswitch-os/deployments`. Otherwise,
4. If a pallet requirement directory exists at `requirements/pallets/github.com/openUC2/rpi-imswitch-os`, then the pallet is determined to be `github.com/openUC2/rpi-imswitch-os`. Otherwise,
5. If a pallet requirement directory exists at `requirements/pallets/github.com/openUC2`, then the pallet is determined to be `github.com/openUC2`. Otherwise,
6. If a pallet requirement directory exists at `requirements/pallets/github.com`, then the pallet is determined to be `github.com`. Otherwise,
7. The pallet cannot be determined; this is an error.

In this case, proper resolution of the package path would require the existence of a pallet requirement directory at `requirements/pallets/github.com/openUC2/rpi-imswitch-os`.

## Pallet layering

TODO

(mention how this is inspired by the way files from various Docker container images can be combined together into a new Docker container image)

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

The definition of a package deployment is stored in a YAML file with file extension `.deploy.yml` within the pallet's subdirectory `deployments`. Here is an example of a package deployment file, in this case at `deployments/infra/caddy-ingress.deploy.yml`:

```
package: /deployments/infra/caddy-ingress.pkg
features:
  - https
  - firewall-allow-direct-http
  - firewall-allow-direct-https
disabled: false
```

### `package` field

This field of the `*.deploy.yml` file declares the package to be deployed by the package deployment.

- This field is required.

- Allowed values are:

  - A [package path](#pallet-paths) for a package provided by an external pallet which can be [resolved](#resolving-package-paths), e.g. `github.com/openUC2/rpi-imswitch-os/deployments/infra/caddy-ingress.pkg`.

  - The subdirectory path of a package in the same pallet as the deployment, formatted like an absolute path relative to the pallet root directory, e.g. `/deployments/infra/caddy-ingress.pkg`

- Example:
  
  ```yaml
  package: github.com/openUC2/rpi-imswitch-os/deployments/infra/caddy-ingress.pkg
  ```

### `features` field

This field of the `*.deploy.yml` file is a list of enabled feature flags, each of which is the string name of a feature exposed by the package to be deployed.

- This field is optional.

- Feature flags are evaluated in lexicographic (alphabetical) order, regardless of the order in which they are listed in the `*.deploy.yml` file.

- Only feature flags in this list will be enabled for the package to be deployed. All other features will be ignored in this package deployment.

- Example:
  
  ```yaml
  features:
    - https
    - firewall-allow-direct-http
  ```

### `disabled` field

This field of the `*.deploy.yml` file is a boolean flag declaring whether the package deployment should be ignored when a pallet is applied, so that the package deployment will not have any effects.

- This field is optional.

- Example:
  
  ```yaml
  disabled: true
  ```

## Pallet requirement definition

The definition of a pallet requirement for an external pallet is stored in the subdirectory `requirements/pallets/{external pallet's path}`, where `{external pallet's path}` should be replaced with the external pallet's path.

### Version requirement definition

The declaration of a version requirement for a pallet requirement is stored in a YAML file named `forklift-version-lock.yml` at the root of the pallet requirement directory. Here is an example of a `forklift-version-lock.yml` file, in this case at `requirements/pallets/github.com/PlanktoScope/pallet-standard/forklift-version-lock.yml`:

```yaml
type: pseudoversion
tag: v2025.0.0-alpha.0
timestamp: "20250702170448"
commit: d6b96488a5c4d8520135c66bd888fc0e933f321e
```

#### `type` field

This field of the `forklift-version-lock.yml` file declares the type of version reference for the pallet requirement.

- This field is required.

- Allowed values are:

  - `version`: a specific [tagged version](#versions) is required.

  - `pseudoversion`: a specific [pseudo-version](#pseudo-versions) is required.

- Example:
  
  ```yaml
  type: version
  ```

#### `tag` field

This field of the `forklift-version-lock.yml` file declares the tagged version (or base tag of a pseudo-version) for the pallet requirement.

- This field is required.

- For the `version` type, the tag is interpreted as specific tagged version which is required.

- For the `pseudoversion` type, the tag is interpreted as the base version prefix of the specific pseudo-version which is required.

- Example:
  
  ```yaml
  tag: v2024.0.0
  ```

#### `timestamp` field

This field of the `forklift-version-lock.yml` file declares the commit time of the Git commit for the pallet requirement.

- This field is required.

- For the `version` type, the timestamp is the commit time of the commit which the tagged version points to.

- For the `pseudoversion` type, the timestamp is the commit time of the commit which the pseudo-version refers to.

- Example:
  
  ```yaml
  timestamp: "20241226040736"
  ```

#### `commit` field

This field of the `forklift-version-lock.yml` file declares the full commit hash of the Git commit for the pallet requirement.

- This field is required.

- For the `version` type, the commit is the commit hash of the commit which the tagged version points to.

- For the `pseudoversion` type, the commit is the commit hash of the commit which the pseudo-version refers to.

- Example:
  
  ```yaml
  commit: afe66e5714587ea34c320025edc5a37d5e857488
  ```

### File import definition

TODO