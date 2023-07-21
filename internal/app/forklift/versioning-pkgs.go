package forklift

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Loading

func LoadVersionedPkg(reposFS, cacheFS fs.FS, pkgPath string) (VersionedPkg, error) {
	repo, err := findVersionedRepoOfPkg(reposFS, pkgPath)
	if err != nil {
		return VersionedPkg{}, errors.Wrapf(
			err, "couldn't find repo providing package %s in local environment", pkgPath,
		)
	}
	version, err := repo.Config.Version()
	if err != nil {
		return VersionedPkg{}, errors.Wrapf(
			err, "couldn't determine version of repo %s in local environment", repo.Path(),
		)
	}
	pkg, err := FindCachedPkg(cacheFS, pkgPath, version)
	if err != nil {
		return VersionedPkg{}, errors.Wrapf(
			err, "couldn't find package %s@%s in cache", pkgPath, version,
		)
	}

	return VersionedPkg{
		Path:   pkgPath,
		Repo:   repo,
		Cached: pkg,
	}, nil
}

func findVersionedRepoOfPkg(reposFS fs.FS, pkgPath string) (VersionedRepo, error) {
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

// Listing

func ListVersionedPkgs(
	cacheFS fs.FS, replacementRepos map[string]*pallets.FSRepo, repos []VersionedRepo,
) (orderedPkgs []*pallets.FSPkg, err error) {
	versionedPkgPaths := make([]string, 0)
	pkgMap := make(map[string]*pallets.FSPkg)
	for _, repo := range repos {
		var pkgs map[string]*pallets.FSPkg
		var paths []string
		if externalRepo, ok := replacementRepos[repo.Path()]; ok {
			pkgs, paths, err = listVersionedPkgsOfExternalRepo(externalRepo)
		} else {
			pkgs, paths, err = listVersionedPkgsOfCachedRepo(cacheFS, repo)
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

func listVersionedPkgsOfCachedRepo(
	cacheFS fs.FS, repo VersionedRepo,
) (pkgMap map[string]*pallets.FSPkg, versionedPkgPaths []string, err error) {
	repoVersion, err := repo.Config.Version()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't determine version of repo %s", repo.Path())
	}
	repoCachePath := filepath.Join(
		fmt.Sprintf("%s@%s", repo.VCSRepoPath, repoVersion),
		repo.RepoSubdir,
	)
	pkgs, err := ListCachedPkgs(cacheFS, repoCachePath)
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
