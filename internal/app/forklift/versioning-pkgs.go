package forklift

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

func ListVersionedPkgs(
	cache *FSCache, replacementRepos map[string]*pallets.FSRepo, repos []VersionedRepo,
) (orderedPkgs []*pallets.FSPkg, err error) {
	versionedPkgPaths := make([]string, 0)
	pkgMap := make(map[string]*pallets.FSPkg)
	for _, repo := range repos {
		var pkgs map[string]*pallets.FSPkg
		var paths []string
		if externalRepo, ok := replacementRepos[repo.Path()]; ok {
			pkgs, paths, err = listVersionedPkgsOfExternalRepo(externalRepo)
		} else {
			pkgs, paths, err = repo.listVersionedPkgs(cache)
		}

		for k, v := range pkgs {
			pkgMap[k] = v
		}
		versionedPkgPaths = append(versionedPkgPaths, paths...)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't list versioned packages of repo %s", repo.Path())
		}
	}

	orderedPkgs = make([]*pallets.FSPkg, 0, len(versionedPkgPaths))
	for _, path := range versionedPkgPaths {
		orderedPkgs = append(orderedPkgs, pkgMap[path])
	}
	return orderedPkgs, nil
}

// TODO: move this into a method on pallets.FSRepo
func listVersionedPkgsOfExternalRepo(
	externalRepo *pallets.FSRepo,
) (pkgMap map[string]*pallets.FSPkg, versionedPkgPaths []string, err error) {
	pkgs, err := ListExternalPkgs(externalRepo, "")
	if err != nil {
		return nil, nil, errors.Wrapf(
			err, "couldn't list packages from external repo at %s", externalRepo.FS.Path(),
		)
	}

	pkgMap = make(map[string]*pallets.FSPkg)
	for _, pkg := range pkgs {
		if prevPkg, ok := pkgMap[pkg.Path()]; ok {
			if prevPkg.Repo.FromSameVCSRepo(pkg.Repo.Repo) && prevPkg.FS.Path() != pkg.FS.Path() {
				return nil, nil, errors.Errorf(
					"package repeatedly defined in the same version of the same cached Github repo: %s, %s",
					prevPkg.FS.Path(), pkg.FS.Path(),
				)
			}
		}
		versionedPkgPaths = append(versionedPkgPaths, pkg.Path())
		pkgMap[pkg.Path()] = pkg
	}

	return pkgMap, versionedPkgPaths, nil
}

func (r VersionedRepo) listVersionedPkgs(
	cache *FSCache,
) (pkgMap map[string]*pallets.FSPkg, versionedPkgPaths []string, err error) {
	repoVersion, err := r.Config.Version()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't determine version of repo %s", r.Path())
	}
	repoCachePath := filepath.Join(fmt.Sprintf("%s@%s", r.VCSRepoPath, repoVersion), r.RepoSubdir)
	pkgs, err := cache.ListPkgs(repoCachePath)
	if err != nil {
		return nil, nil, errors.Wrapf(
			err, "couldn't list packages from repo cached at %s", repoCachePath,
		)
	}

	pkgMap = make(map[string]*pallets.FSPkg)
	for _, pkg := range pkgs {
		versionedPkgPath := fmt.Sprintf(
			"%s@%s/%s", pkg.Repo.Config.Repository.Path, pkg.Repo.Version, pkg.Subdir,
		)
		if prevPkg, ok := pkgMap[versionedPkgPath]; ok {
			if prevPkg.Repo.FromSameVCSRepo(pkg.Repo.Repo) && prevPkg.FS.Path() != pkg.FS.Path() {
				return nil, nil, errors.Errorf(
					"package repeatedly defined in the same version of the same cached Github repo: %s, %s",
					prevPkg.FS.Path(), pkg.FS.Path(),
				)
			}
		}
		versionedPkgPaths = append(versionedPkgPaths, versionedPkgPath)
		pkgMap[versionedPkgPath] = pkg
	}

	return pkgMap, versionedPkgPaths, nil
}
