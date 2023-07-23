package forklift

import (
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// TODO: rename this method
func ListVersionedPkgs(
	cache *FSCache, replacementRepos map[string]*pallets.FSRepo, requirements []*FSRepoReq,
) (orderedPkgs []*pallets.FSPkg, err error) {
	versionedPkgPaths := make([]string, 0)
	pkgMap := make(map[string]*pallets.FSPkg)
	for _, requirement := range requirements {
		var pkgs map[string]*pallets.FSPkg
		var paths []string
		if externalRepo, ok := replacementRepos[requirement.Path()]; ok {
			pkgs, paths, err = listVersionedPkgsOfExternalRepo(externalRepo)
		} else {
			pkgs, paths, err = cache.listVersionedPkgs(requirement.RepoReq)
		}

		for k, v := range pkgs {
			pkgMap[k] = v
		}
		versionedPkgPaths = append(versionedPkgPaths, paths...)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't list versioned packages of repo %s", requirement.Path(),
			)
		}
	}

	orderedPkgs = make([]*pallets.FSPkg, 0, len(versionedPkgPaths))
	for _, path := range versionedPkgPaths {
		orderedPkgs = append(orderedPkgs, pkgMap[path])
	}
	return orderedPkgs, nil
}

// TODO: rename this method
func listVersionedPkgsOfExternalRepo(
	externalRepo *pallets.FSRepo,
) (pkgMap map[string]*pallets.FSPkg, versionedPkgPaths []string, err error) {
	pkgs, err := externalRepo.LoadFSPkgs("**")
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
