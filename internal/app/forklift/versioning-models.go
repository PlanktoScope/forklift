package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

const (
	// RepoRequirementsDirName is the directory in a Forklift environment which contains Pallet
	// repository requirement configurations.
	// TODO: move repositories to requirements/repositories, to allow for a future
	// requirements/environments subdirectory
	RepoRequirementsDirName = "repositories"
	// RepoRequirementFileExt is the file name for versioned repositories.
	// TODO: rename this to "forklift-version-requirement.yml"
	RepoRequirementSpecFile = "forklift-repo.yml"
)

// Repos

// TODO: make a FSRepoRequirement which has a getter for the version requirement

// A RepoVersionRequirement is a requirement for a Pallet repository in an environment.
type RepoVersionRequirement struct {
	VCSRepoPath string
	RepoSubdir  string
	Config      VersionLockConfig
}

// A VersionLockConfig defines a requirement for a Pallet repository or package at a specific
// version.
type VersionLockConfig struct {
	BaseVersion string `yaml:"base-version,omitempty"`
	Timestamp   string `yaml:"timestamp,omitempty"`
	Commit      string `yaml:"commit,omitempty"`
}

// Pkgs

type VersionedPkg struct {
	*pallets.FSPkg
	RepoVersionRequirement RepoVersionRequirement
}
