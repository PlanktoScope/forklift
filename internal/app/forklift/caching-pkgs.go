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
		filename := filepath.Base(pkgConfigFilePath)
		if filename != pallets.PkgSpecFile {
			continue
		}

		pkg, err := loadCachedPkg(cacheFS, filepath.Dir(pkgConfigFilePath))
		if err != nil {
			return CachedPkg{}, errors.Wrapf(
				err, "couldn't check cached pkg defined at %s", pkgConfigFilePath,
			)
		}
		if pkg.Path != pkgPath {
			continue
		}

		if len(candidatePkgs) > 0 {
			return CachedPkg{}, errors.Errorf(
				"package %s repeatedly defined in the same version of the same Github repo: %s, %s",
				pkgPath, candidatePkgs[0].FSPath, pkg.FSPath,
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

func loadCachedPkg(cacheFS fs.FS, pkgConfigPath string) (CachedPkg, error) {
	fsPkg, err := pallets.LoadFSPkg(cacheFS, "", pkgConfigPath)
	if err != nil {
		return CachedPkg{}, errors.Wrapf(
			err, "couldn't load filesystem package from %s", pkgConfigPath,
		)
	}
	repo, err := findRepoOfPkg(cacheFS, pkgConfigPath)
	if err != nil {
		return CachedPkg{}, errors.Wrapf(
			err, "couldn't identify cached repository for package from %s", pkgConfigPath,
		)
	}
	fsPkg.Subdir = strings.TrimPrefix(fsPkg.FSPath, fmt.Sprintf("%s/", repo.ConfigPath))
	fsPkg.Path = fmt.Sprintf("%s/%s", repo.Config.Repository.Path, fsPkg.Subdir)

	return CachedPkg{
		FSPkg: fsPkg,
		Repo:  repo,
	}, nil
}

func findRepoOfPkg(cacheFS fs.FS, pkgConfigPath string) (CachedRepo, error) {
	repoCandidatePath := pkgConfigPath
	for repoCandidatePath != "." {
		repoConfigCandidatePath := filepath.Join(repoCandidatePath, pallets.RepoSpecFile)
		repo, err := LoadCachedRepo(cacheFS, repoConfigCandidatePath)
		if err == nil {
			return repo, nil
		}
		repoCandidatePath = filepath.Dir(repoCandidatePath)
	}
	return CachedRepo{}, errors.Errorf(
		"no repository config file found in any parent directory of %s", pkgConfigPath,
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
		filename := filepath.Base(pkgConfigFilePath)
		if filename != pallets.PkgSpecFile {
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
			if prevPkg.Repo.FromSameVCSRepo(pkg.Repo) && prevPkg.FSPath != pkg.FSPath {
				return nil, errors.Errorf(
					"package repeatedly defined in the same version of the same Github repo: %s, %s",
					prevPkg.FSPath, pkg.FSPath,
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
	repoPathComparison := CompareCachedRepoPaths(p.Repo, q.Repo)
	if repoPathComparison != compareEQ {
		return repoPathComparison
	}
	if p.Subdir != q.Subdir {
		if p.Subdir < q.Subdir {
			return compareLT
		}
		return compareGT
	}
	repoVersionComparison := gosemver.Compare(p.Repo.Version, q.Repo.Version)
	if repoVersionComparison != compareEQ {
		return repoVersionComparison
	}
	return compareEQ
}
