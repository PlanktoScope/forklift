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
	// RepoReqsDirName is the directory in a Forklift environment which contains Pallet
	// repository requirement configurations.
	// TODO: move repositories to requirements/repositories, to allow for a future
	// requirements/environments subdirectory.
	RepoReqsDirName = "repositories"
)

// A FSRepoReq is a Pallet repository requirement stored at the root of a [fs.FS]
// filesystem.
type FSRepoReq struct {
	// RepoReq is the Pallet repository requirement at the root of the filesystem.
	RepoReq
	// FS is a filesystem which contains the repository requirement's contents.
	FS pallets.PathedFS
}

// A RepoReq is a requirement for a specific Pallet repository at a specific version.
type RepoReq struct {
	// VCSRepoPath is the VCS repository path of the required repository.
	VCSRepoPath string
	// RepoSubdir is the Pallet repository subdirectory of the required repository.
	RepoSubdir string
	// VersionLock specifies the version of the required repository.
	VersionLock VersionLock
}

// Package Requirements

// A PkgReq is a requirement for a Pallet package at a specific version.
type PkgReq struct {
	// PkgSubdir is the Pallet package subdirectory of the repository which should provide the
	// required package.
	PkgSubdir string
	// Repo is the Pallet repository which should provide the required package.
	Repo RepoReq
}

// PkgReqLoader is a source of Pallet package requirements.
type PkgReqLoader interface {
	LoadPkgReq(pkgPath string) (PkgReq, error)
}

// Deployment Configurations

const (
	// DeplsDirName is the directory in a Forklift environment which contains deployment
	// configurations.
	DeplsDirName = "deployments"
	// DeplsFileExt is the file extension for deployment configuration files.
	DeplSpecFileExt = ".deploy.yml"
)

// A ResolvedDepl is a deployment with a loaded package.
type ResolvedDepl struct {
	// Depl is the configured deployment of the package represented by Pkg.
	Depl
	// PkgReq is the package requirement for the deployment.
	PkgReq PkgReq
	// Pkg is the package to be deployed.
	Pkg *pallets.FSPkg
}

// A Depl is a Pallets package deployment, a complete configuration of how a Pallet package is to be
// deployed on a Docker host.
type Depl struct {
	// Name is the name of the package depoyment.
	Name string
	// Config is the Forklift package deployment specification for the deployment.
	Config DeplConfig
}

// A DeplConfig defines a Pallets package deployment.
type DeplConfig struct {
	// Package is the Pallet package path of the package to deploy
	Package string `yaml:"package,omitempty"`
	// Features is a list of features from the Pallet package which should be enabled in the
	// deployment.
	Features []string `yaml:"features,omitempty"`
}
