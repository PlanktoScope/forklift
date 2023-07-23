// Package forklift provides the core functionality of the forklift tool
package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// A FSEnv is a Forklift environment configuration stored at the root of a [fs.FS] filesystem.
type FSEnv struct {
	// Env is the Forklift environment at the root of the filesystem.
	Env
	// FS is a filesystem which contains the environment's contents.
	FS pallets.PathedFS
}

// An Env is a Forklift environment, a complete specification of all Pallet package deployments
// which should be active on a Docker host.
type Env struct {
	// Config is the Forklift environment specification for the environment.
	Config EnvConfig
}

// EnvSpecFile is the name of the file defining each Forklift environment.
const EnvSpecFile = "forklift-env.yml"

// A EnvConfig defines a Forklift environment.
type EnvConfig struct {
	// Environment defines the basic metadata for the environment.
	Environment EnvSpec `yaml:"environment,omitempty"`
}

// EnvSpec defines the basic metadata for a Forklift environment.
type EnvSpec struct {
	// Description is a short description of the environment to be shown to users.
	Description string `yaml:"description,omitempty"`
}

// Repo Requirements

const (
	// RepoRequirementsDirName is the directory in a Forklift environment which contains Pallet
	// repository requirement configurations.
	// TODO: move repositories to requirements/repositories, to allow for a future
	// requirements/environments subdirectory
	RepoRequirementsDirName = "repositories"
)

// A FSRepoRequirement is a Pallet repository requirement stoed at the root of a [fs.FS] filesystem.
type FSRepoRequirement struct {
	// RepoRequirement is the Pallet repository requirement at the root of the filesystem.
	RepoRequirement
	// FS is a filesystem which contains the repository requirement's contents.
	FS pallets.PathedFS
}

// A RepoRequirement is a requirement for a Pallet repository in an environment.
type RepoRequirement struct {
	// VCSRepoPath is the VCS repository path of the required repository.
	VCSRepoPath string
	// RepoSubdir is the Pallet repository subdirectory of the required repository.
	RepoSubdir string
	// VersionLock specifies the version of the required repository.
	VersionLock VersionLock
}

// Package Requirements

// TODO: add a PkgRequirement type?

// TODO: rename this
type VersionedPkg struct {
	*pallets.FSPkg
	// TODO: can we replace this with a VersionLock?
	RepoRequirement *FSRepoRequirement
}
