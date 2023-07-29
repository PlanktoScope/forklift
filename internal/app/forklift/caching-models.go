package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// FSRepoLoader is a source of Pallet repositories indexed by path and version.
type FSRepoLoader interface {
	// LoadFSRepo loads the FSRepo with the specified path and version.
	LoadFSRepo(repoPath string, version string) (*pallets.FSRepo, error)
	// LoadFSRepos loads all FSRepos matching the specified search pattern.
	LoadFSRepos(searchPattern string) ([]*pallets.FSRepo, error)
}

// FSPkgLoader is a source of Pallet packages indexed by path and version.
type FSPkgLoader interface {
	// LoadFSPkg loads the FSPkg with the specified path and version.
	LoadFSPkg(pkgPath string, version string) (*pallets.FSPkg, error)
	// LoadFSPkgs loads all FSPkgs matching the specified search pattern.
	LoadFSPkgs(searchPattern string) ([]*pallets.FSPkg, error)
}

// Cache is a source of Pallet repositories and packages.
type Cache interface {
	FSRepoLoader
	FSPkgLoader
}

// PathedCache is a Cache rooted at a single path.
type PathedCache interface {
	Cache
	pallets.Pather
}

// FSCache is a local cache with copies of Pallet repositories (and thus of Pallet packages too),
// stored in a [fs.FS] filesystem.
type FSCache struct {
	// FS is the filesystem which corresponds to the cache.
	FS pallets.PathedFS
}

// LayeredCache is a PathedCache where selected repositories can be overridden by another Cache,
// for loading Pallet repositories and packages.
// The path of the LayeredCache instance is just the path of the underlying cache.
type LayeredCache struct {
	// Underlay is the underlying cache.
	Underlay PathedCache
	// Overlay is the overlying cache which is used instead of the underlying cache for repositories
	// and packages covered by the overlying cache.
	Overlay OverlayCache
}

// OverlayCache is a cache which can report whether it includes any particular repository or
// package.
type OverlayCache interface {
	Cache
	// IncludesFSRepo reports whether the cache expects to have the specified repository.
	// This result does not necessarily correspond to whether the cache actually has it.
	IncludesFSRepo(repoPath string, version string) bool
	// IncludesFSPkg reports whether the cache expects to have the specified package.
	// This result does not necessarily correspond to whether the cache actually has it.
	IncludesFSPkg(pkgPath string, version string) bool
}

// RepoOverrideCache is an [OverlayCache] implementation containing a set of repos which can be
// retrieved from the root of the cache. A repo from the cache will be retrieved if it is stored
// in the cache with a matching version, regardless of whether the repo's own version actually
// matches - in other words, repos can be stored with fictional versions.
type RepoOverrideCache struct {
	// repos is a map associating Pallet repository paths to loaded Pallet repositories underneath
	// this filesystem. Repositories at any level underneath this filesystem should be included, so
	// they don't have to be immediate children - they can be indirect descendants. For each key-value
	// pair, the key must be the Pallet repository path of the repo which is the value of that
	// key-value pair.
	repos map[string]*pallets.FSRepo
	// repoPaths is an alphabetically ordered list of the keys of repos.
	repoPaths []string
	// repoVersions is a map associating Pallet repository paths to Pallet repository version strings.
	// Each version of each Pallet repository will result in a DirEntry for the Pallet repository's
	// VCS repository directory.
	repoVersions map[string][]string
	// repoVersionSets is like repoVersions, but every value is a set of versions rather than a list
	// of versions.
	repoVersionSets map[string]map[string]struct{}
}
