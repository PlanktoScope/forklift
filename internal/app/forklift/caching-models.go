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

// FSCache is a local cache with copies of Pallet repositories (and thus of Pallet packages too),
// stored in a [fs.FS] filesystem.
type FSCache struct {
	// FS is the filesystem which corresponds to the cache.
	FS pallets.PathedFS
}

// OverriddenCache is a cache where selected repositories of any version can be overridden by a set
// of pre-loaded repositories with matching paths, for loading Pallet repositories and packages.
type OverriddenCache struct {
	// Cache is the underlying cache
	Cache Cache
	// Overrides is a map associating Pallet repository paths to loaded Pallet repositories.
	Overrides map[string]*pallets.FSRepo
}
