package core

import (
	"io/fs"

	ffs "github.com/forklift-run/forklift/pkg/fs"
)

// A FSPkg is a Forklift package stored at the root of a [fs.FS] filesystem.
type FSPkg struct {
	// Pkg is the Forklift package at the root of the filesystem.
	Pkg
	// FS is a filesystem which contains the package's contents.
	FS ffs.PathedFS
	// FSPkgTree is a pointer to the [FSPkgTree] instance which provides the package.
	FSPkgTree *FSPkgTree
}

// A Pkg is a Forklift package, a configuration of a software application which can be deployed on a
// Docker host.
type Pkg struct {
	// ParentPath is the path of the package tree which provides the package.
	ParentPath string
	// Subdir is the path of the package within the package tree which provides the package.
	Subdir string
	// Decl is the definition of the package.
	Decl PkgDecl
}

// PkgDeclFile is the name of the file defining each package.
const PkgDeclFile = "forklift-package.yml"

// A PkgDecl defines a package.
type PkgDecl struct {
	// Package defines the basic metadata for the package.
	Package PkgSpec `yaml:"package,omitempty"`
	// Host contains information about the Docker host independent of any deployment of the package.
	Host PkgHostSpec `yaml:"host,omitempty"`
	// Deployment contains information about any deployment of the package.
	Deployment PkgDeplSpec `yaml:"deployment,omitempty"`
	// Features contains optional features which can be enabled or disabled.
	Features map[string]PkgFeatureSpec `yaml:"features,omitempty"`
}

// PkgSpec defines the basic metadata for a package.
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

// PkgMaintainer describes a maintainer of a package.
type PkgMaintainer struct {
	// Name is the maintainer's name.
	Name string `yaml:"name,omitempty"`
	// Email is an email address for contacting the maintainer.
	Email string `yaml:"email,omitempty"`
}

// PkgHostSpec contains information about the Docker host independent of any deployment of the
// package.
type PkgHostSpec struct {
	// Tags is a list of strings associated with the host.
	Tags []string `yaml:"tags,omitempty"`
	// Provides describes resources ambiently provided by the Docker host.
	Provides ProvidedRes `yaml:"provides,omitempty"`
}

// PkgDeplSpec contains information about any deployment of the package.
type PkgDeplSpec struct {
	// ComposeFiles is a list of the names of Docker Compose files specifying the Docker Compose
	// application which will be deployed as part of a package deployment.
	ComposeFiles []string `yaml:"compose-files,omitempty"`
	// Tags is a list of strings associated with the deployment.
	Tags []string `yaml:"tags,omitempty"`
	// Requires describes resource requirements which must be met for a deployment of the package to
	// succeed.
	Requires RequiredRes `yaml:"requires,omitempty"`
	// Provides describes resources provided by a successful deployment of the package.
	Provides ProvidedRes `yaml:"provides,omitempty"`
}

// PkgFeatureSpec defines an optional feature of the package.
type PkgFeatureSpec struct {
	// Description is a short description of the feature to be shown to users.
	Description string `yaml:"description"`
	// ComposeFiles is a list of the names of Docker Compose files specifying the Docker Compose
	// application which will be merged together with any other Compose files as part of a package
	// deployment which enables the feature.
	ComposeFiles []string `yaml:"compose-files,omitempty"`
	// Tags is a list of strings associated with the feature.
	Tags []string `yaml:"tags,omitempty"`
	// Provides describes resource requirements which must be met for a deployment of the package to
	// succeed, if the feature is enabled.
	Requires RequiredRes `yaml:"requires,omitempty"`
	// Provides describes resources provided by a successful deployment of the package, if the feature
	// is enabled.
	Provides ProvidedRes `yaml:"provides,omitempty"`
}

// Resources

// RequiredRes describes a set of resource requirements for some aspect of a package.
type RequiredRes struct {
	// Networks is a list of requirements for Docker networks.
	Networks []NetworkRes `yaml:"networks,omitempty"`
	// Services is a list of requirements for network services.
	Services []ServiceRes `yaml:"services,omitempty"`
	// Filesets is a list of requirements for files/directories.
	Filesets []FilesetRes `yaml:"filesets,omitempty"`
}

// ProvidedRes describes a set of resources provided by some aspect of a package.
type ProvidedRes struct {
	// Listeners is a list of host port listeners.
	Listeners []ListenerRes `yaml:"listeners,omitempty"`
	// Networks is a list of Docker networks.
	Networks []NetworkRes `yaml:"networks,omitempty"`
	// Services is a list of network services.
	Services []ServiceRes `yaml:"services,omitempty"`
	// Filesets is a list of files/directories.
	Filesets []FilesetRes `yaml:"filesets,omitempty"`
	// FileExports is a list of files/directories.
	FileExports []FileExportRes `yaml:"file-exports,omitempty"`
}

// ListenerRes describes a host port listener.
type ListenerRes struct {
	// Description is a short description of the host port listener to be shown to users.
	Description string `yaml:"description,omitempty"`
	// Port is the port number which the host port listener is bound to.
	Port int `yaml:"port,omitempty"`
	// Protocol is the transport protocol (either tcp or udp) which the host port listener is bound
	// to.
	Protocol string `yaml:"protocol,omitempty"`
}

// NetworkRes describes a Docker network.
type NetworkRes struct {
	// Description is a short description of the Docker network to be shown to users.
	Description string `yaml:"description,omitempty"`
	// Name is the name of the Docker network.
	Name string `yaml:"name"`
}

// ServiceRes describes a network service.
type ServiceRes struct {
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
	// Nonblocking, when specified as a resource requirement, specifies that the client of the service
	// does not need to wait for the resource to exist before the client can start.
	Nonblocking bool `yaml:"nonblocking,omitempty"`
}

// FilesetRes describes a set of files/directories.
type FilesetRes struct {
	// Description is a short description of the fileset to be shown to users.
	Description string `yaml:"description,omitempty"`
	// Tags is a list of strings associated with the fileset. Tags are considered in determining which
	// fileset resources meet fileset resource requirements.
	Tags []string `yaml:"tags,omitempty"`
	// Paths is a list of paths where the fileset exists. A path may also be a prefix, indicated
	// by ending the path with an asterisk (`*`).
	Paths []string `yaml:"paths"`
	// Nonblocking, when specified as a resource requirement, specifies that the program requiring the
	// fileset does not need to wait for the fileset to exist before the program can start.
	Nonblocking bool `yaml:"nonblocking,omitempty"`
}

// FileExportRes describes a file exported by Forklift.
type FileExportRes struct {
	// Description is a short description of the file export to be shown to users.
	Description string `yaml:"description,omitempty"`
	// Tags is a list of strings associated with the file export. Tags are not considered in checking
	// resource constraints.
	Tags []string `yaml:"tags,omitempty"`
	// SourceType is either `local` (for a file in the package whose path is set by `Source`) or
	// `http` (for a file which needs to be downloaded from the URL set by `URL`).
	SourceType string `yaml:"source-type,omitempty"`
	// Source is the path in the package of the file to be exported, for a `local` source. If omitted,
	// the source path will be inferred from the Target path.
	Source string `yaml:"source,omitempty"`
	// URL is the URL of the file to be downloaded for export, for a `http` source.
	URL string `yaml:"url,omitempty"`
	// Permissions is the Unix permission bits to attach to the exported file.
	Permissions fs.FileMode `yaml:"permissions,omitempty"`
	// Target is the path where the file will be exported to, relative to an export directory.
	Target string `yaml:"target"`
}

const (
	FileExportSourceTypeLocal       = "local"
	FileExportSourceTypeHTTP        = "http"
	FileExportSourceTypeHTTPArchive = "http-archive"
	FileExportSourceTypeOCIImage    = "oci-image"
)
