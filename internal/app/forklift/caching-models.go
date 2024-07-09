package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/core"
	"github.com/PlanktoScope/forklift/pkg/structures"
)

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
}

// PathedPalletCache is a PalletCache rooted at a single path.
type PathedPalletCache interface {
	PalletCache
	core.Pather
}

// FSPalletCache is a [PathedPalletCache] implementation with copies of pallets
// stored in a [core.PathedFS] filesystem.
type FSPalletCache struct {
	// FS is the filesystem which corresponds to the cache of pallets.
	FS core.PathedFS
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

// Repo

// FSRepoLoader is a source of [core.FSRepo]s indexed by path and version.
type FSRepoLoader interface {
	// LoadFSRepo loads the FSRepo with the specified path and version.
	LoadFSRepo(repoPath string, version string) (*core.FSRepo, error)
	// LoadFSRepos loads all FSRepos matching the specified search pattern.
	LoadFSRepos(searchPattern string) ([]*core.FSRepo, error)
}

// FSPkgLoader is a source of [core.FSPkg]s indexed by path and version.
type FSPkgLoader interface {
	// LoadFSPkg loads the FSPkg with the specified path and version.
	LoadFSPkg(pkgPath string, version string) (*core.FSPkg, error)
	// LoadFSPkgs loads all FSPkgs matching the specified search pattern.
	LoadFSPkgs(searchPattern string) ([]*core.FSPkg, error)
}

// RepoCache is a source of repos and packages.
type RepoCache interface {
	FSRepoLoader
	FSPkgLoader
}

// PathedRepoCache is a RepoCache rooted at a single path.
type PathedRepoCache interface {
	RepoCache
	core.Pather
}

// FSRepoCache is a [PathedRepoCache] implementation with copies of repos (and thus of packages too)
// stored in a [core.PathedFS] filesystem.
type FSRepoCache struct {
	// FS is the filesystem which corresponds to the cache of repos.
	FS core.PathedFS
}

// LayeredRepoCache is a [PathedRepoCache] implementation where selected repos can be overridden by
// an [OverlayRepoCache], for loading repos and packages.
// The path of the LayeredRepoCache instance is just the path of the underlying cache.
type LayeredRepoCache struct {
	// Underlay is the underlying cache.
	Underlay PathedRepoCache
	// Overlay is the overlying cache which is used instead of the underlying cache for repos and
	// packages covered by the overlying cache.
	Overlay OverlayRepoCache
}

// OverlayRepoCache is a [RepoCache] which can report whether it includes any particular repo or
// package.
type OverlayRepoCache interface {
	RepoCache
	// IncludesFSRepo reports whether the cache expects to have the specified repo.
	// This result does not necessarily correspond to whether the cache actually has it.
	IncludesFSRepo(repoPath string, version string) bool
	// IncludesFSPkg reports whether the cache expects to have the specified package.
	// This result does not necessarily correspond to whether the cache actually has it.
	IncludesFSPkg(pkgPath string, version string) bool
}

// RepoOverrideCache is an [OverlayRepoCache] implementation containing a set of repos which
// can be retrieved from the root of the cache. A repo from the cache will be retrieved if it is
// stored in the cache with a matching version, regardless of whether the repo's own version
// actually matches - in other words, repos can be stored with fictional versions.
type RepoOverrideCache struct {
	// repos is a map associating repo paths to loaded repos.
	// For each key-value pair, the key must be the path of the repo which is the value of that
	// key-value pair.
	repos map[string]*core.FSRepo
	// repoPaths is an alphabetically ordered list of the keys of repos.
	repoPaths []string
	// repoVersions is a map associating repo paths to repo version strings.
	repoVersions map[string][]string
	// repoVersionSets is like repoVersions, but every value is a set of versions rather than a
	// list of versions.
	repoVersionSets map[string]structures.Set[string]
}

// Download

// FSDownloadCache is a source of downloaded files saved on the filesystem.
type FSDownloadCache struct {
	// FS is the filesystem which corresponds to the cache of downloads.
	FS core.PathedFS
}
