package forklift

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	gosemver "golang.org/x/mod/semver"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Loading

func FindCachedPkg(cacheFS fs.FS, pkgPath string, version string) (CachedPkg, error) {
	vcsRepoPath, _, err := SplitRepoPathSubdir(pkgPath)
	if err != nil {
		return CachedPkg{}, errors.Wrapf(err, "couldn't parse path of Pallet package %s", pkgPath)
	}
	pkgInnermostDir := filepath.Base(pkgPath)
	// The package subdirectory path in the package path (under the VCS repo path) might not match the
	// filesystem directory path with the pallet-package.yml file, so we must check every
	// directory whose name matches the last part of the package path to look for the package
	searchPattern := fmt.Sprintf(
		"%s@%s/**/%s/%s", vcsRepoPath, version, pkgInnermostDir, pallets.PkgSpecFile,
	)
	candidatePkgConfigFiles, err := doublestar.Glob(cacheFS, searchPattern)
	if err != nil {
		return CachedPkg{}, errors.Wrapf(
			err, "couldn't search for cached Pallet package configs matching pattern %s", searchPattern,
		)
	}
	if len(candidatePkgConfigFiles) == 0 {
		return CachedPkg{}, errors.Errorf(
			"no matching Pallet package configs were found in %s@%s", vcsRepoPath, version,
		)
	}
	candidatePkgs := make([]CachedPkg, 0)
	for _, pkgConfigFilePath := range candidatePkgConfigFiles {
		if filepath.Base(pkgConfigFilePath) != pallets.PkgSpecFile {
			continue
		}

		pkg, err := loadCachedPkg(cacheFS, filepath.Dir(pkgConfigFilePath))
		if err != nil {
			return CachedPkg{}, errors.Wrapf(
				err, "couldn't check cached pkg defined at %s", pkgConfigFilePath,
			)
		}
		if pkg.Path() != pkgPath {
			continue
		}

		if len(candidatePkgs) > 0 {
			return CachedPkg{}, errors.Errorf(
				"package %s repeatedly defined in the same version of the same Github repo: %s, %s",
				pkgPath, candidatePkgs[0].FS.Path(), pkg.FS.Path(),
			)
		}
		candidatePkgs = append(candidatePkgs, pkg)
	}
	if len(candidatePkgs) == 0 {
		return CachedPkg{}, errors.Errorf(
			"no cached packages were found matching %s@%s", pkgPath, version,
		)
	}
	return candidatePkgs[0], nil
}

func loadCachedPkg(cacheFS fs.FS, subdirPath string) (CachedPkg, error) {
	repo, err := findRepoContaining(cacheFS, subdirPath)
	if err != nil {
		return CachedPkg{}, errors.Wrapf(
			err, "couldn't identify cached repository for package from %s", subdirPath,
		)
	}
	return loadPkgFromRepo(repo, strings.TrimPrefix(subdirPath, fmt.Sprintf("%s/", repo.FS.Path())))
}

func findRepoContaining(cacheFS fs.FS, subdirPath string) (pallets.FSRepo, error) {
	repoCandidatePath := subdirPath
	for repoCandidatePath != "." {
		repo, err := loadCachedRepo(cacheFS, repoCandidatePath)
		if err == nil {
			return repo, nil
		}
		repoCandidatePath = filepath.Dir(repoCandidatePath)
	}
	return pallets.FSRepo{}, errors.Errorf(
		"no repository config file found in any parent directory of %s", subdirPath,
	)
}

// Listing

func ListCachedPkgs(cacheFS fs.FS, cachedPrefix string) ([]CachedPkg, error) {
	searchPattern := fmt.Sprintf("**/%s", pallets.PkgSpecFile)
	if cachedPrefix != "" {
		searchPattern = filepath.Join(cachedPrefix, searchPattern)
	}
	pkgConfigFiles, err := doublestar.Glob(cacheFS, searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't search for cached package configs")
	}

	repoVersionedPkgPaths := make([]string, 0, len(pkgConfigFiles))
	pkgMap := make(map[string]CachedPkg)
	for _, pkgConfigFilePath := range pkgConfigFiles {
		if filepath.Base(pkgConfigFilePath) != pallets.PkgSpecFile {
			continue
		}
		pkg, err := loadCachedPkg(cacheFS, filepath.Dir(pkgConfigFilePath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load cached package from %s", pkgConfigFilePath)
		}

		versionedPkgPath := fmt.Sprintf(
			"%s@%s/%s", pkg.Repo.Config.Repository.Path, pkg.Repo.Version, pkg.Subdir,
		)
		if prevPkg, ok := pkgMap[versionedPkgPath]; ok {
			if prevPkg.Repo.FromSameVCSRepo(pkg.Repo.Repo) && prevPkg.FS.Path() != pkg.FS.Path() {
				return nil, errors.Errorf(
					"package repeatedly defined in the same version of the same Github repo: %s, %s",
					prevPkg.FS.Path(), pkg.FS.Path(),
				)
			}
		}
		repoVersionedPkgPaths = append(repoVersionedPkgPaths, versionedPkgPath)
		pkgMap[versionedPkgPath] = pkg
	}

	orderedPkgs := make([]CachedPkg, 0, len(repoVersionedPkgPaths))
	for _, path := range repoVersionedPkgPaths {
		orderedPkgs = append(orderedPkgs, pkgMap[path])
	}
	return orderedPkgs, nil
}

// Sorting

func CompareCachedPkgs(p, q CachedPkg) int {
	repoPathComparison := pallets.CompareRepoPaths(p.Repo.Repo, q.Repo.Repo)
	if repoPathComparison != pallets.CompareEQ {
		return repoPathComparison
	}
	if p.Subdir != q.Subdir {
		if p.Subdir < q.Subdir {
			return pallets.CompareLT
		}
		return pallets.CompareGT
	}
	repoVersionComparison := gosemver.Compare(p.Repo.Version, q.Repo.Version)
	if repoVersionComparison != pallets.CompareEQ {
		return repoVersionComparison
	}
	return pallets.CompareEQ
}
