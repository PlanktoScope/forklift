# Forklift package specification

This specification defines Forklift packages and repositories.

## Introduction

This specification's design is heavily inspired by the design of the Go programming language and its module system, and this reference document tries to echo the [reference document for Go modules](https://go.dev/ref/mod) for familiarity; certain aspects of packages are also inspired by the Rust programming language's [Cargo](https://doc.rust-lang.org/cargo/) package management system.

## Repositories, packages, and versions

A Forklift *repository* is a collection of software configuration packages which are tested, released, distributed, and upgraded together. Packages are how Forklift makes apps recomposable. A Forklift repository is just a Git repository hosted at a stable location on the internet (e.g. on GitHub), with a special configuration file declaring the repository. A repository is identified by a [*repository path*](#repository-paths), which is declared in a `forklift-repository.yml` file at the root of the repository.

A Forklift *package* is a configuration of a software application which can be deployed on a Docker host. Each package within a repository specifies the prerequisites and consequences of its deployment on the host. Typically, a package declares some or all of the following:

- A Docker Compose application which will be deployed on the Docker host.
- A set of optional [*features*](#package-features) which may be enabled for the deployment. Features control the functionalities or behavior of the *package deployment*, which is fully and uniquely specified by declaring the name of the deployment, the package to deploy, and the features to enable in the deployment. Thus, a single package may be instantiated in multiple distinct deployments (each with different configurations) on a Docker host. Features work by modifying the specification of the Docker Compose file.
- A set of [*constraints*](#package-deployments-and-constraints) for determining whether the package deployment is allowed on the Docker host, including a list of any host [*resources*](#package-resource-constraints) which are required by the package deployment or provided by the package deployment. Features may also declare their own constraints, which are only evaluated when those features are enabled.

All of this information is declared in a `forklift-package.yml` file. The *package root directory* is the directory that contains the package's `forklift-package.yml` file.

### Repository paths

A *repository path* is the canonical name for a Forklift repository, declared with the `path` field in the repository's `forklift-repository.yml` file. A repository's path is the prefix for the package paths of packages provided by the repository.

A repository path should communicate both what the repository does and where to find it. A Forklift repository path is just the path of the Git repository which contains the Forklift repository. `github.com/PlanktoScope/device-pkgs` is an example of a repository path.

### Versions

A *version* is a Git tag which identifies an immutable snapshot of a repository and all packages in the repository; thus, all packages in any single commit of a repository will have always have identical versions, and all packages in a repository will always have the same version for a given Git commit. A version may be either a release or a pre-release. Once a Git tag is created, it should not be deleted or changed to a different revision. Versions should be authenticated to ensure safe, repeatable deployments. If a tag is modified, clients may see a security error when downloading it.

Each version starts with the letter `v`, followed by either a semantic version or a calendar version. The [Semantic Versioning 2.0.0 specification](https://semver.org/spec/v2.0.0.html) expains how semantic versions should be formatted, interpreted, and compared; the [Calendar Versioning reference](https://calver.org/) describes a variety of ways that calendar versions may be constructed, but any calendar versioning scheme used must meet the following requirements:

- The calendar version must have three parts (major, minor, and micro), and it may have additional labels for pre-release and build metadata following the semantic versioning specification.
- No version part may be zero-padded (so e.g. `2022.4.0` and `22.4.0` are allowed, while `2022.04.0` and `22.04.0` are not allowed).
- The calendar version must conform to the semantic versioning specifications for precedence, so that versions can be compared and sequentially ordered.

### Package paths

The path of a package is the repository path joined with the subdirectory (relative to the repository root) which contains the package's `forklift-package.yml` file. That subdirectory is the *package root directory*. For example, the repository `github.com/PlanktoScope/device-pkgs` contains a Forklift package in the subdirectory `core/infra/caddy-ingress`. The `core/infra/caddy-ingress` directory contains a `forklift-package.yml` file and thus is the root directory of a package which has a package path of `github.com/PlanktoScope/device-pkgs/core/infra/caddy-ingress`. Note that the package path is not necessarily resolveable as a web page URL (so for example <https://github.com/PlanktoScope/device-pkgs/core/infra/caddy-ingress> gives a HTTP 404 Not Found error), because the package path is only resolvable in the context of a specific GitHub repository version.

## Package deployments and constraints

Usually, multiple package deployments are simultaneously active on a Docker host, and multiple package deployments may be modified by a package manager operation, for example:

- Adding new package deployments
- Removing existing package deployments
- Modifying the enabled features of existing package deployments
- Upgrading the versions of the repositories providing deployed packages
- Downgrading the versions of the repositories providing deployed packages

Each such operation will modify the set of all active package deployments on the Docker host, and it will succeed if (and only if) all of the following constraints will be satisfied by the resulting set of all package deployments:

- Package deployment name constraints:
   - Uniqueness constraint: no package deployment will attempt to use the same name as another distinct package deployment; package deployments are distinct if they have different package paths and/or if they declare different sets of enabled features.
- Resource constraints:
   - Dependency constraint (*resource requirements*): all of the resources required by all of the active package deployments will also be resources provided by some subset of the active package deployments.
   - Uniqueness constraint (*provided resources*): none of the resources provided by any of the active package deployments will conflict with resources provided by any other active package deployments.

### Package resource constraints

The resource requirements and provided resources associated with a package deployment are its *resource interface* and are part of the set of constraints which determine whether a set of package deployments is allowed. When a set of package deployments is not allowed, information about unsatisfied resource constraints should be used by the package manager to help users correct resource conflicts and unresolved resource dependencies between package deployments. The resource interface of a package deployment is determined from the package deployment's configuration and information specified by the package. The design of the resource interface system for determining the validity of a combination of package deployments is inspired by design of implicit interfaces in the Go programming language.

A package deployment's declaration of resource requirements and provided resources is also a declaration of its external interface on the Docker host. Currently, resources can be:

1. Docker networks
2. Host port listeners bound to network ports on the host
3. Network services mapped to the host
4. Files on the host (which may be dynamically generated by software deployed by the package deployment, and are specified relative to the filesystem root)
5. Files exported to the host (which are statically specified by the package deployment, and are specified relative to an export path defined by each implementation of the Forklift packaging specification)

Resource requirements and provided resources are specified as a set of *identification criteria* for determining whether two provided resources have conflicting identities or whether the identity of a package deployment's required resource matches the identity of a resource provided by another package.

Because some Docker hosts may already have ambiently-available resources not provided by applications running in Docker (for example, an SSH server on port 22 installed using `apt-get`), a Forklift package may also include a list of resources already ambiently provided by the host; if such a resource is declared, it should be provided by the Docker host regardless of whether the package is deployed. Adding or removing a deployment of such a package will not affect the actual existence of such resources; it will only change a package manager's assumptions about what resources are ambiently provided by the host.

### Package features

Forklift package *features* provide a mechanism to express optional resource constraints (both required resources and provided resources) and functionalities of a package. Each feature is identified by a name unique within the package. The design of Forklift package features is inspired by the design of the [features system](https://doc.rust-lang.org/cargo/reference/features.html) in the Rust Cargo package management system.

A package defines a set of named features in its `forklift-package.yml` metadata file, and each feature can be either enabled or disabled by a package manager. Each package feature specifies any resources it requires from the Docker host, as well as any resources it provides on the Docker host, and the names of any additional Docker Compose files which should be merged together into the Docker Compose application defined by the package when the feature is enabled.

### Versioning with constraints and features

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

## Repository definition

The definition of a repository is stored in a YAML file named `forklift-repository.yml` in the repository's root directory. Here is an example of a `forklift-repository.yml` file:

```yaml
forklift-version: v0.4.0

repository:
  path: github.com/PlanktoScope/device-pkgs
  description: Packages for the PlanktoScope software distribution
  readme-file: README.md
```

### `forklift-version` field

This field of the `forklift-repository.yml` file declares that the repository was written assuming the semantics of a given version of Forklift.

- This field is required.

- The version must be a valid version of the Forklift tool or the Forklift specification.

- The version sets the minimum version of the Forklift tool required to use the repository. The Forklift tool refuses to use repositories declaring newer Forklift versions (or excessively old Forklift versions) for any operations beyond printing information.

- Example:
  
  ```yaml
  forklift-version: v0.4.0
  ```

All other fields in the repository metadata file are under a `repository` section.

### `repository` section

This section of the `forklift-repository.yml` file contains some basic metadata to help describe and identify the repository. Here is an example of a `repository` section:

```yaml
repository:
  path: github.com/PlanktoScope/device-pkgs
  description: Packages for the PlanktoScope software distribution
  readme-file: README.md
```

#### `path` field

This field of the `repository` section is the repository path.

- This field is required.

- Example:
  
  ```yaml
  path: github.com/PlanktoScope/device-pkgs
  ```

#### `description` field

This field of the `repository` section is a short (one-sentence) description of the repository to be shown to users.

- This field is required.

- Example:
  
  ```yaml
  description: Packages for the PlanktoScope software distribution
  ```

#### `readme-file` field

This field of the `repository` section is the filename of a readme file to be shown to users.

- This field is required.

- The file must be located in the same directory as the `forklift-repository.yml` file.

- The file must be a text file.

- It is recommended for this file to be named `README.md` and to be formatted in [GitHub-flavored Markdown](https://github.github.com/gfm/).

- Example:
  
  ```yaml
  readme-file: README.md
  ```

## Package definition

The definition of a package is stored in a YAML file named `forklift-package.yml` in the package's root directory. Here is an example of a `forklift-package.yml` file:

```yaml
package:
  description: Reverse proxy for web services
  maintainers:
    - name: Ethan Li
      email: lietk12@gmail.com
  license: MIT
  sources:
    - https://github.com/lucaslorentz/caddy-docker-proxy

deployment:
  name: caddy-ingress
  compose-files:
    - compose.yml

features:
  service-proxy:
    description: Provides reverse-proxying access to Docker Swarm services defined by other packages
    requires:
      networks:
        - description: Bridge network to the host
          name: bridge
    provides:
      networks:
        - description: Overlay network for Caddy to connect to upstream services
          name: caddy-ingress
      listeners:
        - description: Web server for all HTTP requests
          port: 80
          protocol: tcp
        - description: Web server for all HTTPS requests
          port: 443
          protocol: tcp
      services:
        - description: Web server which reverse-proxies PlanktoScope web services
          tags: [caddy-docker-proxy]
          port: 80
          protocol: http
        - description: Reverse-proxy web server which provides TLS termination to PlanktoScope web services
          tags: [caddy-docker-proxy]
          port: 443
          protocol: https
```

The file has four sections: `package`, `host` (an optional section), `deployment` (a required section), and `features` (an optional section).

### `package` section

This section of the `forklift-package.yml` file contains some basic metadata to help describe and identify the package. Here is an example of a `package` section:

```yaml
package:
  description: MQTT broker ambiently provided by the PlanktoScope
  maintainers:
    - name: Ethan Li
      email: lietk12@gmail.com
  license: (EPL-2.0 OR BSD-3-Clause)
  sources:
    - https://github.com/eclipse/mosquitto
```

#### `description` field

This field of the `package` section is a short (one-sentence) description of the package to be shown to users.

- This field is required.

- Example:
  
  ```yaml
  description: Web GUI for operating the PlanktoScope
  ```

#### `maintainers` field

This field of the `package` section is an array of maintainer objects listing the people who maintain the Forklift package.

- This field is optional.

- In most cases, the maintainers of the Forklift package will be different from the maintainers of the original software applications provided by the package. The maintainers of the package are specifically the people responsible for maintaining the software configurations specified by the package.

- Example:
  
  ```yaml
  maintainers:
    - name: Ethan Li
      email: lietk12@gmail.com
    - name: Thibaut Pollina
  ```

A maintainer object consists of the following fields:

- `name` is a string with the maintainer's name.
  
   - This field is optional.
  
   - Example:
     
     ```yaml
     name: Ethan Li
     ```

- `email` is a string with an email address for contacting the maintainer.
  
   - This field is optional.
  
   - Example:
     
     ```yaml
     email: lietk12@gmail.com
     ```

#### `license` field

This field of the `package` section is an [SPDX 2.1 license expression](https://spdx.github.io/spdx-spec/v2-draft/SPDX-license-expressions/) specifying the licensing terms of the software provided by the Forklift package.

- This field is optional.

- Usually, an SPDX license name will be sufficient; however, some software applications are released under multiple licenses, in which case a more complex SPDX license expression (such as `MIT OR Apache-2.0`) is needed.

- If a package is using a nonstandard license, then the `license-file` field may be specified in lieu of the `license` field.

- Example:
  
  ```yaml
  license: GPL-3.0
  ```

#### `license-file` field

This field of the `package` section is the filename of a license file describing the licensing terms of the software provided by the Forklift package.

- This field is optional.

- The file must be a text file.

- The file must be located in the same directory as the `forklift-package.yml` file.

- Example:
  
  ```yaml
  license-file: LICENSE-ZeroTier-BSL
  ```

#### `sources` field

This field of the `package` section is an array of URLs which can be opened to access the source code for the software provided by the Forklift package.

- This field is optional.

- Example:
  
  ```yaml
  sources:
    - https://github.com/zerotier/ZeroTierOne
    - https://github.com/sargassum-world/docker-zerotier-controller
  ```

### `host` section

This optional section of the `forklift-package.yml` file describes any relevant resources already ambiently provided by the Docker host. Such resources will exist whether or not the package is deployed; specifying resources in this section provides necessary information for checking [package resource constraints](#package-resource-constraints). Here is an example of a `host` section:

```yaml
host:
  tags:
    - device-portal-name=Cockpit (direct-access fallback)
    - device-portal-description=Provides fallback access to the Cockpit application, accessible even if the system's service proxy stops working
    - device-portal-type=Browser applications
    - device-portal-purpose=System recovery
    - device-portal-entrypoint=/admin/cockpit/
  provides:
    listeners:
      - description: Web server for the Cockpit dashboard
        port: 9090
        protocol: tcp
    services:
      - description: The Cockpit system administration dashboard
        port: 9090
        protocol: http
        paths:
          - /admin/cockpit/*
```

#### `tags` field

This field of the `host` section is an array of strings to associate with the host or with resources provided by the host. These tags have no semantic meaning within the Forklift package specification, but can be used by other applications for arbitrary purposes.

- This field is optional.

- Example:
  
  ```yaml
  tags:
    - device-portal-name=SSH server
    - device-portal-description=Provides SSH access to the PlanktoScope on port 22
    - device-portal-type=System infrastructure
    - device-portal-purpose=Networking
    - systemd-service=sshd.service
    - config-file=/etc/ssh/sshd_config
    - system
    - networking
    - remote-access
  ```

#### `provides` subsection

This optional subsection of the `host` section specifies the resources ambiently provided by the Docker host. Here is an example of a `provides` section:

```yaml
provides:
  listeners:
    - description: SSH server
      port: 22
      protocol: tcp
  services:
    - description: SSH server
      tags: [sshd]
      port: 22
      protocol: ssh
```

##### `listeners` field

This field of the `provides` subsection is an array of host port listener objects listing the network port/protocol pairs which are already bound to host processes which are running on the Docker host and listening for incoming traffic on those port/protocol pairs, on any/all IP addresses.

- This field is optional.

- Each host port listener object describes a host port listener resource which may or may not be in conflict with other host port listener resources; this is because multiple processes are not allowed to simultaneously bind to the same port/protocol pair on all IP addresses.

- If a set of package deployments contains two or more host port listener resources for the same port/protocol pair from different package deployments, the package deployments declaring those respective host port listeners will be reported as conflicting with each other. Therefore, the overall set of package deployments will not be allowed because its [package resource constraints](#package-resource-constraints) for uniqueness of host port listener resources will not be satisfied.

- Currently, this specification does not allow multiple host port listeners to bind to the same port/protocol pair on different IP addresses; instead for simplicity, processes are assumed to be listening for that port/protocol pair on *all* IP addresses on the host.

- Example:
  
  ```yaml
  listeners:
    - description: ZeroTier traffic to the rest of the world
      port: 9993
      protocol: udp
    - description: ZeroTier API for control from the ZeroTier UI and the ZeroTier CLI
      port: 9993
      protocol: tcp
  ```

A host port listener object consists of the following fields:

- `description` is a short (one-sentence) description of the host port listener resource to be shown to users.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     description: Web server for the Cockpit dashboard
     ```

- `port` is a number specifying the [network port](https://en.wikipedia.org/wiki/Port_(computer_networking)) bound by a process running on the host.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     port: 9090
     ```

- `protocol` is a string specifying whether the bound network port is for the TCP transport protocol or for the UDP transport protocol.
  
   - This field is required.
  
   - The value of this field must be either `tcp` or `udp`.
  
   - Example:
     
     ```yaml
     protocol: tcp
     ```

##### `networks` field

This field of the `provides` subsection is an array of host Docker network objects listing the Docker networks which are already available on the Docker host.

- This field is optional.

- Each host Docker network object describes a Docker network resource which may or may not be in conflict with other Docker network resources; this is because multiple Docker networks are not allowed to have the same name.

- If a set of package deployments contains two or more Docker network resources for networks with the same name from different package deployments, the package deployments declaring those respective Docker networks will be reported as conflicting with each other. Therefore, the overall set of package deployments will not be allowed because its [package resource constraints](#package-resource-constraints) for uniqueness of host Docker network names will not be satisfied.

- Example:
  
  ```yaml
  networks:
    - description: Default bridge to the host
      name: bridge
  ```

A Docker network object consists of the following fields:

- `description` is a short (one-sentence) description of the Docker network resource to be shown to users.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     description: Default host network
     ```

- `name` is a string specifying the name of the Docker network.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     name: host
     ```

##### `services` field

This field of the `provides` subsection is an array of network service objects listing the network services which are already available on the Docker host.

- This field is optional.

- The route of a network service can be defined either as a port/protocol pair or as a combination of port, protocol, and one or more paths. A network service whose route is defined only as a port/protocol pair will overlap with another network service if and only if the other network service whose route is also defined only as a port/protocol pair. A network service whose route is defined with one or more paths will overlap with another network service if and only if both network services have the same port, the same protocol, and at least one overlapping path (for a definition of overlapping paths, refer below to description of the `path` field of the network service object).

- Each network service object describes a network service resource which may or may not be in conflict with other network service resources; this is because multiple network services are not allowed to have overlapping routes.

- If a set of package deployments contains two or more network service resources for services with overlapping routes from different package deployments, then the package deployments declaring those respective network services will be reported as conflicting with each other. Therefore, the overall set of package deployments will not be allowed because its [package resource constraints](#package-resource-constraints) for uniqueness of network services will not be satisfied.

- Example:
  
  ```yaml
  services:
    - description: SSH server
      port: 22
      protocol: ssh
  ```

A network service object consists of the following fields:

- `description` is a short (one-sentence) description of the network service resource to be shown to users.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     description: The Cockpit system administration dashboard
     ```

- `port` is a number specifying the network port used for accessing the service.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     port: 9090
     ```

- `protocol` is a string specifying the application-level protocol used for accessing the service.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     protocol: https
     ```

- `paths` is an array of strings which are paths used for accessing the service.
  
   - This field is optional.
  
   - A path may optionally have an asterisk (`*`) at the end, in which case it is a prefix path - so the network service covers all paths beginning with that prefix (i.e. the string before the asterisk).
  
   - If a network service specifies a port and protocol but no paths, it will conflict with another network service which also specifies the same port and protocol but no paths; it will not conflict with another network service which specifies the same port and protocol and also specifies some paths. In other words, not listing any paths in a network service is equivalent to not having any conflicts with other services available at specific paths on the same port and protocol.
     This is useful for describing systems involving HTTP reverse-proxies or involving message brokers, where one package deployment may provide a network service which routes specific messages to network services from other package deployments on specific paths; then the reverse-proxy or message broker would be specified on some port and protocol with no paths, while the network services behind it would be specified on the same port and protocol but with a set of specific paths.
  
   - If a package deployment has a dependency on a network service with a specific path which matches a prefix path in a network service from another package deployment, that dependency will be satisfied. For example, a dependency on a network service requiring a path `/admin/cockpit/system` would be met by a network service provided with the path prefix `/admin/cockpit/*`, assuming they have the same port and protocol.
  
   - If a package deployment provides a network service with a specific path which matches a prefix path in a network service provided by another package deployment, those two package deployments will be in conflict with each other. For example, a network service providing a path `/admin/cockpit/system` would conflict with a network service providing the path prefix `/admin/cockpit/*`, assuming they have the same port and protocol. This is because those overlapping paths would cause the network services to overlap with each other, which is not allowed.
  
   - Example:
     
     ```yaml
     paths:
       - /admin/cockpit/*
     ```

- `tags` is an array of strings which constrain resolution of network service resource dependencies among package deployments. These tags are ignored in determining whether network services conflict with each other, since they are not part of the network service's route.
  
   - This field is optional.
  
   - These tags have no semantic meaning within the Forklift package specification, but tag requirements can be used for arbitrary purposes. For example, tags can be used to annotate a network service with information about API versions, subprotocols, etc. If a package deployment specifies that it requires a network service with one or more tags, then another package deployment will only be considered to satisfy the network service dependency if it provides a network service matching both the required route and all required tags. This is useful in ensuring that a network service provided by one package deployment is compatible with the API version required by a service client from another package deployment, for example.
  
   - Example:
     
     ```yaml
     tags:
       - https-only
       - tls-client-certs-required
     ```

##### `filesets` field

This field of the `provides` subsection is an array of fileset objects listing the files (which can include directories) which are already available on the Docker host.

- This field is optional.

- A fileset is defined as a list of one or more paths to files. A fileset will overlap with another fileset if and only if both filesets have at least one overlapping path (for a definition of overlapping paths, refer below to description of the `path` field of the fileset object).

- Each fileset object describes a fileset resource which may or may not be in conflict with other fileset resources; this is because multiple filesets are not allowed to have overlapping paths.

- If a set of package deployments contains two or more fileset resources for filesets with overlapping paths from different package deployments, then the package deployments declaring those respective filesets will be reported as conflicting with each other. Therefore, the overall set of package deployments will not be allowed because its [package resource constraints](#package-resource-constraints) for uniqueness of filesets will not be satisfied.

- Example:
  
  ```yaml
  filesets:
    - description: File containing the device's machine name
      paths:
      - /var/lib/planktoscope/machine-name
  ```

A fileset object consists of the following fields:

- `description` is a short (one-sentence) description of the fileset resource to be shown to users.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     description: Directory tree containing PlanktoScope datasets
     ```

- `paths` is an array of strings which are paths where the fileset exists.
  
   - This field is required.
  
   - A path may optionally have an asterisk (`*`) at the end, in which case it is a prefix path - so the fileset covers all paths beginning with that prefix (i.e. the string before the asterisk).
  
   - If a package deployment has a dependency on a fileset with a specific path which matches a prefix path in a fileset from another package deployment, that dependency will be satisfied. For example, a dependency on a fileset requiring a path `/home/pi/data/img` would be met by a fileset provided with the path prefix `/home/pi/data/*`.
  
   - If a package deployment provides a fileset with a specific path which matches a prefix path in a fileset provided by another package deployment, those two package deployments will be in conflict with each other. For example, a fileset providing a path `/home/pi/data/img` would conflict with a network service providing the path prefix `/home/pi/*`. This is because those overlapping paths would cause the filesets to overlap with each other, which is not allowed.
  
   - Example:
     
     ```yaml
     paths:
       - /home/pi/data
       - /home/pi/data/img
       - /home/pi/data/export
     ```

- `tags` is an array of strings which constrain resolution of fileset resource dependencies among package deployments. These tags are ignored in determining whether filesets conflict with each other, since they are not part of the fileset's location.
  
   - This field is optional.
  
   - These tags have no semantic meaning within the Forklift package specification, but tag requirements can be used for arbitrary purposes. For example, tags can be used to annotate a file with information about file type, file permissions, schema versions, etc. If a package deployment specifies that it requires a fileset with one or more tags, then another package deployment will only be considered to satisfy the fileset dependency if it provides a fileset matching both the required path(s) and all required tags. This is useful in ensuring that a fileset provided by one package deployment is compatible with the schema version required by another package deployment, for example.
  
   - Example:
     
     ```yaml
     tags:
       - directory
       - owner-1000
       - writable
     ```

### `deployment` section

This optional section of the `forklift-package.yml` file specifies the Docker Compose file provided by the package, as well as any resources required for deployment of the package to succeed, as well as any resources provided by deployment of the package. If resource requirements are not met, the deployment will not be allowed; resources provided by deployment of the package will only exist once the package deployment is successfully applied. Here is an example of a `deployment` section:

```yaml
deployment:
  compose-files:
    - compose.yml
  provides:
    networks:
      - description: Overlay network for the Portainer server to connect to Portainer agents
        name: portainer-agent
```

#### `compose-files` field

This field of the `deployment` section is an array of the string filenames of one or more Docker Compose files specifying the Docker Compose application which will be deployed when the package is deployed.

- This field is optional.

- The filenames must be for YAML files following the [Docker Compose file specification](https://docs.docker.com/compose/compose-file/).

- The files must be located in the same directory as the `forklift-package.yml` file, or in subdirectories.

- Example:
  
  ```yaml
  compose-files:
    - compose.yml
  ```

#### `tags` field

This field of the `deployment` section is an array of strings to associate with the package deployment or with resources required or provided by the package deployment. These tags have no semantic meaning within the Forklift package specification, but can be used by other applications for arbitrary purposes.

- This field is optional.

- Example:
  
  ```yaml
  tags:
    - remote-access
  ```

#### `requires` subsection

This optional subsection of the `deployment` section specifies the resources required for a deployment of the package to successfully become active. Here is an example of a `requires` section:

```yaml
requires:
  services:
    - tags: [planktoscope-api-v2]
      port: 1883
      protocol: mqtt
      paths:
        - /actuator/pump
        - /actuator/focus
        - /imager/image
        - /segmenter/segment
        - /status/pump
        - /status/focus
        - /status/imager
        - /status/segmenter
        - /status/segmenter/name
        - /status/segmenter/object_id
        - /status/segmenter/metric
```

##### `networks` field

This optional field of the `requires` subsection is an array of Docker network objects listing the Docker networks which must be available on the Docker host in order for a deployment of the package to successfully become active.

- This field is optional.

- The Docker network object describes a Docker network which must be provided by either the Docker host itself or by another package deployment. If the Docker network does not exist and won't be created, then the package deployment will not be allowed because its [package resource constraints](#package-resource-constraints) for dependencies on Docker networks will not be satisfied.

- Example:
  
  ```yaml
  networks:
    - description: Overlay network for Caddy to connect to upstream services
      name: caddy-ingress
  ```

A Docker network object consists of the following fields:

- `description` is a short (one-sentence) description of the required Docker network resource to be shown to users.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     description: Overlay network for the Portainer server to connect to Portainer agents
     ```

- `name` is a string specifying the name of the required Docker network.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     name: portainer-agent
     ```

##### `services` field

This optional field of the `requires` subsection is an array of network service objects listing the network services which must be available on the Docker host in order for a deployment of the package to successfully become active.

- This field is optional.

- The route of a network service requirement can be defined either as a port/protocol pair or as a combination of port, protocol, and one or more paths. A network service requirement whose route is defined only as a port/protocol pair can be satisfied by a network service defined with or without paths. A network service requirement whose route is defined with one or more paths will be satisfied by one or more network services if and only if all of those network services have the same port/protocol pair as the network service requirement, *and* the set union of the paths of the network services overlaps with every path listed in the network service requirement (for a definition of overlapping paths, refer below to description of the `path` field of the network service object).
  Thus, in any particular set of package deployments, one network service from one package deployment may be sufficient to satisfy a network service requirement from some other package deployment, or multiple network services from multiple packages may be necessary to fully satisfy that network service requirement.

- If a set of package deployments contains a network service resource requirement with a route which does not overlap with the routes of any network services provided by other package deployments, then the package deployment declaring that network service requirement will be reported as having an unmet dependency. Therefore, the overall set of package deployments will not be allowed because its [package resource constraints](#package-resource-constraints) for resource dependencies will not be satisfied.

- Example:
  
  ```yaml
  services:
    - description: A reverse-proxy server configured with Docker labels
      tags: [caddy-docker-proxy]
      port: 80
      protocol: http
  ```

A network service object consists of the following fields:

- `description` is a short (one-sentence) description of the network service resource requirement to be shown to users.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     description: An MJPEG stream from the Raspberry Pi's camera
     ```

- `port` is a number specifying the network port which must be usable for accessing the required service.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     port: 8000
     ```

- `protocol` is a string specifying the application-level protocol which must be usable for accessing the required service.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     protocol: http
     ```

- `paths` is an array of strings which are paths which must be accessible on the required service.
  
   - This field is optional.
  
   - A path may optionally have an asterisk (`*`) at the end, in which case it is a prefix path - so the required network service must declare that it can be used with any path beginning with that prefix (i.e. the string before the asterisk).
  
   - If a network service requirement specifies a port and protocol but no paths, that requirement will be satisfied by any network service which also specifies the same port and protocol and has the required tags (if any), regardless of whether the service specifies any paths. In other words, not listing any paths in a network service requirement is equivalent to not having any requirements about the paths exposed by a network service.
  
   - If a package deployment has a requirement for a network service with a specific path which matches the prefix path of a network service provided by another package deployment, the network service requirement will be met. For example, a requirement for a network service with a path `/stream.mjpg` would be met by a network service provided with the path prefix `/*`, assuming they have the same port and protocol.
  
   - Example:
     
     ```yaml
     paths:
       - /stream.mjpg
     ```

- `tags` is an array of strings specifying labels which must be associated with the required service.
  
   - This field is optional.
  
   - These tags have no semantic meaning within the Forklift package specification, but tag requirements can be used for arbitrary purposes. For example, tags can be used to require a network service annotated with information about specific API versions, subprotocols, etc. If a package deployment specifies that it requires a network service with one or more tags, then another package deployment will only be considered to satisfy the network service dependency if it provides a network service matching both the required route and all required tags. This is useful in ensuring that a network service provided by one package deployment is compatible with the API version required by a service client from another package deployment, for example.
  
   - Example:
     
     ```yaml
     tags:
       - mjpeg-stream
     ```

- `nonblocking` is a boolean flag specifying whether the package deployment providing the required service is allowed to start after starting the package deployment with the service requirement.
  
   - This field is optional.
  
   - This is a performance optimization hint which may be ignored; it's only meaningful if package deployments can be started concurrently. However, it can help to reduce the startup time needed for the critical path of a chain of dependencies between package deployments.
  
   - This field can be set to true if the service client can gracefully handle the temporary absence of the service while package deployments are being applied; otherwise, this field should not be set to true.
  
   - Example:
     
     ```yaml
     nonblocking: true
     ```

##### `filesets` field

This optional field of the `requires` subsection is an array of fileset objects listing the files (which can include directories) which must be available on the Docker host in order for a deployment of the package to successfully become active.

- This field is optional.

- A fileset requirement is defined a list of one or more paths. A fileset requirement will be satisfied by one or more filesets if and only if the set union of the paths of the filesets overlaps with every path listed in the fileset requirement (for a definition of overlapping paths, refer below to description of the `path` field of the network service object). Thus, in any particular set of package deployments, one fileset from one package deployment may be sufficient to satisfy a fileset requirement from some other package deployment, or multiple filesets from multiple packages may be necessary to fully satisfy that fileset requirement.

- If a set of package deployments contains a fileset resource requirement with a path which does not overlap with the paths of any filesets provided by other package deployments, then the package deployment declaring that fileset requirement will be reported as having an unmet dependency. Therefore, the overall set of package deployments will not be allowed because its [package resource constraints](#package-resource-constraints) for resource dependencies will not be satisfied.

- Example:
  
  ```yaml
  filesets:
    - description: The directory where logs will be saved
      tags:
        - directory
        - owner-1000
        - writable
      paths:
        - /home/pi/device-backend-logs/processing/segmenter
  ```

A fileset object consists of the following fields:

- `description` is a short (one-sentence) description of the fileset resource requirement to be shown to users.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     description: A file containing the device's machine name
     ```

- `paths` is an array of strings which are paths of files which must exist.
  
   - This field is required.
  
   - A path may optionally have an asterisk (`*`) at the end, in which case it is a prefix path - so the required fileset must declare that it can be used with any path beginning with that prefix (i.e. the string before the asterisk).
  
   - If a package deployment has a requirement for a fileset with a specific path which matches the prefix path of a fileset provided by another package deployment, the fileset requirement will be met. For example, a requirement for a fileset with a path `/var/lib/planktoscope/machine-name` would be met by a fileset provided with the path prefix `/var/lib/planktoscope/*`.
  
   - Example:
     
     ```yaml
     paths:
       - /var/lib/planktoscope/machine-name
     ```

- `tags` is an array of strings specifying labels which must be associated with the required fileset.
  
   - This field is optional.
  
   - These tags have no semantic meaning within the Forklift package specification, but tag requirements can be used for arbitrary purposes. For example, tags can be used to require a fileset annotated with information about file types, file permissions, schema versions, etc. If a package deployment specifies that it requires a fileset with one or more tags, then another package deployment will only be considered to satisfy the fileset dependency if it provides a fileset matching both the required path(s) and all required tags. This is useful in ensuring that a fileset provided by one package deployment is compatible with the schema version required by another package deployment, for example.
  
   - Example:
     
     ```yaml
     tags:
       - file
       - plain-text
     ```

- `nonblocking` is a boolean flag specifying whether the package deployment providing the required fileset is allowed to start after starting the package deployment with the fileset requirement.
  
   - This field is optional.
  
   - This is a performance optimization hint which may be ignored; it's only meaningful if package deployments can be started concurrently. However, it can help to reduce the startup time needed for the critical path of a chain of dependencies between package deployments.
  
   - This field can be set to true if the program requiring the fileset can gracefully handle the temporary absence of the fileset while package deployments are being applied; otherwise, this field should not be set to true.
  
   - Example:
     
     ```yaml
     nonblocking: true
     ```

#### `provides` subsection

This optional subsection of the `deployment` section specifies the resources provided by an active deployment of the package. This is the same as the `provides` subsection of the `host` section, except that here the resources only exist when a package deployment is active. Here is an example of a `provides` section:

```yaml
provides:
  listeners:
    - description: MQTT broker
      port: 1883
      protocol: mqtt
  services:
    - description: MQTT broker for the PlanktoScope backend's MQTT API
      tags: [mqtt-broker]
      port: 1883
      protocol: mqtt
```

##### `listeners` field

This optional field of the `provides` subsection is an array of host port listener objects listing the network port/protocol pairs which will be bound to processes running in an active deployment of the package and listening for incoming traffic on those port/protocol pairs, on any/all IP addresses.

- This field is optional.

- Generally, a host port listener object should correspond to a [published port](https://docs.docker.com/network/#published-ports) of a Docker container.

- Each host port listener object describes a host port listener resource which may or may not be in conflict with other host port listener resources; this is because multiple processes are not allowed to simultaneously bind to the same port/protocol pair on all IP addresses.

- If a set of package deployments contains two or more host port listener resources for the same port/protocol pair from different package deployments, the package deployments declaring those respective host port listeners will be reported as conflicting with each other. Therefore, the overall set of package deployments will not be allowed because its [package resource constraints](#package-resource-constraints) for uniqueness of host port listener resources will not be satisfied.

- Currently, this specification does not allow multiple host port listeners to bind to the same port/protocol pair on different IP addresses; instead for simplicity, processes are assumed to be listening for that port/protocol pair on *all* IP addresses on the host.

- Example:
  
  ```yaml
  listeners:
    - description: MQTT broker
      port: 1883
      protocol: mqtt
  ```

A host port listener object consists of the following fields:

- `description` is a short (one-sentence) description of the host port listener resource to be shown to users.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     description: Web server for all HTTP requests
     ```

- `port` is a number specifying the [network port](https://en.wikipedia.org/wiki/Port_(computer_networking)) bound by a process running on the host.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     port: 80
     ```

- `protocol` is a string specifying whether the bound network port is for the TCP transport protocol or for the UDP transport protocol.
  
   - This field is required.
  
   - The value of this field must be either `tcp` or `udp`.
  
   - Example:
     
     ```yaml
     protocol: tcp
     ```

##### `networks` field

This optional field of the `provides` subsection is an array of Docker network objects listing the Docker networks which are created when a deployment of the package becomes active.

- This field is optional.

- Each host Docker network object describes a Docker network resource which may or may not be in conflict with other Docker network resources; this is because multiple Docker networks are not allowed to have the same name.

- If a set of package deployments contains two or more Docker network resources for networks with the same name from different package deployments, the package deployments declaring those respective Docker networks will be reported as conflicting with each other. Therefore, the overall set of package deployments will not be allowed because its [package resource constraints](#package-resource-constraints) for uniqueness of host Docker network names will not be satisfied.

- Example:
  
  ```yaml
  networks:
    - description: Overlay network for Caddy to connect to upstream services
      name: caddy-ingress
  ```

A Docker network object consists of the following fields:

- `description` is a short (one-sentence) description of the Docker network resource to be shown to users.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     description: Overlay network for the Portainer server to connect to Portainer agents
     ```

- `name` is a string specifying the name of the Docker network.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     name: portainer-agent
     ```

##### `services` field

This optional field of the `provides` subsection is an array of network service objects listing the network services which are created when a deployment of the package becomes active.

- This field is optional.

- The route of a network service can be defined either as a port/protocol pair or as a combination of port, protocol, and one or more paths. A network service whose route is defined only as a port/protocol pair will overlap with another network service if and only if the other network service whose route is also defined only as a port/protocol pair. A network service whose route is defined with one or more paths will overlap with another network service if and only if both network services have the same port, the same protocol, and at least one overlapping path (for a definition of overlapping paths, refer below to description of the `path` field of the network service object).

- Each network service object describes a network service resource which may or may not be in conflict with other network service resources; this is because multiple network services are not allowed to have overlapping routes.

- If a set of package deployments contains two or more network service resources for services with overlapping routes from different package deployments, then the package deployments declaring those respective network services will be reported as conflicting with each other. Therefore, the overall set of package deployments will not be allowed because its [package resource constraints](#package-resource-constraints) for uniqueness of network services will not be satisfied.

- Example:
  
  ```yaml
  services:
    - description: MJPEG stream of last segmented object from the PlanktoScope object segmenter
      tags: [mjpeg-stream]
      port: 8001
      protocol: http
      paths:
        - /
        - /stream.mjpg
    - description: MQTT handling of segmenter commands and broadcasting of segmenter statuses
      tags: [planktoscope-api-v2]
      port: 1883
      protocol: mqtt
      paths:
        - /segmenter/segment
        - /status/segmenter
        - /status/segmenter/name
        - /status/segmenter/object_id
        - /status/segmenter/metric
  ```

A network service object consists of the following fields:

- `description` is a short (one-sentence) description of the network service resource to be shown to users.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     description: PlanktoScope documentation site
     ```

- `port` is a number specifying the network port used for accessing the service.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     port: 80
     ```

- `protocol` is a string specifying the application-level protocol used for accessing the service.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     protocol: http
     ```

- `paths` is an array of strings which are paths used for accessing the service.
  
   - This field is optional.
  
   - A path may optionally have an asterisk (`*`) at the end, in which case it is a prefix path - so the network service covers all paths beginning with that prefix (i.e. the string before the asterisk).
  
   - If a network service specifies a port and protocol but no paths, it will conflict with another network service which also specifies the same port and protocol but no paths; it will not conflict with another network service which specifies the same port and protocol and also specifies some paths. In other words, not listing any paths in a network service is equivalent to not having any conflicts with other services available at specific paths on the same port and protocol.
     This is useful for describing systems involving HTTP reverse-proxies or involving message brokers, where one package deployment may provide a network service which routes specific messages to network services from other package deployments on specific paths; then the reverse-proxy or message broker would be specified on some port and protocol with no paths, while the network services behind it would be specified on the same port and protocol but with a set of specific paths.
  
   - If a package deployment has a dependency on a network service with a specific path which matches a prefix path in a network service from another package deployment, that dependency will be satisfied. For example, a dependency on a network service requiring a path `/ps/docs/hardware` would be met by a network service provided with the path prefix `/ps/docs/*`, assuming they have the same port and protocol.
  
   - If a package deployment provides a network service with a specific path which matches a prefix path in a network service provided by another package deployment, those two package deployments will be in conflict with each other. For example, a network service providing a path `/ps/docs/hardware` would conflict with a network service providing the path prefix `/ps/docs/*`, assuming they have the same port and protocol. This is because those overlapping paths would cause the network services to overlap with each other, which is not allowed.
  
   - Example:
     
     ```yaml
     paths:
       - /ps/docs
       - /ps/docs/*
     ```

- `tags` is an array of strings which constrain resolution of network service resource dependencies among package deployments. These tags are ignored in determining whether network services conflict with each other, since they are not part of the network service's route.
  
   - This field is optional.
  
   - These tags have no semantic meaning within the Forklift package specification, but tag requirements can be used for arbitrary purposes. For example, tags can be used to annotate a network service with information about API versions, subprotocols, etc. If a package deployment specifies that it requires a network service with one or more tags, then another package deployment will only be considered to satisfy the network service dependency if it provides a network service matching both the required route and all required tags. This is useful in ensuring that a network service provided by one package deployment is compatible with the API version required by a service client from another package deployment, for example.
  
   - Example:
     
     ```yaml
     tags:
       - website
     ```

##### `filesets` field

This optional field of the `provides` subsection is an array of fileset objects listing the files (which can include directories) which are created when a deployment of the package becomes active.

- This field is optional.

- A fileset is defined as a list of one or more paths. A fileset will overlap with another fileset if and only if both filesets have at least one overlapping path (for a definition of overlapping paths, refer below to description of the `path` field of the fileset object).

- Each fileset object describes a fileset resource which may or may not be in conflict with other fileset resources; this is because multiple filesets are not allowed to have overlapping paths.

- If a set of package deployments contains two or more fileset resources for filesets with overlapping paths from different package deployments, then the package deployments declaring those respective filesets will be reported as conflicting with each other. Therefore, the overall set of package deployments will not be allowed because its [package resource constraints](#package-resource-constraints) for uniqueness of filesets will not be satisfied.

- Example:
  
  ```yaml
  services:
    - description: File containing the device's machine name
      tags:
        - file
        - plain-text
      paths:
        - /var/lib/planktoscope/machine-name
  ```

A fileset object consists of the following fields:

- `description` is a short (one-sentence) description of the fileset resource to be shown to users.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     description: Directory of EcoTaxa export archives
     ```

- `paths` is an array of strings which are paths of files in the fileset.
  
   - This field is required.
  
   - A path may optionally have an asterisk (`*`) at the end, in which case it is a prefix path - so the fileset covers all paths beginning with that prefix (i.e. the string before the asterisk).
  
   - If a package deployment has a dependency on a fileset with a specific path which matches a prefix path in a fileset from another package deployment, that dependency will be satisfied. For example, a dependency on a fileset requiring a path `/home/pi/device-logs/controller` would be met by a network service provided with the path prefix `/home/pi/device-logs/*`.
  
   - If a package deployment provides a fileset with a specific path which matches a prefix path in a fileset provided by another package deployment, those two package deployments will be in conflict with each other. For example, a fileset providing a path `/home/pi/data/export/ecotaxa` would conflict with a fileset providing the path prefix `/home/pi/data/export/*`. This is because those overlapping paths would cause the filesets to overlap with each other, which is not allowed.
  
   - Example:
     
     ```yaml
     paths:
       - /home/pi/data/export/ecotaxa
     ```

- `tags` is an array of strings which constrain resolution of fileset resource dependencies among package deployments. These tags are ignored in determining whether filesets conflict with each other, since they are not part of the fileset's location.
  
   - This field is optional.
  
   - These tags have no semantic meaning within the Forklift package specification, but tag requirements can be used for arbitrary purposes. For example, tags can be used to annotate a file with information about file type, file permissions, schema versions, etc. If a package deployment specifies that it requires a fileset with one or more tags, then another package deployment will only be considered to satisfy the fileset dependency if it provides a fileset matching both the required path(s) and all required tags. This is useful in ensuring that a fileset provided by one package deployment is compatible with the schema version required by another package deployment, for example.
  
   - Example:
     
     ```yaml
     tags:
       - directory
       - owner-1000
       - writable
     ```

##### `file-exports` field

This optional field of the `provides` subsection is an array of file export objects, each specifying a file provided by the package which should be exported to a particular path.

- This field is optional.

- A file export is defined as a source path and a target path. A file export will overlap with another file export if and only if both file exports have overlapping target paths (for a definition of overlapping paths, refer below to description of the `path` field of the file export object).

- Each file export object describes a file export resource which may or may not be in conflict with other file export resources; this is because multiple file exports are not allowed to have overlapping target paths.

- If a set of package deployments contains two or more file export resources for file exports with overlapping target paths from different package deployments, then the package deployments declaring those respective file exports will be reported as conflicting with each other. Therefore, the overall set of package deployments will not be allowed because its [package resource constraints](#package-resource-constraints) for uniqueness of file exports will not be satisfied.

- Example:
  
  ```yaml
  file-exports:
    - description: Systemd service definition
      tags:
        - systemd-unit
        - systemd-service
        - networking
      target: overlays/etc/systemd/system/enable-interface-forwarding.service
    - description: Symlink to enable the systemd service
      tags:
        - systemd-symlink
      target: overlays/etc/systemd/system/network-online.target.wants/enable-interface-forwarding.service
  ```

A file export object consists of the following fields:

- `description` is a short (one-sentence) description of the file export resource to be shown to users.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     description: Basic dnsmasq configuration
     ```

- `source-type` is the way that the source file is provided for export.
  
   - This field is optional: if it's not specified, it's assumed to be of type `local`.
  
   - Allowed values are:
     
      - `local`: the file is provided in the package's directory, or in a subdirectory of the package.
     
      - `http`: the file is downloaded from an HTTP/HTTPS URL.
  
   - Example:
     
     ```yaml
     source-type: local
     ```

- `source` is the filesystem path of the file to be exported; the meaning of this field varies depending on the value of `source-type`.
  
   - This field is optional: if it's not specified, it's assumed to be the path set by the `target` field.
  
   - For the `local` source type, the source path is interpreted as being relative to the path of the package.
  
   - For the `http` source type, the source path is ignored.
  
   - Example:
     
     ```yaml
     source: dhcp-and-dns.conf
     ```

- `url` is the URL of the file to be downloaded for export; the meaning of this field varies depending on the value of `source-type`.
  
   - This field is required for the `http` source type and ignored for the `local` source type.
  
   - For the `http` source type, the URL should be of the file which is downloaded and directly exported as a file.
  
   - Example:
     
     ```yaml
     url: https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-linux-amd64
     ```

- `target` is the path where the file should be exported to (e.g. by copying the file to that path), relative to an export directory defined by the tool which implements the Forklift packaging specification.
  
   - This field is required.
  
   - If a package deployment provides a file export with a specific target path which is identical to - or a parent directory of - the target path of a file export provided by another package deployment, those two package deployments will be in conflict with each other. For example, a file export to a target file named `overlays/etc/dnsmasq.d/dhcp-and-dns.conf` (which ensures that `overlays/etc/dnsmasq.d` is a directory) would conflict with a file export to a target file named `overlays/etc/dnsmasq.d` (which may cause `overlays/etc/dnsmasq.d` to be a non-directory file). This is because those overlapping paths would cause the file exports to overlap with each other, which is not allowed.
  
   - Example:
     
     ```yaml
     paths:
       - overlays/etc/dnsmasq.d/dhcp-and-dns.conf
     ```

- `tags` is an array of strings which describe the file export. These tags are ignored in determining whether file exports conflict with each other, since they are not part of the file export's location(s).
  
   - This field is optional.
  
   - These tags have no semantic meaning within the Forklift package specification, but tag requirements can be used for arbitrary purposes. For example, tags can be used to annotate a file with information about file type, file permissions, schema versions, etc.
  
   - Example:
     
     ```yaml
     tags:
       - drop-in-config
       - hostapd
     ```

### `features` section

This optional section of the `forklift-package.yml` file specifies the optional features which can be enabled for a deployment of the package, as well as any resources required for each enabled feature, as well as any resources provided by each enabled feature. If resource requirements of any enabled feature are not met, the deployment will not be allowed; resources provided by an enabled feature in a deployment of the package will only exist once the package deployment is successfully applied.

The `features` section is a map (i.e. dictionary) whose keys are feature names and whose values are feature specification objects.  Here is an example of a `features` section:

```yaml
features:
  editor:
    description: Provides access to the Node-RED admin editor for modifying the GUI
    compose-files: [compose-editor.yml]
    tags:
    - device-portal.name=Node-RED dashboard editor
    - device-portal.description=Provides a Node-RED flow editor to modify the Node-RED dashboard
    - device-portal.type=Browser applications
    - device-portal.purpose=Software development
    - device-portal.entrypoint=/admin/ps/node-red-v2/
    requires:
      networks:
        - description: Overlay network for Caddy to connect to upstream services
          name: caddy-ingress
      services:
        - tags: [caddy-docker-proxy]
          port: 80
          protocol: http
        - port: 1880
          protocol: http
          paths:
            - /admin/ps/node-red-v2/*
    provides:
      services:
        - description: The Node-RED editor for the v2 PlanktoScope dashboard
          port: 80
          protocol: http
          paths:
            - /admin/ps/node-red-v2
            - /admin/ps/node-red-v2/*
  frontend:
    description: Provides access to the GUI
    compose-files: [compose-frontend.yml]
    tags:
    - device-portal.name=Node-RED dashboard
    - device-portal.description=Provides a Node-RED dashboard to operate the PlanktoScope
    - device-portal.type=Browser applications
    - device-portal.purpose=PlanktoScope operation
    - device-portal.entrypoint=/ps/node-red-v2/ui/
    requires:
      networks:
        - description: Overlay network for Caddy to connect to upstream services
          name: caddy-ingress
      services:
        - tags: [caddy-docker-proxy]
          port: 80
          protocol: http
        - port: 1880
          protocol: http
          paths:
            - /ps/node-red-v2/ui/*
        - tags: [mjpeg-stream]
          port: 80
          protocol: http
          paths:
            - /ps/hal/camera/streams/preview.mjpg
            - /ps/processing/segmenter/streams/object.mjpg
    provides:
      services:
        - description: The v2 PlanktoScope dashboard for configuring the PlanktoScope and collecting data
          port: 80
          protocol: http
          paths:
            - /ps/node-red-v2/ui
            - /ps/node-red-v2/ui/*
```

A feature specification object consists of the following fields:

- `description` is a short (one-sentence) description of the network service resource requirement to be shown to users.
  
   - This field is required.
  
   - Example:
     
     ```yaml
     description: Provides access to the GUI
     ```

- `compose-files` is an array of the string filenames of one or more Docker Compose files specifying modifications to the Docker Compose application which will be applied if the feature is enabled.
  
   - This field is optional.
  
   - The filenames must be for YAML files which are fragments of a [Docker Compose file specification](https://docs.docker.com/compose/compose-file/). These files will be merged together with any other Compose files specified in the [`deployment` section](#deployment-section) of the `forklift-package.yml` file and in any other enabled features according to Docker Compose's [compose file merging mechanism](https://docs.docker.com/compose/multiple-compose-files/merge/).
  
   - The files must be located in the same directory as the `forklift-package.yml` file, or in subdirectories.
  
   - For clarity, it is strongly recommended that the order in which the files for different feature flags are merged should not affect the final result of merging.
  
   - Example:
     
     ```yaml
     compose-files: [compose-frontend.yml]
     ```

- `tags` is an array of strings to associate with the feature or with resources required or provided by the feature.
  
   - This field is optional.
  
   - These tags have no semantic meaning within the Forklift package specification, but can be used by other applictions for arbitrary purposes.
  
   - Example:
     
     ```yaml
     tags:
       - device-portal-name=Portainer
       - device-portal-description=Provides a Docker administration dashboard
       - device-portal-type=Browser applications
       - device-portal-purpose=System administration and troubleshooting
       - device-portal-entrypoint=/admin/portainer/
     ```

- `requires` is a specification of resources required for a deployment of the package, with the feature enabled, to successfully become active.
  
   - This field is optional.
  
   - The contents of this field have the same syntax and semantics as the contents of the [`requires` subsection](#requires-subsection) of the `deployment` section of the Forklift package specification, except that resource requirements are only evaluated for features configured as "enabled" for each deployment of each package; resource requirements for disabled features will be ignored.
  
   - Example:
     
     ```yaml
     requires:
       networks:
         - description: Overlay network for Caddy to connect to upstream services
           name: caddy-ingress
       services:
         - tags: [caddy-docker-proxy]
           port: 80
           protocol: http
     ```

- `provides` is a specification of resources provided by a deployment of the package, if the feature is enabled for that deployment.
  
   - This field is optional.
  
   - The contents of this field have the same syntax and semantics as the contents of the [`provides` subsection](#provides-subsection) of the `deployment` section of the Forklift package specification, except that provided resources are only considered for features configured as "enabled" for each deployment of each package; provided resources for disabled features will be ignored.
  
   - Example:
     
     ```yaml
     provides:
       services:
         - description: The Portainer Docker management dashboard
           port: 80
           protocol: http
           paths:
             - /admin/portainer
             - /admin/portainer/*
     ```
