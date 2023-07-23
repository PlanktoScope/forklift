package forklift

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// RepoVersionRequirement

func (r RepoVersionRequirement) Path() string {
	return filepath.Join(r.VCSRepoPath, r.RepoSubdir)
}

// TODO: rename this method
func (r RepoVersionRequirement) listVersionedPkgs(
	cache *FSCache,
) (pkgMap map[string]*pallets.FSPkg, versionedPkgPaths []string, err error) {
	repoCachePath := filepath.Join(
		fmt.Sprintf("%s@%s", r.VCSRepoPath, r.VersionLock.Version), r.RepoSubdir,
	)
	pkgs, err := cache.LoadFSPkgs(repoCachePath)
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

func CompareRepoVersionRequirements(r, s RepoVersionRequirement) int {
	if r.VCSRepoPath != s.VCSRepoPath {
		if r.VCSRepoPath < s.VCSRepoPath {
			return pallets.CompareLT
		}
		return pallets.CompareGT
	}
	if r.RepoSubdir != s.RepoSubdir {
		if r.RepoSubdir < s.RepoSubdir {
			return pallets.CompareLT
		}
		return pallets.CompareGT
	}
	return pallets.CompareEQ
}
