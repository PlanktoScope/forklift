package forklift

import (
	"github.com/forklift-run/forklift/pkg/core"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	"github.com/forklift-run/forklift/pkg/structures"
)

// Mirror

// FSMirrorCache is a [PathedPalletCache] implementation with git repository mirrors
// stored in a [core.PathedFS] filesystem.
type FSMirrorCache struct {
	// FS is the filesystem which corresponds to the cache of pallets.
	FS ffs.PathedFS
}

// Pallet

// FSPalletLoader is a source of [FSPallet]s indexed by path and version.
type FSPalletLoader interface {
	// LoadFSPallet loads the FSPallet with the specified path and version.
	LoadFSPallet(palletPath string, version string) (*FSPallet, error)
	// LoadFSPallets loads all FSPallets matching the specified search pattern.
	LoadFSPallets(searchPattern string) ([]*FSPallet, error)
}

// PalletCache is a source of pallets.
type PalletCache interface {
	FSPalletLoader
	FSPkgLoader
}

// PathedPalletCache is a PalletCache rooted at a single path.
type PathedPalletCache interface {
	PalletCache
	ffs.Pather
}

// FSPalletCache is a [PathedPalletCache] implementation with copies of unmerged pallets
// stored in a [core.PathedFS] filesystem.
type FSPalletCache struct {
	// FS is the filesystem which corresponds to the cache of pallets.
	FS ffs.PathedFS
}

// FSMergedPalletCache is a [PathedPalletCache] implementation with copies of merged pallets
// stored in a [core.PathedFS] filesystem.
// FIXME: implement this!
type FSMergedPalletCache struct {
	// FS is the filesystem which corresponds to the cache of pallets.
	FS ffs.PathedFS
}

// LayeredPalletCache is a [PathedPalletCache] implementation where selected pallets can be
// overridden by an [OverlayPalletCache], for loading pallets.
// The path of the LayeredPalletCache instance is just the path of the underlying cache.
type LayeredPalletCache struct {
	// Underlay is the underlying cache.
	Underlay PathedPalletCache
	// Overlay is the overlying cache which is used instead of the underlying cache for pallets
	// covered by the overlying cache.
	Overlay OverlayPalletCache
}

// OverlayPalletCache is a [PalletCache] which can report whether it includes any particular pallet.
type OverlayPalletCache interface {
	PalletCache
	// IncludesFSPallet reports whether the cache expects to have the specified pallet.
	// This result does not necessarily correspond to whether the cache actually has it.
	IncludesFSPallet(palletPath string, version string) bool
	// IncludesFSPkg reports whether the cache expects to have the specified package.
	// This result does not necessarily correspond to whether the cache actually has it.
	IncludesFSPkg(pkgPath string, version string) bool
}

// PalletOverrideCache is an [OverlayPalletCache] implementation containing a set of pallets which
// can be retrieved from the root of the cache. A pallet from the cache will be retrieved if it is
// stored in the cache with a matching version, regardless of whether the pallet's own version
// actually matches - in other words, pallets can be stored with fictional versions.
type PalletOverrideCache struct {
	// pallets is a map associating pallet paths to loaded pallets.
	// For each key-value pair, the key must be the path of the pallet which is the value of that
	// key-value pair.
	pallets map[string]*FSPallet
	// palletPaths is an alphabetically ordered list of the keys of pallets.
	palletPaths []string
	// palletVersions is a map associating pallet paths to pallet version strings.
	palletVersions map[string][]string
	// palletVersionSets is like palletVersions, but every value is a set of versions rather than a
	// list of versions.
	palletVersionSets map[string]structures.Set[string]
}

// FSPkgTree

// FSPkgLoader is a source of [core.FSPkg]s indexed by path and version.
type FSPkgLoader interface {
	// LoadFSPkg loads the FSPkg with the specified path and version.
	LoadFSPkg(pkgPath string, version string) (*core.FSPkg, error)
	// LoadFSPkgs loads all FSPkgs matching the specified search pattern.
	LoadFSPkgs(searchPattern string) ([]*core.FSPkg, error)
}

// FSPkgTreeCache is a source of repos and packages.
type FSPkgTreeCache interface {
	FSPkgLoader
}

// PathedFSPkgTreeCache is a FSPkgTreeCache rooted at a single path.
type PathedFSPkgTreeCache interface {
	FSPkgTreeCache
	ffs.Pather
}

// OverlayFSPkgTreeCache is a [FSPkgTreeCache] which can report whether it includes any particular repo or
// package.
type OverlayFSPkgTreeCache interface {
	FSPkgTreeCache
	// IncludesFSPkgTree reports whether the cache expects to have the specified repo.
	// This result does not necessarily correspond to whether the cache actually has it.
	IncludesFSPkgTree(repoPath string, version string) bool
	// IncludesFSPkg reports whether the cache expects to have the specified package.
	// This result does not necessarily correspond to whether the cache actually has it.
	IncludesFSPkg(pkgPath string, version string) bool
}

// Download

// FSDownloadCache is a source of downloaded files saved on the filesystem.
type FSDownloadCache struct {
	// FS is the filesystem which corresponds to the cache of downloads.
	FS ffs.PathedFS
}
