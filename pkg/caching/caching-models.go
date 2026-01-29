package caching

import (
	ffs "github.com/forklift-run/forklift/pkg/fs"
	fpkg "github.com/forklift-run/forklift/pkg/packaging"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
	"github.com/forklift-run/forklift/pkg/structures"
)

// Mirror

// FSMirrorCache is a [PathedPalletCache] implementation with git repository mirrors
// stored in a [fpkg.PathedFS] filesystem.
type FSMirrorCache struct {
	// FS is the filesystem which corresponds to the cache of pallets.
	FS ffs.PathedFS
}

// Pallet

// PalletCache is a source of pallets.
type PalletCache interface {
	fplt.FSPalletLoader
	fplt.FSPkgLoader
}

// PathedPalletCache is a PalletCache rooted at a single path.
type PathedPalletCache interface {
	PalletCache
	ffs.Pather
}

// FSPalletCache is a [PathedPalletCache] implementation with copies of pallets stored in a
// [fpkg.PathedFS] filesystem.
type FSPalletCache struct {
	// pkgTree is the filesystem which corresponds to the cache of pallets.
	pkgTree *fpkg.FSPkgTree
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
	pallets map[string]*fplt.FSPallet
	// palletPaths is an alphabetically ordered list of the keys of pallets.
	palletPaths []string
	// palletVersions is a map associating pallet paths to pallet version strings.
	palletVersions map[string][]string
	// palletVersionSets is like palletVersions, but every value is a set of versions rather than a
	// list of versions.
	palletVersionSets map[string]structures.Set[string]
}

// FSPkgTree

// FSPkgTreeCache is a source of packages.
type FSPkgTreeCache interface {
	fplt.FSPkgLoader
}

// PathedFSPkgTreeCache is a FSPkgTreeCache rooted at a single path.
type PathedFSPkgTreeCache interface {
	FSPkgTreeCache
	ffs.Pather
}

// OverlayFSPkgTreeCache is a [FSPkgTreeCache] which can report whether it includes any particular
// package.
type OverlayFSPkgTreeCache interface {
	FSPkgTreeCache
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
