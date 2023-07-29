package forklift

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// TODO: refactor and rename this method
func ListVersionedPkgsOfRepo(
	loader FSPkgLoader, req RepoReq,
) (pkgMap map[string]*pallets.FSPkg, versionedPkgPaths []string, err error) {
	repoCachePath := filepath.Join(
		fmt.Sprintf("%s@%s", req.VCSRepoPath, req.VersionLock.Version), req.RepoSubdir,
	)
	pkgs, err := loader.LoadFSPkgs(filepath.Join(repoCachePath, "**"))
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

// TODO: refactor and rename this method
func ListVersionedPkgsOfRepos(loader FSPkgLoader, reqs []*FSRepoReq) ([]*pallets.FSPkg, error) {
	versionedPkgPaths := make([]string, 0)
	pkgMap := make(map[string]*pallets.FSPkg)
	for _, req := range reqs {
		pkgs, paths, err := ListVersionedPkgsOfRepo(loader, req.RepoReq)
		for k, v := range pkgs {
			pkgMap[k] = v
		}
		versionedPkgPaths = append(versionedPkgPaths, paths...)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't list versioned packages of repo %s", req.Path())
		}
	}

	orderedPkgs := make([]*pallets.FSPkg, 0, len(versionedPkgPaths))
	for _, path := range versionedPkgPaths {
		orderedPkgs = append(orderedPkgs, pkgMap[path])
	}
	return orderedPkgs, nil
}
