package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// FSPalletLoader is a source of pallets indexed by path and version.
type FSPalletLoader interface {
	// LoadFSPallet loads the FSPallet with the specified path and version.
	LoadFSPallet(palletPath string, version string) (*pallets.FSPallet, error)
	// LoadFSPallets loads all FSPallets matching the specified search pattern.
	LoadFSPallets(searchPattern string) ([]*pallets.FSPallet, error)
}

// FSPkgLoader is a source of packages indexed by path and version.
type FSPkgLoader interface {
	// LoadFSPkg loads the FSPkg with the specified path and version.
	LoadFSPkg(pkgPath string, version string) (*pallets.FSPkg, error)
	// LoadFSPkgs loads all FSPkgs matching the specified search pattern.
	LoadFSPkgs(searchPattern string) ([]*pallets.FSPkg, error)
}

// Cache is a source of pallets and packages.
type Cache interface {
	FSPalletLoader
	FSPkgLoader
}

// PathedCache is a Cache rooted at a single path.
type PathedCache interface {
	Cache
	pallets.Pather
}

// FSCache is a local cache with copies of pallets (and thus of packages too), stored in a [fs.FS]
// filesystem.
type FSCache struct {
	// FS is the filesystem which corresponds to the cache.
	FS pallets.PathedFS
}

// LayeredCache is a PathedCache where selected pallets can be overridden by another Cache, for
// loading pallets and packages.
// The path of the LayeredCache instance is just the path of the underlying cache.
type LayeredCache struct {
	// Underlay is the underlying cache.
	Underlay PathedCache
	// Overlay is the overlying cache which is used instead of the underlying cache for pallets and
	// packages covered by the overlying cache.
	Overlay OverlayCache
}

// OverlayCache is a cache which can report whether it includes any particular pallet or package.
type OverlayCache interface {
	Cache
	// IncludesFSPallet reports whether the cache expects to have the specified pallet.
	// This result does not necessarily correspond to whether the cache actually has it.
	IncludesFSPallet(palletPath string, version string) bool
	// IncludesFSPkg reports whether the cache expects to have the specified package.
	// This result does not necessarily correspond to whether the cache actually has it.
	IncludesFSPkg(pkgPath string, version string) bool
}

// PalletOverrideCache is an [OverlayCache] implementation containing a set of pallets which can be
// retrieved from the root of the cache. A pallet from the cache will be retrieved if it is stored
// in the cache with a matching version, regardless of whether the pallet's own version actually
// matches - in other words, pallets can be stored with fictional versions.
type PalletOverrideCache struct {
	// pallets is a map associating pallet paths to loaded pallets.
	// For each key-value pair, the key must be the path of the pallet which is the value of that
	// key-value pair.
	pallets map[string]*pallets.FSPallet
	// palletPaths is an alphabetically ordered list of the keys of pallets.
	palletPaths []string
	// palletVersions is a map associating pallet paths to pallet version strings.
	palletVersions map[string][]string
	// palletVersionSets is like palletVersions, but every value is a set of versions rather than a
	// list of versions.
	palletVersionSets map[string]map[string]struct{}
}
