package forklift

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/core"
	"github.com/PlanktoScope/forklift/pkg/structures"
)

// FSRepoCache

// Exists checks whether the cache actually exists on the OS's filesystem.
func (c *FSRepoCache) Exists() bool {
	return DirExists(filepath.FromSlash(c.FS.Path()))
}

// Remove deletes the cache from the OS's filesystem, if it exists.
func (c *FSRepoCache) Remove() error {
	return os.RemoveAll(filepath.FromSlash(c.FS.Path()))
}

// Path returns the path of the cache's filesystem.
func (c *FSRepoCache) Path() string {
	return c.FS.Path()
}

// FSRepoCache: FSRepoLoader

// LoadFSRepo loads the FSRepo with the specified path and version.
// The loaded FSRepo instance is fully initialized.
func (c *FSRepoCache) LoadFSRepo(repoPath string, version string) (*core.FSRepo, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	repo, err := core.LoadFSRepo(c.FS, fmt.Sprintf("%s@%s", repoPath, version))
	if err != nil {
		return nil, err
	}
	repo.Version = version
	return repo, nil
}

// LoadFSRepos loads all FSRepos from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching repo directories to
// search for.
// The loaded FSRepo instances are fully initialized.
func (c *FSRepoCache) LoadFSRepos(searchPattern string) ([]*core.FSRepo, error) {
	if c == nil {
		return nil, nil
	}

	repos, err := core.LoadFSRepos(c.FS, searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load repos from cache")
	}

	// set the Version field of the repo based on its path in the cache
	for _, repo := range repos {
		var repoPath string
		var ok bool
		if repoPath, repo.Version, ok = strings.Cut(core.GetSubdirPath(c, repo.FS.Path()), "@"); !ok {
			return nil, errors.Wrapf(
				err, "couldn't parse path of cached repo configured at %s as repo_path@version",
				repo.FS.Path(),
			)
		}
		if repoPath != repo.Path() {
			return nil, errors.Errorf(
				"cached repo %s is in cache at %s@%s instead of %s@%s",
				repo.Path(), repoPath, repo.Version, repo.Path(), repo.Version,
			)
		}
	}

	return repos, nil
}

// FSRepoCache: FSPkgLoader

// LoadFSPkg loads the FSPkg with the specified path and version.
// The loaded FSPkg instance is fully initialized.
func (c *FSRepoCache) LoadFSPkg(pkgPath string, version string) (*core.FSPkg, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	// Search for the package by starting with the shortest possible package subdirectory path and the
	// longest possible repo path, and shifting path components from the repo path to the package
	// subdirectory path until we successfully load the package.
	repoPath := path.Dir(pkgPath)
	pkgSubdir := path.Base(pkgPath)
	for repoPath != "." && repoPath != "/" {
		repo, err := c.LoadFSRepo(repoPath, version)
		if err != nil {
			pkgSubdir = path.Join(path.Base(repoPath), pkgSubdir)
			repoPath = path.Dir(repoPath)
			continue
		}
		pkg, err := repo.LoadFSPkg(pkgSubdir)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't load package %s from repo %s at version %s",
				pkgPath, repoPath, version,
			)
		}
		return pkg, nil
	}
	return nil, errors.Errorf("no cached packages were found matching %s@%s", pkgPath, version)
}

// LoadFSPkgs loads all FSPkgs from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching package directories
// to search for.
// The loaded FSPkg instances are fully initialized.
func (c *FSRepoCache) LoadFSPkgs(searchPattern string) ([]*core.FSPkg, error) {
	if c == nil {
		return nil, nil
	}

	pkgs, err := core.LoadFSPkgs(c.FS, searchPattern)
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgs {
		repo, err := c.loadFSRepoContaining(core.GetSubdirPath(c, pkg.FS.Path()))
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't find the cached repo providing the cached package at %s", pkg.FS.Path(),
			)
		}
		if err = pkg.AttachFSRepo(repo); err != nil {
			return nil, errors.Wrap(err, "couldn't attach cached repo to cached package")
		}
	}
	return pkgs, nil
}

// loadFSRepoContaining finds and loads the FSRepo which contains the provided subdirectory
// path.
func (c *FSRepoCache) loadFSRepoContaining(
	subdirPath string,
) (repo *core.FSRepo, err error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	if repo, err = core.LoadFSRepoContaining(c.FS, subdirPath); err != nil {
		return nil, errors.Wrapf(err, "couldn't find any repo containing %s", subdirPath)
	}
	var repoPath string
	var ok bool
	if repoPath, repo.Version, ok = strings.Cut(core.GetSubdirPath(c, repo.FS.Path()), "@"); !ok {
		return nil, errors.Wrapf(
			err, "couldn't parse path of cached repo configured at %s as repo_path@version",
			repo.FS.Path(),
		)
	}
	if repoPath != repo.Path() {
		return nil, errors.Errorf(
			"cached repo %s is in cache at %s@%s instead of %s@%s",
			repo.Path(), repoPath, repo.Version, repo.Path(), repo.Version,
		)
	}
	return repo, nil
}

// LayeredRepoCache

// Path returns the path of the underlying cache.
func (c *LayeredRepoCache) Path() string {
	return c.Underlay.Path()
}

// LoadFSRepo loads the FSRepo with the specified path and version.
// The loaded FSRepo instance is fully initialized.
// If the overlay cache expects to have the repo, it will attempt to load the repo; otherwise,
// the underlay cache will attempt to load the repo.
func (c *LayeredRepoCache) LoadFSRepo(
	repoPath string, version string,
) (*core.FSRepo, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	if c.Overlay.IncludesFSRepo(repoPath, version) {
		repo, err := c.Overlay.LoadFSRepo(repoPath, version)
		return repo, errors.Wrap(err, "couldn't load repo from overlay")
	}
	repo, err := c.Underlay.LoadFSRepo(repoPath, version)
	return repo, errors.Wrap(err, "couldn't load repo from underlay")
}

// LoadFSRepos loads all FSRepos from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching repo directories to
// search for.
// The loaded FSRepo instances are fully initialized.
// All matching repos from the overlay cache will be included; all matching repos from the
// underlay cache will also be included, except for those repos which the overlay cache expected
// to have.
func (c *LayeredRepoCache) LoadFSRepos(searchPattern string) ([]*core.FSRepo, error) {
	if c == nil {
		return nil, nil
	}

	loadedRepos, err := c.Overlay.LoadFSRepos(searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load repos from overlay")
	}

	underlayRepos, err := c.Underlay.LoadFSRepos(searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load repos from underlay")
	}
	for _, repo := range underlayRepos {
		if c.Overlay.IncludesFSRepo(repo.Path(), repo.Version) {
			continue
		}
		loadedRepos = append(loadedRepos, repo)
	}

	sort.Slice(loadedRepos, func(i, j int) bool {
		return core.CompareRepos(loadedRepos[i].Repo, loadedRepos[j].Repo) == core.CompareLT
	})
	return loadedRepos, nil
}

// LoadFSPkg loads the FSPkg with the specified path and version.
// The loaded FSPkg instance is fully initialized.
// If the overlay cache expects to have the package, it will attempt to load the package; otherwise,
// the underlay cache will attempt to load the package.
func (c *LayeredRepoCache) LoadFSPkg(pkgPath string, version string) (*core.FSPkg, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	if c.Overlay.IncludesFSPkg(pkgPath, version) {
		pkg, err := c.Overlay.LoadFSPkg(pkgPath, version)
		return pkg, errors.Wrap(err, "couldn't load package from overlay")
	}

	pkg, err := c.Underlay.LoadFSPkg(pkgPath, version)
	return pkg, errors.Wrap(err, "couldn't load package from underlay")
}

// LoadFSPkgs loads all FSPkgs from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching package directories
// to search for.
// The loaded FSPkg instances are fully initialized.
// All matching packages from the overlay cache will be included; all matching packages from the
// underlay cache will also be included, except for those packages which the overlay cache expected
// to have.
func (c *LayeredRepoCache) LoadFSPkgs(searchPattern string) ([]*core.FSPkg, error) {
	if c == nil {
		return nil, nil
	}

	pkgs, err := c.Overlay.LoadFSPkgs(searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load packages from overlay")
	}

	underlayPkgs, err := c.Underlay.LoadFSPkgs(searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load packages from underlay")
	}
	for _, pkg := range underlayPkgs {
		if c.Overlay.IncludesFSPkg(pkg.Path(), pkg.Repo.Version) {
			continue
		}
		pkgs = append(pkgs, pkg)
	}

	sort.Slice(pkgs, func(i, j int) bool {
		return core.ComparePkgs(pkgs[i].Pkg, pkgs[j].Pkg) == core.CompareLT
	})
	return pkgs, nil
}

// RepoOverrideCache

// NewRepoOverrideCache instantiates a new RepoOverrideCache with a given list of repos, and a
// map associating repo paths with lists of versions which the respective repos will be
// associated with.
func NewRepoOverrideCache(
	overrideRepos []*core.FSRepo, repoVersions map[string][]string,
) (*RepoOverrideCache, error) {
	c := &RepoOverrideCache{
		repos:           make(map[string]*core.FSRepo),
		repoPaths:       make([]string, 0, len(overrideRepos)),
		repoVersions:    make(map[string][]string),
		repoVersionSets: make(map[string]structures.Set[string]),
	}
	for _, repo := range overrideRepos {
		repoPath := repo.Path()
		if _, ok := c.repos[repoPath]; ok {
			return nil, errors.Errorf("repo %s was provided multiple times", repoPath)
		}
		c.repos[repoPath] = repo
		c.repoPaths = append(c.repoPaths, repoPath)
		if repoVersions == nil {
			continue
		}

		c.repoVersions[repoPath] = append(c.repoVersions[repoPath], repoVersions[repoPath]...)
		sort.Strings(c.repoVersions[repoPath])
		if _, ok := c.repoVersionSets[repoPath]; !ok {
			c.repoVersionSets[repoPath] = make(structures.Set[string])
		}
		for _, version := range repoVersions[repoPath] {
			c.repoVersionSets[repoPath].Add(version)
		}
	}
	sort.Strings(c.repoPaths)
	return c, nil
}

// SetVersions configures the cache to cover the specified versions of the specified repo.
func (c *RepoOverrideCache) SetVersions(repoPath string, versions structures.Set[string]) {
	if _, ok := c.repoVersionSets[repoPath]; !ok {
		c.repoVersionSets[repoPath] = make(structures.Set[string])
	}
	sortedVersions := make([]string, 0, len(versions))
	for version := range versions {
		sortedVersions = append(sortedVersions, version)
		c.repoVersionSets[repoPath].Add(version)
	}
	sort.Strings(sortedVersions)
	c.repoVersions[repoPath] = sortedVersions
}

// RepoOverrideCache: OverlayCache

// IncludesFSRepo reports whether the RepoOverrideCache instance has a repo with the
// specified path and version.
func (c *RepoOverrideCache) IncludesFSRepo(repoPath string, version string) bool {
	if c == nil {
		return false
	}
	if _, ok := c.repos[repoPath]; !ok {
		return false
	}
	return c.repoVersionSets[repoPath].Has(version)
}

// LoadFSRepo loads the FSRepo with the specified path, if the version matches any of versions
// for the repo in the cache.
// The loaded FSRepo instance is fully initialized.
func (c *RepoOverrideCache) LoadFSRepo(repoPath string, version string) (*core.FSRepo, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	repo, ok := c.repos[repoPath]
	if !ok {
		return nil, errors.Errorf("couldn't find a repo with path %s", repoPath)
	}
	if !c.repoVersionSets[repoPath].Has(version) {
		return nil, errors.Errorf("found repo %s, but not with version %s", repoPath, version)
	}
	return repo, nil
}

// LoadFSRepos loads all FSRepos matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching repos to search
// for.
// The loaded FSRepo instances are fully initialized.
func (c *RepoOverrideCache) LoadFSRepos(searchPattern string) ([]*core.FSRepo, error) {
	if c == nil {
		return nil, nil
	}

	loadedRepos := make(map[string]*core.FSRepo) // indexed by repo cache path
	repoCachePaths := make([]string, 0)
	for _, repoPath := range c.repoPaths {
		repo := c.repos[repoPath]
		for _, version := range c.repoVersions[repoPath] {
			repoCachePath := fmt.Sprintf("%s@%s", repoPath, version)
			repoCachePaths = append(repoCachePaths, repoCachePath)
			loadedRepos[repoCachePath] = repo
		}
	}

	matchingRepoCachePaths := make([]string, 0, len(repoCachePaths))
	for _, cachePath := range repoCachePaths {
		ok, err := doublestar.Match(searchPattern, cachePath)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't search for repos using pattern %s", searchPattern)
		}
		if ok {
			matchingRepoCachePaths = append(matchingRepoCachePaths, cachePath)
		}
	}
	sort.Strings(matchingRepoCachePaths)

	matchingRepos := make([]*core.FSRepo, 0, len(matchingRepoCachePaths))
	for _, cachePath := range matchingRepoCachePaths {
		matchingRepos = append(matchingRepos, loadedRepos[cachePath])
	}
	return matchingRepos, nil
}

// IncludesFSPkg reports whether the RepoOverrideCache instance has a repo with the specified
// version which covers the specified package path.
func (c *RepoOverrideCache) IncludesFSPkg(pkgPath string, version string) bool {
	if c == nil {
		return false
	}

	// Beyond a certain number of repos, it's probably faster to just recurse down via the subdirs.
	// But we probably don't need to worry about this for now.
	for _, repo := range c.repos {
		if !core.CoversPath(repo, pkgPath) {
			continue
		}
		return c.repoVersionSets[repo.Path()].Has(version)
	}
	return false
}

// LoadFSPkg loads the FSPkg with the specified path, if the version matches any of versions for
// the package's repo in the cache.
// The loaded FSPkg instance is fully initialized.
func (c *RepoOverrideCache) LoadFSPkg(pkgPath string, version string) (*core.FSPkg, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	// Beyond a certain number of repos, it's probably faster to just recurse down via the subdirs.
	// But we probably don't need to worry about this for now.
	for _, repo := range c.repos {
		if !core.CoversPath(repo, pkgPath) {
			continue
		}
		if !c.repoVersionSets[repo.Path()].Has(version) {
			return nil, errors.Errorf(
				"found repo %s providing package %s, but not at version %s", repo.Path(), pkgPath, version,
			)
		}
		return repo.LoadFSPkg(core.GetSubdirPath(repo, pkgPath))
	}
	return nil, errors.Errorf("couldn't find a repo providing package %s", pkgPath)
}

// LoadFSPkgs loads all FSPkgs matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching package directories
// to search for.
// The loaded FSPkg instances are fully initialized.
func (c *RepoOverrideCache) LoadFSPkgs(searchPattern string) ([]*core.FSPkg, error) {
	if c == nil {
		return nil, nil
	}

	pkgs := make(map[string]*core.FSPkg) // indexed by package cache path
	pkgCachePaths := make([]string, 0)
	for _, repoPath := range c.repoPaths {
		repo := c.repos[repoPath]
		loaded, err := repo.LoadFSPkgs("**")
		if err != nil {
			return nil, errors.Errorf("couldn't list packages in repo %s", repo.Path())
		}
		for _, version := range c.repoVersions[repoPath] {
			for _, pkg := range loaded {
				pkgCachePath := path.Join(fmt.Sprintf("%s@%s", repoPath, version), pkg.Subdir)
				pkgCachePaths = append(pkgCachePaths, pkgCachePath)
				pkgs[pkgCachePath] = pkg
			}
		}
	}

	matchingPkgCachePaths := make([]string, 0, len(pkgCachePaths))
	for _, cachePath := range pkgCachePaths {
		ok, err := doublestar.Match(searchPattern, cachePath)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't search for packages using pattern %s", searchPattern)
		}
		if ok {
			matchingPkgCachePaths = append(matchingPkgCachePaths, cachePath)
		}
	}
	sort.Strings(matchingPkgCachePaths)

	matchingPkgs := make([]*core.FSPkg, 0, len(matchingPkgCachePaths))
	for _, cachePath := range matchingPkgCachePaths {
		matchingPkgs = append(matchingPkgs, pkgs[cachePath])
	}
	return matchingPkgs, nil
}
