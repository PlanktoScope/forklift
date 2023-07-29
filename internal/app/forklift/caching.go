package forklift

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// FSPkgLoader

// LoadFSPkgFromPkgReq loads the required package from the cache.
func LoadFSPkgFromPkgReq(loader FSPkgLoader, req PkgReq) (*pallets.FSPkg, error) {
	pkg, err := loader.LoadFSPkg(req.Path(), req.Repo.VersionLock.Version)
	return pkg, errors.Wrapf(
		err, "couldn't load required package %s@%s", req.Path(), req.Repo.VersionLock.Version,
	)
}

// LoadFSPkgsFromPkgReqs loads the required packages from the cache.
func LoadFSPkgsFromPkgReqs(loader FSPkgLoader, reqs []PkgReq) (p []*pallets.FSPkg, err error) {
	for _, req := range reqs {
		fsPkg, err := LoadFSPkgFromPkgReq(loader, req)
		if err != nil {
			return nil, err
		}
		p = append(p, fsPkg)
	}
	return p, nil
}

// FSCache

// Exists checks whether the cache actually exists on the OS's filesystem.
func (c *FSCache) Exists() bool {
	return Exists(c.FS.Path())
}

// Remove deletes the cache from the OS's filesystem, if it exists.
func (c *FSCache) Remove() error {
	return os.RemoveAll(c.FS.Path())
}

// Path returns the path of the cache
func (c *FSCache) Path() string {
	return c.FS.Path()
}

// FSCache: FSRepoLoader

// LoadFSRepo loads the FSRepo with the specified path and version.
// The loaded FSRepo instance is fully initialized.
func (c *FSCache) LoadFSRepo(repoPath string, version string) (*pallets.FSRepo, error) {
	vcsRepoPath, _, err := pallets.SplitRepoPathSubdir(repoPath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't parse path of Pallet repo %s", repoPath)
	}
	// The repo subdirectory path in the repo path (under the VCS repo path) might not match the
	// filesystem directory path with the pallet-repository.yml file, so we must check every
	// pallet-repository.yml file to find the actual repo path
	searchPattern := fmt.Sprintf("%s@%s/**", vcsRepoPath, version)
	repos, err := c.LoadFSRepos(searchPattern)
	if err != nil {
		return nil, err
	}

	candidateRepos := make([]*pallets.FSRepo, 0)
	for _, repo := range repos {
		if repo.Path() != repoPath {
			continue
		}

		if len(candidateRepos) > 0 {
			return nil, errors.Errorf(
				"version %s of repository %s was found in multiple different locations: %s, %s",
				version, repoPath, candidateRepos[0].FS.Path(), repo.FS.Path(),
			)
		}
		candidateRepos = append(candidateRepos, repo)
	}
	if len(candidateRepos) == 0 {
		return nil, errors.Errorf("no cached repos were found matching %s@%s", repoPath, version)
	}
	return candidateRepos[0], nil
}

// LoadFSRepos loads all FSRepos from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching repo directories to
// search for.
// The loaded FSRepo instances are fully initialized.
func (c *FSCache) LoadFSRepos(searchPattern string) ([]*pallets.FSRepo, error) {
	repos, err := pallets.LoadFSRepos(c.FS, searchPattern, c.processLoadedFSRepo)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load repos from cache")
	}

	return repos, nil
}

// processLoadedFSRepo sets the Version field of the repo based on its path in the cache.
func (c *FSCache) processLoadedFSRepo(repo *pallets.FSRepo) (err error) {
	var vcsRepoPath string
	if vcsRepoPath, repo.Version, err = splitRepoPathVersion(
		pallets.GetSubdirPath(c, repo.FS.Path()),
	); err != nil {
		return errors.Wrapf(
			err, "couldn't parse path of cached repo configured at %s", repo.FS.Path(),
		)
	}
	if vcsRepoPath != repo.VCSRepoPath {
		return errors.Errorf(
			"cached repo %s is in cache at %s@%s instead of %s@%s",
			repo.Path(), vcsRepoPath, repo.Version, repo.VCSRepoPath, repo.Version,
		)
	}
	return nil
}

// splitRepoPathVersion splits paths of form github.com/user-name/git-repo-name/etc@version into
// github.com/user-name/git-repo-name and version.
func splitRepoPathVersion(repoPath string) (vcsRepoPath, version string, err error) {
	const sep = "/"
	pathParts := strings.Split(repoPath, sep)
	if pathParts[0] != "github.com" {
		return "", "", errors.Errorf(
			"pallet repo %s does not begin with github.com, and handling of non-Github repositories is "+
				"not yet implemented",
			repoPath,
		)
	}
	vcsRepoName, version, ok := strings.Cut(pathParts[2], "@")
	if !ok {
		return "", "", errors.Errorf(
			"Couldn't parse Github repository name %s as name@version", pathParts[2],
		)
	}
	vcsRepoPath = strings.Join([]string{pathParts[0], pathParts[1], vcsRepoName}, sep)
	return vcsRepoPath, version, nil
}

// FSCache: FSPkgLoader

// LoadFSPkg loads the FSPkg with the specified path and version.
// The loaded FSPkg instance is fully initialized.
func (c *FSCache) LoadFSPkg(pkgPath string, version string) (*pallets.FSPkg, error) {
	vcsRepoPath, _, err := pallets.SplitRepoPathSubdir(pkgPath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't parse path of Pallet package %s", pkgPath)
	}
	pkgInnermostDir := filepath.Base(pkgPath)
	// The package subdirectory path in the package path (under the VCS repo path) might not match the
	// filesystem directory path with the pallet-package.yml file, so we must check every
	// directory whose name matches the last part of the package path to look for the package
	searchPattern := fmt.Sprintf("%s@%s/**/%s", vcsRepoPath, version, pkgInnermostDir)
	pkgs, err := c.LoadFSPkgs(searchPattern)
	if err != nil {
		return nil, err
	}

	candidatePkgs := make([]*pallets.FSPkg, 0)
	for _, pkg := range pkgs {
		if pkg.Path() != pkgPath {
			continue
		}

		if len(candidatePkgs) > 0 {
			return nil, errors.Errorf(
				"version %s of package %s was found in multiple different locations: %s, %s",
				version, pkgPath, candidatePkgs[0].FS.Path(), pkg.FS.Path(),
			)
		}
		candidatePkgs = append(candidatePkgs, pkg)
	}
	if len(candidatePkgs) == 0 {
		return nil, errors.Errorf("no cached packages were found matching %s@%s", pkgPath, version)
	}
	return candidatePkgs[0], nil
}

// LoadFSPkgs loads all FSPkgs from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching package directories
// to search for.
// The loaded FSPkg instances are fully initialized.
func (c *FSCache) LoadFSPkgs(searchPattern string) ([]*pallets.FSPkg, error) {
	pkgs, err := pallets.LoadFSPkgs(c.FS, searchPattern)
	if err != nil {
		return nil, err
	}

	pkgMap := make(map[string]*pallets.FSPkg)
	for _, pkg := range pkgs {
		repo, err := c.loadFSRepoContaining(pallets.GetSubdirPath(c, pkg.FS.Path()))
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't find the cached repo providing the package at %s", pkg.FS.Path(),
			)
		}
		if err = pkg.AttachFSRepo(repo); err != nil {
			return nil, errors.Wrap(err, "couldn't attach repo to package")
		}
		versionedPkgPath := fmt.Sprintf(
			"%s@%s/%s", pkg.Repo.Config.Repository.Path, pkg.Repo.Version, pkg.Subdir,
		)
		if prevPkg, ok := pkgMap[versionedPkgPath]; ok {
			if prevPkg.Repo.FromSameVCSRepo(pkg.Repo.Repo) && prevPkg.FS.Path() != pkg.FS.Path() {
				return nil, errors.Errorf(
					"the same version of package %s was found in multiple different locations: %s, %s",
					pkg.Path(), prevPkg.FS.Path(), pkg.FS.Path(),
				)
			}
		}
		pkgMap[versionedPkgPath] = pkg
	}

	return pkgs, nil
}

// loadFSRepoContaining finds and loads the FSRepo which contains the provided subdirectory path.
func (c *FSCache) loadFSRepoContaining(subdirPath string) (repo *pallets.FSRepo, err error) {
	if repo, err = pallets.LoadFSRepoContaining(c.FS, subdirPath); err != nil {
		return nil, errors.Wrapf(err, "couldn't find any repo containing %s", subdirPath)
	}
	if err = c.processLoadedFSRepo(repo); err != nil {
		return nil, err
	}
	return repo, nil
}

// LayeredCache

// LoadFSRepo loads the FSRepo with the specified path and version.
// The loaded FSRepo instance is fully initialized.
// If the overlay cache expects to have the repo, it will attempt to load the repo; otherwise, the
// underlay cache will attempt to load the repo.
func (c *LayeredCache) LoadFSRepo(repoPath string, version string) (*pallets.FSRepo, error) {
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
// All matching repos from the overlay cache will be included; all matching repos from the underlay
// cache will also be included, except for those repos which the overlay cache expected to have.
func (c *LayeredCache) LoadFSRepos(searchPattern string) ([]*pallets.FSRepo, error) {
	repos, err := c.Overlay.LoadFSRepos(searchPattern)
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
		repos = append(repos, repo)
	}

	sort.Slice(repos, func(i, j int) bool {
		return pallets.CompareRepos(repos[i].Repo, repos[j].Repo) == pallets.CompareLT
	})
	return repos, nil
}

// LoadFSPkg loads the FSPkg with the specified path and version.
// The loaded FSPkg instance is fully initialized.
// If the overlay cache expects to have the package, it will attempt to load the package; otherwise,
// the underlay cache will attempt to load the package.
func (c *LayeredCache) LoadFSPkg(pkgPath string, version string) (*pallets.FSPkg, error) {
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
func (c *LayeredCache) LoadFSPkgs(searchPattern string) ([]*pallets.FSPkg, error) {
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
		return pallets.ComparePkgs(pkgs[i].Pkg, pkgs[j].Pkg) == pallets.CompareLT
	})
	return pkgs, nil
}

// Path returns the path of the underlying cache
func (c *LayeredCache) Path() string {
	return c.Underlay.Path()
}

// RepoOverrideCache

func NewRepoOverrideCache(
	repos []*pallets.FSRepo, repoVersions map[string][]string,
) (*RepoOverrideCache, error) {
	c := &RepoOverrideCache{
		repos:           make(map[string]*pallets.FSRepo),
		repoPaths:       make([]string, 0, len(repos)),
		repoVersions:    make(map[string][]string),
		repoVersionSets: make(map[string]map[string]struct{}),
	}
	for _, repo := range repos {
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
			c.repoVersionSets[repoPath] = make(map[string]struct{})
		}
		for _, version := range repoVersions[repoPath] {
			c.repoVersionSets[repoPath][version] = struct{}{}
		}
	}
	sort.Strings(c.repoPaths)
	return c, nil
}

func (f *RepoOverrideCache) SetVersions(repoPath string, versions map[string]struct{}) {
	if _, ok := f.repoVersionSets[repoPath]; !ok {
		f.repoVersionSets[repoPath] = make(map[string]struct{})
	}
	sortedVersions := make([]string, 0, len(versions))
	for version := range versions {
		sortedVersions = append(sortedVersions, version)
		f.repoVersionSets[repoPath][version] = struct{}{}
	}
	sort.Strings(sortedVersions)
	f.repoVersions[repoPath] = sortedVersions
}

// RepoOverrideCache: OverlayCache

// IncludesFSRepo reports whether the RepoOverrideCache instance has a repository with the
// specified path and version.
func (f *RepoOverrideCache) IncludesFSRepo(repoPath string, version string) bool {
	_, ok := f.repos[repoPath]
	if !ok {
		return false
	}
	_, ok = f.repoVersionSets[repoPath][version]
	return ok
}

// LoadFSRepo loads the FSRepo with the specified path, if the version matches any of versions for
// the repo in the cache.
// The loaded FSRepo instance is fully initialized.
func (f *RepoOverrideCache) LoadFSRepo(repoPath string, version string) (*pallets.FSRepo, error) {
	repo, ok := f.repos[repoPath]
	if !ok {
		return nil, errors.Errorf("couldn't find a repo with path %s", repoPath)
	}
	_, ok = f.repoVersionSets[repoPath][version]
	if !ok {
		return nil, errors.Errorf("found repo %s, but not with version %s", repoPath, version)
	}
	return repo, nil
}

// LoadFSRepos loads all FSRepos matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching repos to search for.
// The loaded FSRepo instances are fully initialized.
func (f *RepoOverrideCache) LoadFSRepos(searchPattern string) ([]*pallets.FSRepo, error) {
	versionedRepos := make(map[string]*pallets.FSRepo)
	versionedRepoPaths := make([]string, 0)
	for _, repoPath := range f.repoPaths {
		repo := f.repos[repoPath]
		for _, version := range f.repoVersions[repoPath] {
			vcsRepoPath, repoSubdir, err := pallets.SplitRepoPathSubdir(repoPath)
			if err != nil {
				return nil, errors.Wrapf(err, "couldn't parse path of Pallet repo %s", repoPath)
			}
			versionedRepoPath := fmt.Sprintf("%s@%s/%s", vcsRepoPath, version, repoSubdir)
			versionedRepoPaths = append(versionedRepoPaths, versionedRepoPath)
			versionedRepos[versionedRepoPath] = repo
		}
	}

	matchingVersionedRepoPaths := make([]string, 0, len(versionedRepoPaths))
	for _, path := range versionedRepoPaths {
		ok, err := doublestar.Match(searchPattern, path)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't search for repositories using pattern %s", searchPattern,
			)
		}
		if ok {
			matchingVersionedRepoPaths = append(matchingVersionedRepoPaths, path)
		}
	}
	sort.Strings(matchingVersionedRepoPaths)

	matchingRepos := make([]*pallets.FSRepo, 0, len(matchingVersionedRepoPaths))
	for _, path := range matchingVersionedRepoPaths {
		matchingRepos = append(matchingRepos, versionedRepos[path])
	}
	return matchingRepos, nil
}

// IncludesFSPkg reports whether the RepoOverrideCache instance has a repository with the specified
// version which covers the specified package path.
func (f *RepoOverrideCache) IncludesFSPkg(pkgPath string, version string) bool {
	// Beyond a certain number of repos, it's probably faster to just recurse down via the subdirs.
	// But we probably don't need to worry about this for now.
	for _, repo := range f.repos {
		if !pallets.CoversPath(repo, pkgPath) {
			continue
		}
		_, ok := f.repoVersionSets[repo.Path()][version]
		return ok
	}
	return false
}

// LoadFSPkg loads the FSPkg with the specified path, if the version matches any of versions for
// the package's repo in the cache.
// The loaded FSPkg instance is fully initialized.
func (f *RepoOverrideCache) LoadFSPkg(pkgPath string, version string) (*pallets.FSPkg, error) {
	// Beyond a certain number of repos, it's probably faster to just recurse down via the subdirs.
	// But we probably don't need to worry about this for now.
	for _, repo := range f.repos {
		if !pallets.CoversPath(repo, pkgPath) {
			continue
		}
		_, ok := f.repoVersionSets[repo.Path()][version]
		if !ok {
			return nil, errors.Errorf(
				"found repo %s providing package %s, but not at version %s",
				repo.Path(), pkgPath, version,
			)
		}
		return repo.LoadFSPkg(pallets.GetSubdirPath(repo, pkgPath))
	}
	return nil, errors.Errorf("couldn't find a repo providing package %s", pkgPath)
}

// LoadFSPkgs loads all FSPkgs matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching package directories
// to search for.
// The loaded FSPkg instances are fully initialized.
func (f *RepoOverrideCache) LoadFSPkgs(searchPattern string) ([]*pallets.FSPkg, error) {
	versionedPkgs := make(map[string]*pallets.FSPkg)
	versionedPkgPaths := make([]string, 0)
	for _, repoPath := range f.repoPaths {
		repo := f.repos[repoPath]
		pkgs, err := repo.LoadFSPkgs("**")
		if err != nil {
			return nil, errors.Errorf("couldn't list packages in repo %s", repo.Path())
		}
		for _, version := range f.repoVersions[repoPath] {
			vcsRepoPath, repoSubdir, err := pallets.SplitRepoPathSubdir(repoPath)
			if err != nil {
				return nil, errors.Wrapf(err, "couldn't parse path of Pallet repo %s", repoPath)
			}
			for _, pkg := range pkgs {
				versionedPkgPath := fmt.Sprintf("%s@%s/%s/%s", vcsRepoPath, version, repoSubdir, pkg.Subdir)
				versionedPkgPaths = append(versionedPkgPaths, versionedPkgPath)
				versionedPkgs[versionedPkgPath] = pkg
			}
		}
	}

	matchingVersionedPkgPaths := make([]string, 0, len(versionedPkgPaths))
	for _, path := range versionedPkgPaths {
		ok, err := doublestar.Match(searchPattern, path)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't search for packages using pattern %s", searchPattern)
		}
		if ok {
			matchingVersionedPkgPaths = append(matchingVersionedPkgPaths, path)
		}
	}
	sort.Strings(matchingVersionedPkgPaths)

	matchingPkgs := make([]*pallets.FSPkg, 0, len(matchingVersionedPkgPaths))
	for _, path := range matchingVersionedPkgPaths {
		matchingPkgs = append(matchingPkgs, versionedPkgs[path])
	}
	return matchingPkgs, nil
}
