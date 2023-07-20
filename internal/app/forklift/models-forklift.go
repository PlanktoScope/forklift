// Package forklift provides the core functionality of the forklift tool
package forklift

import (
	"io/fs"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Environment specifications

type EnvSpec struct {
	Description string `yaml:"description,omitempty"`
}

type EnvConfig struct {
	Environment EnvSpec `yaml:"environment,omitempty"`
}

// Repo versioning specifications

type VersionedRepo struct {
	VCSRepoPath string
	RepoSubdir  string
	Config      RepoVersionConfig
}

type RepoVersionConfig struct {
	BaseVersion string `yaml:"base-version,omitempty"`
	Timestamp   string `yaml:"timestamp,omitempty"`
	Commit      string `yaml:"commit,omitempty"`
}

// Package deployment specifications

type DeplConfig struct {
	Package  string   `yaml:"package,omitempty"`
	Features []string `yaml:"features,omitempty"`
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
	Config      pallets.RepoConfig
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
	Config     pallets.PkgConfig
}

// External repository loading

type ExternalRepo struct {
	Repo CachedRepo
	FS   fs.FS
}
