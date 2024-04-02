package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/core"
)

// Bundle

const (
	// deploymentsDirName is the name of the directory containing bundled files for each package
	// deployment.
	deploymentsDirName = "deployments"
)

// A FSBundle is a Forklift pallet bundle stored at the root of a [fs.FS] filesystem.
type FSBundle struct {
	// Bundle is the pallet bundle at the root of the filesystem.
	Bundle
	// FS is a filesystem which contains the bundle's contents.
	FS core.PathedFS
}

// A Bundle is a Forklift pallet bundle, a complete compilation of all files (except container
// images) needed for a pallet to be applied to a Docker host. Required repos & pallets are included
// directly in the bundle.
type Bundle struct {
	// Def is the Forklift bundle definition for the pallet bundle.
	Def BundleDef
}

// BundleDefFile is the name of the file defining each Forklift pallet bundle.
const BundleDefFile = "forklift-bundle.yml"

// A BundleDef defines a Forklift pallet bundle.
type BundleDef struct {
	// ForkliftVersion indicates that the pallet bundle was created assuming the semantics of a given
	// version of Forklift. The version must be a valid Forklift version, and it sets the minimum
	// version of Forklift required to use the pallet bundle. The Forklift tool refuses to use pallet
	// bundles declaring newer Forklift versions for any operations beyond printing information. The
	// Forklift version of the pallet bundle must be greater than or equal to the Forklift version of
	// every required Forklift repo or pallet bundle.
	ForkliftVersion string `yaml:"forklift-version"`
	// Pallet defines the basic metadata for the bundled pallet.
	Pallet BundlePallet `yaml:"pallet,omitempty"`
	// Includes describes repos and pallets used by the bundled pallet to define the bundle's
	// package deployments.
	Includes BundlePalletInclusions `yaml:"includes,omitempty"`
	// Deploys describes deployments provided by the bundle. Keys are names of deployments.
	Deploys map[string]DeplDef `yaml:"deploys,omitempty"`
}

// BundlePallet describes a bundle's bundled pallet.
type BundlePallet struct {
	// Path is the pallet bundle's path, which acts as the canonical name for the pallet bundle. It
	// should just be the path of the VCS repository for the bundled pallet.
	Path string `yaml:"path"`
	// Version is the version or pseudoversion of the bundled pallet, if one can be determined.
	Version string `yaml:"version"`
	// Clean indicates whether the bundled pallet has been determined to have no changes beyond its
	// latest Git commit, if the pallet is version-controlled with Git. This does not account for
	// overrides of required repos/pallets - those should be checked in BundlePalletInclusions
	// instead.
	Clean bool `yaml:"clean"`
	// Description is a short description of the bundled pallet to be shown to users.
	Description string `yaml:"description,omitempty"`
}

// BundlePalletInclusions describes the requirements used to build the bundled pallet.
type BundlePalletInclusions struct {
	// Pallets describes external pallets used to build the bundled pallet.
	Pallets map[string]BundlePalletInclusion `yaml:"pallets,omitempty"`
	// Repos describes package repositories used to build the bundled pallet.
	Repos map[string]BundleRepoInclusion `yaml:"repositories,omitempty"`
}

// BundlePalletInclusion describes a pallet used to build the bundled pallet.
type BundlePalletInclusion struct {
	Req PalletReq `yaml:"requirement,inline"`
	// Override describes the pallet used to override the required pallet, if an override was
	// specified for the pallet when building the bundled pallet.
	Override BundleInclusionOverride `yaml:"override,omitempty"`
}

// BundleRepoInclusion describes a package repository used to build the bundled pallet.
type BundleRepoInclusion struct {
	Req RepoReq `yaml:"requirement,inline"`
	// Override describes the pallet used to override the required pallet, if an override was
	// specified for the pallet when building the bundled pallet.
	Override BundleInclusionOverride `yaml:"override,omitempty"`
}

type BundleInclusionOverride struct {
	// Path is the path of the override. This should be a filesystem path.
	Path string `yaml:"path"`
	// Version is the version or pseudoversion of the override, if one can be determined.
	Version string `yaml:"version"`
	// Clean indicates whether the override has been determined to have no changes beyond its latest
	// Git commit, if the it's version-controlled with Git.
	Clean bool `yaml:"clean"`
}
