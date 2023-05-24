// Package forklift provides the core functionality of the forklift tool
package forklift

// Environment specifications

type EnvSpec struct {
	Description string `yaml:"description"`
}

type EnvConfig struct {
	Environment EnvSpec `yaml:"environment"`
}

// Repo versioning specifications

type VersionedRepo struct {
	VCSRepoPath string
	RepoSubdir  string
	Config      RepoVersionConfig
}

type RepoVersionConfig struct {
	BaseVersion string `yaml:"base-version"`
	Timestamp   string `yaml:"timestamp"`
	Commit      string `yaml:"commit"`
}

// Package deployment specifications

type DeplConfig struct {
	Package  string   `yaml:"package"`
	Features []string `yaml:"features"`
}

type Depl struct {
	Name   string
	Config DeplConfig
	Pkg    VersionedPkg
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
