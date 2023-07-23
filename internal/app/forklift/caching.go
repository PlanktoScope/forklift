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

// CoversPath checks whether the provided path is within the scope of the cache.
func (c *FSCache) CoversPath(path string) bool {
	return strings.HasPrefix(path, fmt.Sprintf("%s/", c.FS.Path()))
}

// TrimCachePathPrefix removes the path of the cache from the start of the provided path.
func (c *FSCache) TrimCachePathPrefix(path string) string {
	return strings.TrimPrefix(path, fmt.Sprintf("%s/", c.FS.Path()))
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
		c.TrimCachePathPrefix(repo.FS.Path()),
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
		repo, err := c.loadFSRepoContaining(c.TrimCachePathPrefix(pkg.FS.Path()))
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

// OverriddenCache

// LoadFSRepo loads the FSRepo with the specified path and version.
// The loaded FSRepo instance is fully initialized.
// If the repo path matches of the overriding FSRepos in the OverriddenCache instance,
// regardless of version, it is returned; otherwise, the repo is loaded from the underlying cache.
func (c *OverriddenCache) LoadFSRepo(repoPath string, version string) (*pallets.FSRepo, error) {
	if repo, ok := c.Overrides[repoPath]; ok {
		return repo, nil
	}
	return c.LoadFSRepo(repoPath, version)
}

// LoadFSRepos loads all FSRepos from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching repo directories to
// search for.
// The loaded FSRepo instances are fully initialized.
// If a matching repo is one of the overriding FSRepos in the OverriddenCache instance,
// regardless of version, it is included, and repos with the same path from the underlying cache are
// excluded regardless of version; otherwise, the repo is loaded from the underlying cache.
func (c *OverriddenCache) LoadFSRepos(searchPattern string) ([]*pallets.FSRepo, error) {
	repos := make([]*pallets.FSRepo, 0, len(c.Overrides))
	for _, repo := range c.Overrides {
		match, err := doublestar.Match(searchPattern, repo.Path())
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't match with search pattern %s", searchPattern)
		}
		if !match {
			continue
		}
		repos = append(repos, repo)
	}
	cachedRepos, err := c.Cache.LoadFSRepos(searchPattern)
	if err != nil {
		return nil, err
	}
	for _, repo := range cachedRepos {
		if _, ok := c.Overrides[repo.Path()]; ok {
			// We already added the repo in the overrides
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
// If the package path is covered by one of the overriding FSRepos in the OverriddenCache instance,
// regardless of version and regardless of whether that repo actually provides the package, that
// repo is used to try to load the package; otherwise, the package is loaded from the underlying
// cache.
func (c *OverriddenCache) LoadFSPkg(pkgPath string, version string) (*pallets.FSPkg, error) {
	for _, repo := range c.Overrides {
		if repo.CoversPath(pkgPath) {
			return repo.LoadFSPkg(repo.GetPkgSubdir(pkgPath))
		}
	}
	return c.LoadFSPkg(pkgPath, version)
}

// LoadFSPkgs loads all FSPkgs from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching package directories
// to search for.
// The loaded FSPkg instances are fully initialized.
// If one of the overriding FSRepos in the OverriddenCache instance provides matching packages,
// regardless of version, those packages are included; repos from the underlying cache with the same
// path as any overriding repo will not be used for loading packages; packages from other repos are
// loaded from the underlying cache.
func (c *OverriddenCache) LoadFSPkgs(searchPattern string) ([]*pallets.FSPkg, error) {
	pkgs := make([]*pallets.FSPkg, 0, len(c.Overrides))
	for _, repo := range c.Overrides {
		repoPkgs, err := repo.LoadFSPkgs(searchPattern)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load packages from overriding repo %s", repo.Path())
		}
		pkgs = append(pkgs, repoPkgs...)
	}
	cachedPkgs, err := c.Cache.LoadFSPkgs(searchPattern)
	if err != nil {
		return nil, err
	}
	for _, pkg := range cachedPkgs {
		if _, ok := c.Overrides[pkg.Repo.Path()]; ok {
			// We already added packages from the repo in the overrides
			continue
		}
		pkgs = append(pkgs, pkg)
	}

	sort.Slice(pkgs, func(i, j int) bool {
		return pallets.ComparePkgs(pkgs[i].Pkg, pkgs[j].Pkg) == pallets.CompareLT
	})
	return pkgs, nil
}
