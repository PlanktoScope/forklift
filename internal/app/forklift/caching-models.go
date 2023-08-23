package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/core"
)

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
	repoVersionSets map[string]map[string]struct{}
}
