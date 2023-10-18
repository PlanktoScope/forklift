// Package forklift provides the core functionality of the forklift tool
package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/core"
)

// A FSPallet is a Forklift pallet stored at the root of a [fs.FS] filesystem.
type FSPallet struct {
	// Pallet is the pallet at the root of the filesystem.
	Pallet
	// FS is a filesystem which contains the pallet's contents.
	FS core.PathedFS
}

// A Pallet is a Forklift pallet, a complete specification of all package deployments which should
// be active on a Docker host.
type Pallet struct {
	// Def is the Forklift pallet definition for the pallet.
	Def PalletDef
}

// PalletDefFile is the name of the file defining each Forklift pallet.
const PalletDefFile = "forklift-pallet.yml"

// A PalletDef defines a Forklift pallet.
type PalletDef struct {
	// ForkliftVersion indicates that the pallet was written assuming the semantics of a given version
	// of Forklift. The version must be a valid Forklift version, and it sets the minimum version of
	// Forklift required to use the pallet. The Forklift tool refuses to use pallets declaring newer
	// Forklift versions for any operations beyond printing information. The Forklift version of the
	// pallet must be greater than or equal to the Forklift version of every required Forklift repo or
	// pallet.
	ForkliftVersion string `yaml:"forklift-version"`
	// Pallet defines the basic metadata for the pallet.
	Pallet PalletSpec `yaml:"pallet,omitempty"`
}

// PalletSpec defines the basic metadata for a Forklift pallet.
type PalletSpec struct {
	// Path is the pallet path, which acts as the canonical name for the pallet. It should just be the
	// path of the VCS repository for the pallet.
	Path string `yaml:"path"`
	// Description is a short description of the pallet to be shown to users.
	Description string `yaml:"description"`
	// ReadmeFile is the name of a readme file to be shown to users.
	ReadmeFile string `yaml:"readme-file"`
}

// Deployment Configurations

const (
	// DeplsDirName is the directory in a pallet which contains deployment
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
	// Package is the package path of the package to deploy.
	Package string `yaml:"package"`
	// Features is a list of features from the package which should be enabled in the deployment.
	Features []string `yaml:"features"`
	// Disabled represents whether the deployment should be ignored.
	Disabled bool `yaml:"disabled"`
}

// Requirements

// ReqsDirName is the directory in a Forklift pallet which contains requirement configurations.
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
	// ReqsReposDirName is the subdirectory in the requirements directory of a Forklift pallet which
	// contains repo requirement configurations.
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
