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
	// Version is the version or pseudoversion of the pallet.
	Version string
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

// Requirements

// ReqsDirName is the directory in a Forklift pallet which contains requirement declarations.
const ReqsDirName = "requirements"

// A GitRepoReq is a requirement for a specific Git repository (e.g. a pallet or Forklift repo) at
// a specific version.
type GitRepoReq struct {
	// GitRepoPath is the path of the required Git repository.
	RequiredPath string `yaml:"-"`
	// VersionLock specifies the version of the required Git repository.
	VersionLock VersionLock `yaml:"version-lock"`
}

const (
	// ReqsPalletsDirName is the subdirectory in the requirements directory of a Forklift pallet which
	// contains pallet requirement declarations.
	ReqsPalletsDirName = "pallets"
)

// A FSPalletReq is a pallet requirement stored at the root of a [fs.FS] filesystem.
type FSPalletReq struct {
	// PalletReq is the pallet requirement at the root of the filesystem.
	PalletReq
	// FS is a filesystem which contains the pallet requirement's contents.
	FS core.PathedFS
}

// A PalletReq is a requirement for a specific pallet at a specific version.
type PalletReq struct {
	GitRepoReq `yaml:",inline"`
}

// PalletReqLoader is a source of pallet requirements.
type PalletReqLoader interface {
	LoadPalletReq(palletPath string) (PalletReq, error)
}

const (
	// ReqsReposDirName is the subdirectory in the requirements directory of a Forklift pallet which
	// contains repo requirement declarations.
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
	GitRepoReq `yaml:",inline"`
}

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

// Deployments

const (
	// DeplsDirName is the directory in a pallet which contains deployment declarations.
	DeplsDirName = "deployments"
	// DeplsFileExt is the file extension for deployment declaration files.
	DeplDefFileExt = ".deploy.yml"
)

// A ResolvedDepl is a deployment with a loaded package.
type ResolvedDepl struct {
	// Depl is the declared deployment of the package represented by Pkg.
	Depl
	// PkgReq is the package requirement for the deployment.
	PkgReq PkgReq
	// Pkg is the package to be deployed.
	Pkg *core.FSPkg
}

// A Depl is a package deployment, a complete declaration of how a package is to be deployed on a
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
	Features []string `yaml:"features,omitempty"`
	// Disabled represents whether the deployment should be ignored.
	Disabled bool `yaml:"disabled,omitempty"`
}

// Imports

const (
	// ImportDefFileExt is the file extension for import group files.
	ImportDefFileExt = ".imports.yml"
)

// A ResolvedImport is a deployment with a loaded pallet.
type ResolvedImport struct {
	// Import is the declared file import group.
	Import
	// Pallet is the pallet which files will be imported from
	Pallet *FSPallet
}

// An Import is an import group, a declaration of a group of files to import from a required pallet.
type Import struct {
	// Name is the name of the package file import.
	Name string
	// Def is the file import definition for the file import.
	Def ImportDef
}

// A ImportDef defines a file import group.
type ImportDef struct {
	// Description is a short description of the import group to be shown to users.
	Description string `yaml:"description,omitempty"`
	// Modifiers is a list of modifiers evaluated in the provided order to build up a set of files to
	// import.
	Modifiers []ImportModifier `yaml:"modifiers"`
	// Disabled represents whether the import should be ignored.
	Disabled bool `yaml:"disabled,omitempty"`
}

// An ImportModifier defines an operation for transforming a set of files for importing into a
// different set of files for importing.
type ImportModifier struct {
	// Description is a short description of the import modifier to be shown to users.
	Description string `yaml:"description,omitempty"`
	// Type is either `add` (for adding one or more files to the set of files to import) or `remove`
	// (for removing one or more files from the set of files to import)
	Type string `yaml:"type,omitempty"`
	// Source is the path in the required pallet of the file/directory to be imported, for an `add`
	// modifier. If omitted, the source path will be inferred from the Target path.
	Source string `yaml:"source,omitempty"`
	// Target is the path which the file/directory will be imported as, for an `add` modifier; or the
	// path of the file/directory which will be removed from the set of files to import, for a
	// `remove` modifier.
	Target string `yaml:"target,omitempty"`
	// OnlyMatchingAny is, if the source is a directory, a list of glob patterns (relative to the
	// source path) of files which will be added/removed (depending on modifier type). Any file which
	// matches none of patterns provided in this field will be ignored for the add/remove modifier. If
	// omitted, no files in the source directory will be ignored.
	OnlyMatchingAny []string `yaml:"only-matching-any,omitempty"`
}
