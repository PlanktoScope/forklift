package forklift

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/pkg/errors"
)

func ListVersionedPkgs(cacheFS fs.FS, repos []VersionedRepo) ([]CachedPkg, error) {
	versionedPkgPaths := make([]string, 0)
	pkgMap := make(map[string]CachedPkg)
	for _, repo := range repos {
		repoVersion, err := repo.Version()
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't determine version of repo %s", repo.Path())
		}
		repoCachePath := filepath.Join(
			fmt.Sprintf("%s@%s", repo.VCSRepoPath, repoVersion),
			repo.RepoSubdir,
		)
		pkgs, err := ListCachedPkgs(cacheFS, repoCachePath)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't list cached packages for repo cached at %s", repoCachePath,
			)
		}

		for _, pkg := range pkgs {
			versionedPkgPath := fmt.Sprintf(
				"%s@%s/%s", pkg.Repo.Config.Repository.Path, pkg.Repo.Version, pkg.PkgSubdir,
			)
			if prevPkg, ok := pkgMap[versionedPkgPath]; ok {
				if prevPkg.Repo.FromSameVCSRepo(pkg.Repo) && prevPkg.ConfigPath != pkg.ConfigPath {
					return nil, errors.Errorf(
						"package repeatedly defined in the same version of the same cached Github repo: %s, %s",
						prevPkg.ConfigPath, pkg.ConfigPath,
					)
				}
			}
			versionedPkgPaths = append(versionedPkgPaths, versionedPkgPath)
			pkgMap[versionedPkgPath] = pkg
		}
	}

	orderedPkgs := make([]CachedPkg, 0, len(versionedPkgPaths))
	for _, path := range versionedPkgPaths {
		orderedPkgs = append(orderedPkgs, pkgMap[path])
	}
	return orderedPkgs, nil
}

func FindVersionedRepoOfPkg(envFS fs.FS, pkgPath string) (VersionedRepo, error) {
	reposFS, err := VersionedReposFS(envFS)
	if err != nil {
		return VersionedRepo{}, errors.Wrap(
			err, "couldn't open directory for Pallet repositories in local environment",
		)
	}

	repoCandidatePath := filepath.Dir(pkgPath)
	for repoCandidatePath != "." {
		repo, err := LoadVersionedRepo(reposFS, repoCandidatePath)
		if err == nil {
			return repo, nil
		}
		repoCandidatePath = filepath.Dir(repoCandidatePath)
	}
	return VersionedRepo{}, errors.Errorf(
		"no repository config file found in %s or any parent directory in local environment",
		filepath.Dir(pkgPath),
	)
}
