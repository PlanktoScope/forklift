// Package forklift provides the core functionality of the forklift tool
package forklift

// Pallet repository specifications

type RepoSpec struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

type RepoConfig struct {
	Repository RepoSpec `yaml:"repository"`
}

// Pallet package specifications

type PkgConfig struct {
	Package    PkgSpec                   `yaml:"package"`
	Host       PkgHostSpec               `yaml:"host"`
	Deployment PkgDeplSpec               `yaml:"deployment"`
	Features   map[string]PkgFeatureSpec `yaml:"features"`
}

type PkgMaintainer struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

type PkgSpec struct {
	Description string          `yaml:"description"`
	Maintainers []PkgMaintainer `yaml:"maintainers"`
	License     string          `yaml:"license"`
	LicenseFile string          `yaml:"license-file"`
	Sources     []string        `yaml:"sources"`
}

type ProvidedResources struct {
	Listeners []ListenerResource `yaml:"listeners"`
	Networks  []NetworkResource  `yaml:"networks"`
	Services  []ServiceResource  `yaml:"services"`
}

type RequiredResources struct {
	Networks []NetworkResource `yaml:"networks"`
	Services []ServiceResource `yaml:"services"`
}

type ListenerResource struct {
	Description string `yaml:"description"`
	Port        int    `yaml:"port"`
	Protocol    string `yaml:"protocol"`
}

type NetworkResource struct {
	Description string `yaml:"description"`
	Name        string `yaml:"name"`
}

type ServiceResource struct {
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	Port        int      `yaml:"port"`
	Protocol    string   `yaml:"protocol"`
	Paths       []string `yaml:"paths"`
}

type PkgHostSpec struct {
	Provides ProvidedResources `yaml:"provides"`
}

type PkgDeplSpec struct {
	Name           string            `yaml:"name"`
	DefinitionFile string            `yaml:"definition-file"`
	Requires       RequiredResources `yaml:"requires"`
	Provides       ProvidedResources `yaml:"provides"`
}

type PkgFeatureSpec struct {
	Description string            `yaml:"description"`
	Requires    RequiredResources `yaml:"requires"`
	Provides    ProvidedResources `yaml:"provides"`
}

// Repository versioning

type VersionedRepo struct {
	VCSRepoPath string
	RepoSubdir  string
	Config      RepoVersionConfig
	Lock        RepoVersionLock
}

type RepoVersionConfig struct {
	Release string `yaml:"release"`
}

type RepoVersionLock struct {
	Version   string `yaml:"version"`
	Timestamp string `yaml:"timestamp"`
	Commit    string `yaml:"commit"`
}

// Repository caching

type CachedRepo struct {
	VCSRepoPath string
	Version     string
	RepoSubdir  string
	ConfigPath  string
	Config      RepoConfig
}

// Package versioning

type VersionedPkg struct {
	Path   string
	Repo   VersionedRepo
	Cached CachedPkg
}

// Package caching

type CachedPkg struct {
	Repo       CachedRepo
	Path       string
	PkgSubdir  string
	ConfigPath string
	Config     PkgConfig
}

// Deployments

type DeplConfig struct {
	Package  string   `yaml:"package"`
	Features []string `yaml:"features"`
}

type Depl struct {
	Name   string
	Config DeplConfig
	Pkg    VersionedPkg
}
