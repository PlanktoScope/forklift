package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/core"
)

// Bundle

const (
	// bundledPalletDirName is the name of the directory containing the bundled pallet.
	bundledPalletDirName = "pallet"
	// bundledMergedPalletDirName is the name of the directory containing the bundled pallet, merged
	// with file imports from its required pallets.
	bundledMergedPalletDirName = "merged-pallet"
	// packagesDirName is the name of the directory containing bundled files for each package.
	packagesDirName = "packages"
	// exportsDirName is the name of the directory containing exported files for all package
	// deployments, collected together.
	exportsDirName = "exports"
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
	// Manifest is the Forklift bundle manifest for the pallet bundle.
	Manifest BundleManifest
}

// BundleManifestFile is the name of the file describing each Forklift pallet bundle.
const BundleManifestFile = "forklift-bundle.yml"

// A BundleManifest describes a Forklift pallet bundle.
type BundleManifest struct {
	// ForkliftVersion indicates that the pallet bundle was created assuming the semantics of a given
	// version of Forklift. The version must be a valid Forklift version, and it sets the minimum
	// version of Forklift required to use the pallet bundle. The Forklift tool refuses to use pallet
	// bundles declaring newer Forklift versions for any operations beyond printing information. The
	// Forklift version of the pallet bundle must be greater than or equal to the Forklift version of
	// every required Forklift repo or pallet bundle.
	ForkliftVersion string `yaml:"forklift-version"`
	// Pallet describes the basic metadata for the bundled pallet.
	Pallet BundlePallet `yaml:"pallet"`
	// Includes describes repos and pallets used to define the bundle's package deployments.
	Includes BundleInclusions `yaml:"includes,omitempty"`
	// Imports lists the files imported from required pallets and the fully-qualified paths of those
	// source files (relative to their respective source pallets). Keys are the target paths of the
	// files, while values are lists showing the chain of provenance of the respective files (with
	// the deepest ancestor at the end of each list).
	Imports map[string][]string `yaml:"imports,omitempty"`
	// Downloads lists the URLs of files and OCI images downloaded for export by the bundle's
	// deployments. Keys are names of the bundle's deployments which export downloaded files.
	Downloads map[string][]string `yaml:"downloads,omitempty"`
	// Deploys describes deployments provided by the bundle. Keys are names of deployments.
	Deploys map[string]DeplDef `yaml:"deploys,omitempty"`
	// Exports lists the target paths of file exports provided by the bundle's deployments. Keys are
	// names of the bundle's deployments which provide file exports.
	Exports map[string][]string `yaml:"exports,omitempty"`
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
	// overrides of required repos/pallets - those should be checked in BundleInclusions instead.
	Clean bool `yaml:"clean"`
	// Description is a short description of the bundled pallet to be shown to users.
	Description string `yaml:"description,omitempty"`
}

// BundleInclusions describes the requirements used to build the bundled pallet.
type BundleInclusions struct {
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
	// Includes describes pallets used to define the pallet, omitting information about file imports.
	Includes map[string]BundlePalletInclusion `yaml:"includes,omitempty"`
	// Imports lists the files imported from the pallet, organized by import group. Keys are the names
	// of the import groups, and values are the results of evaluating the respective import groups -
	// i.e. maps whose keys are target file paths (where the files are imported to) and whose values
	// are source file paths (where the files are imported from).
	Imports map[string]map[string]string `yaml:"imports,omitempty"`
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
