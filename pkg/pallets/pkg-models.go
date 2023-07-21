package pallets

// A Pkg is a Pallet package.
type Pkg struct {
	// Path is the Pallet repository path of the Pallet repository path which provides the package.
	RepoPath string
	// Subdir is the path of the package within the Pallet repository which provides the package.
	Subdir string
	// Config is the Pallet package specification for the package.
	Config PkgConfig
}

// A FSPkg is a Pallet package stored at the root of a [fs.FS] filesystem.
type FSPkg struct {
	// Pkg is the Pallet package at the root of the filesystem.
	Pkg
	// FS is a filesystem which contains the package's contents.
	FS PathedFS
}

// PkgSpecFile is the name of the file defining each Pallet package.
const PkgSpecFile = "pallet-package.yml"

// A PkgConfig defines a Pallet package.
type PkgConfig struct {
	// Repository defines the basic metadata for the package.
	Package PkgSpec `yaml:"package,omitempty"`
	// Host contains information about the Docker host independent of any deployment of the package.
	Host PkgHostSpec `yaml:"host,omitempty"`
	// Deployment contains information about any deployment of the package.
	Deployment PkgDeplSpec `yaml:"deployment,omitempty"`
	// Features contains optional features which can be enabled or disabled.
	Features map[string]PkgFeatureSpec `yaml:"features,omitempty"`
}

// PkgSpec defines the basic metadata for a Pallet package.
type PkgSpec struct {
	// Description is a short description of the package to be shown to users.
	Description string `yaml:"description"`
	// Maintainers is a list of people who maintain the package.
	Maintainers []PkgMaintainer `yaml:"maintainers,omitempty"`
	// License is an SPDX 2.1 license expression specifying the licensing terms of the software
	// provided by the package.
	License string `yaml:"license"`
	// LicenseFile is the name of a license file describing the licensing terms of the software
	// provided by the package.
	LicenseFile string `yaml:"license-file,omitempty"`
	// Sources is a list of URLs providing the source code of the software provided by the package.
	Sources []string `yaml:"sources,omitempty"`
}

// PkgMaintainer describes a maintainer of a Pallet package.
type PkgMaintainer struct {
	// Name is the maintainer's name.
	Name string `yaml:"name,omitempty"`
	// Email is an email address for contacting the maintainer.
	Email string `yaml:"email,omitempty"`
}

// PkgHostSpec contains information about the Docker host independent of any deployment of the
// Pallet package.
type PkgHostSpec struct {
	// Tags is a list of strings associated with the host.
	Tags []string `yaml:"tags,omitempty"`
	// Provides describes resources ambiently provided by the Docker host.
	Provides ProvidedResources `yaml:"provides,omitempty"`
}

// PkgDeplSpec contains information about any deployment of the Pallet package.
type PkgDeplSpec struct {
	// DefinitionFile is the name of a Docker Compose file specifying the Docker stack which will be
	// deployed as part of a package deployment.
	DefinitionFile string `yaml:"definition-file,omitempty"`
	// Tags is a list of strings associated with the deployment.
	Tags []string `yaml:"tags,omitempty"`
	// Provides describes resource requirements which must be met for a deployment of the package to
	// succeed.
	Requires RequiredResources `yaml:"requires,omitempty"`
	// Provides describes resources provided by a successful deployment of the package.
	Provides ProvidedResources `yaml:"provides,omitempty"`
}

// PkgFeatureSpec defines an optional feature of the Pallet package.
type PkgFeatureSpec struct {
	// Description is a short description of the feature to be shown to users.
	Description string `yaml:"description"`
	// Tags is a list of strings associated with the feature.
	Tags []string `yaml:"tags,omitempty"`
	// Provides describes resource requirements which must be met for a deployment of the package to
	// succeed, if the feature is enabled.
	Requires RequiredResources `yaml:"requires,omitempty"`
	// Provides describes resources provided by a successful deployment of the package, if the feature
	// is enabled.
	Provides ProvidedResources `yaml:"provides,omitempty"`
}

// Resources

// RequiredResources describes a set of resource requirements for some aspect of a Pallet package.
type RequiredResources struct {
	// Networks is a list of requirements for Docker networks.
	Networks []NetworkResource `yaml:"networks,omitempty"`
	// Services is a list of requirements for network services.
	Services []ServiceResource `yaml:"services,omitempty"`
}

// ProvidedResources describes a set of resources provided by some aspect of a Pallet package.
type ProvidedResources struct {
	// Listeners is a list of host port listeners.
	Listeners []ListenerResource `yaml:"listeners,omitempty"`
	// Networks is a list of Docker networks.
	Networks []NetworkResource `yaml:"networks,omitempty"`
	// Services is a list of network services.
	Services []ServiceResource `yaml:"services,omitempty"`
}

// ListenerResource describes a host port listener.
type ListenerResource struct {
	// Description is a short description of the host port listener to be shown to users.
	Description string `yaml:"description,omitempty"`
	// Port is the port number which the host port listener is bound to.
	Port int `yaml:"port,omitempty"`
	// Protocol is the transport protocol (either tcp or udp) which the host port listener is bound
	// to.
	Protocol string `yaml:"protocol,omitempty"`
}

// NetworkResource describes a Docker network.
type NetworkResource struct {
	// Description is a short description of the Docker network to be shown to users.
	Description string `yaml:"description,omitempty"`
	// Name is the name of the Docker network.
	Name string `yaml:"name,omitempty"`
}

// ServiceResource describes a network service.
type ServiceResource struct {
	// Description is a short description of the network service to be shown to users.
	Description string `yaml:"description,omitempty"`
	// Port is the network port used for accessing the service.
	Port int `yaml:"port,omitempty"`
	// Protocol is the application-level protocol (e.g. http or mqtt) used for accessing the service.
	Protocol string `yaml:"protocol,omitempty"`
	// Tags is a list of strings associated with the service. Tags are considered in determining which
	// service resources meet service resource requirements.
	Tags []string `yaml:"tags,omitempty"`
	// Paths is a list of paths used for accessing the service. A path may also be a prefix, indicated
	// by ending the path with an asterisk (`*`).
	Paths []string `yaml:"paths,omitempty"`
}
