// Package forklift provides the core functionality of the forklift tool
package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/core"
)

// A FSEnv is a Forklift environment configuration stored at the root of a [fs.FS] filesystem.
type FSEnv struct {
	// Env is the Forklift environment at the root of the filesystem.
	Env
	// FS is a filesystem which contains the environment's contents.
	FS core.PathedFS
}

// An Env is a Forklift environment, a complete specification of all package deployments which
// should be active on a Docker host.
type Env struct {
	// Def is the Forklift environment definition for the environment.
	Def EnvDef
}

// EnvDefFile is the name of the file defining each Forklift environment.
const EnvDefFile = "forklift-env.yml"

// A EnvDef defines a Forklift environment.
type EnvDef struct {
	// Environment defines the basic metadata for the environment.
	Environment EnvSpec `yaml:"environment,omitempty"`
}

// EnvSpec defines the basic metadata for a Forklift environment.
type EnvSpec struct {
	// Description is a short description of the environment to be shown to users.
	Description string `yaml:"description,omitempty"`
}

// Deployment Configurations

const (
	// DeplsDirName is the directory in a Forklift environment which contains deployment
	// configurations.
	DeplsDirName = "deployments"
	// DeplsFileExt is the file extension for deployment configuration files.
	DeplDefFileExt = ".deploy.yml"
)

// A ResolvedDepl is a deployment with a loaded package.
type ResolvedDepl struct {
	// Depl is the configured deployment of the package represented by Pkg.
	Depl
	// PkgReq is the package requirement for the deployment.
	PkgReq PkgReq
	// Pkg is the package to be deployed.
	Pkg *core.FSPkg
}

// A Depl is a package deployment, a complete configuration of how a package is to be deployed on a
// Docker host.
type Depl struct {
	// Name is the name of the package depoyment.
	Name string
	// Def is the Forklift package deployment definition for the deployment.
	Def DeplDef
}

// A DeplDef defines a package deployment.
type DeplDef struct {
	// Package is the package path of the package to deploy
	Package string `yaml:"package,omitempty"`
	// Features is a list of features from the package which should be enabled in the deployment.
	Features []string `yaml:"features,omitempty"`
}

// Requirements

// ReqsDirName is the directory in a Forklift environment which contains requirement configurations.
const ReqsDirName = "requirements"

// A PkgReq is a requirement for a package at a specific version.
type PkgReq struct {
	// PkgSubdir is the package subdirectory in the repo which should provide the required package.
	PkgSubdir string
	// Repo is the repo which should provide the required package.
	Repo RepoReq
}

// PkgReqLoader is a source of package requirements.
type PkgReqLoader interface {
	LoadPkgReq(pkgPath string) (PkgReq, error)
}

const (
	// ReqsReposDirName is the subdirectory in the requirements directory of a Forklift environment
	// which contains repo requirement configurations.
	ReqsReposDirName = "repositories"
)

// A FSRepoReq is a repo requirement stored at the root of a [fs.FS] filesystem.
type FSRepoReq struct {
	// RepoReq is the repo requirement at the root of the filesystem.
	RepoReq
	// FS is a filesystem which contains the repo requirement's contents.
	FS core.PathedFS
}

// A RepoReq is a requirement for a specific repo at a specific version.
type RepoReq struct {
	// RepoPath is the repository path of the required repo.
	RepoPath string
	// VersionLock specifies the version of the required repo.
	VersionLock VersionLock
}
