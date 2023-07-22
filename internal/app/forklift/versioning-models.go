package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

const (
	// VersionedRepoDirName is the directory in a Forklift environment which contains versioned
	// repository configurations.
	VersionedReposDirName = "repositories"
	// VersionedRepoFileExt is the file name for versioned repositories.
	VersionedRepoSpecFile = "forklift-repo.yml"
)

// Repos

type VersionedRepo struct {
	VCSRepoPath string
	RepoSubdir  string
	Config      RepoVersionConfig
}

// A RepoVersionConfig defines a requirement for a Pallets repository at a specific version.
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
