package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Repos

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

// Pkgs

type VersionedPkg struct {
	*pallets.FSPkg
	VersionedRepo VersionedRepo
}
