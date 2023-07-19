package forklift

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	gosemver "golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

func (s PkgDeplSpec) DefinesStack() bool {
	return s.DefinitionFile != ""
}

func CompareCachedPkgs(p, q CachedPkg) int {
	repoPathComparison := CompareCachedRepoPaths(p.Repo, q.Repo)
	if repoPathComparison != compareEQ {
		return repoPathComparison
	}
	if p.PkgSubdir != q.PkgSubdir {
		if p.PkgSubdir < q.PkgSubdir {
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

func ListCachedPkgs(cacheFS fs.FS, cachedPrefix string) ([]CachedPkg, error) {
	searchPattern := "**/pallet-package.yml"
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
		if filename != "pallet-package.yml" {
			continue
		}
		pkg, err := loadCachedPkg(cacheFS, pkgConfigFilePath)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load cached package from %s", pkgConfigFilePath)
		}

		versionedPkgPath := fmt.Sprintf(
			"%s@%s/%s", pkg.Repo.Config.Repository.Path, pkg.Repo.Version, pkg.PkgSubdir,
		)
		if prevPkg, ok := pkgMap[versionedPkgPath]; ok {
			if prevPkg.Repo.FromSameVCSRepo(pkg.Repo) && prevPkg.ConfigPath != pkg.ConfigPath {
				return nil, errors.Errorf(
					"package repeatedly defined in the same version of the same Github repo: %s, %s",
					prevPkg.ConfigPath, pkg.ConfigPath,
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

func loadCachedPkg(cacheFS fs.FS, pkgConfigFilePath string) (CachedPkg, error) {
	config, err := loadPkgConfig(cacheFS, pkgConfigFilePath)
	if err != nil {
		return CachedPkg{}, errors.Wrapf(
			err, "couldn't load cached package config from %s", pkgConfigFilePath,
		)
	}

	pkg := CachedPkg{
		ConfigPath: filepath.Dir(pkgConfigFilePath),
		Config:     config,
	}
	pkg.Repo, err = findRepoOfPkg(cacheFS, pkgConfigFilePath)
	if err != nil {
		return CachedPkg{}, errors.Wrapf(
			err, "couldn't identify cached repository for package from %s", pkgConfigFilePath,
		)
	}
	pkg.PkgSubdir = strings.TrimPrefix(pkg.ConfigPath, fmt.Sprintf("%s/", pkg.Repo.ConfigPath))
	pkg.Path = fmt.Sprintf("%s/%s", pkg.Repo.Config.Repository.Path, pkg.PkgSubdir)
	return pkg, nil
}

func loadPkgConfig(cacheFS fs.FS, filePath string) (PkgConfig, error) {
	bytes, err := fs.ReadFile(cacheFS, filePath)
	if err != nil {
		return PkgConfig{}, errors.Wrapf(err, "couldn't read package config file %s", filePath)
	}
	config := PkgConfig{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return PkgConfig{}, errors.Wrap(err, "couldn't parse package config")
	}
	return config, nil
}

func findRepoOfPkg(cacheFS fs.FS, pkgConfigFilePath string) (CachedRepo, error) {
	repoCandidatePath := filepath.Dir(pkgConfigFilePath)
	for repoCandidatePath != "." {
		repoConfigCandidatePath := filepath.Join(repoCandidatePath, "pallet-repository.yml")
		repo, err := LoadCachedRepo(cacheFS, repoConfigCandidatePath)
		if err == nil {
			return repo, nil
		}
		repoCandidatePath = filepath.Dir(repoCandidatePath)
	}
	return CachedRepo{}, errors.Errorf(
		"no repository config file found in any parent directory of %s", pkgConfigFilePath,
	)
}

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
		"%s@%s/**/%s/pallet-package.yml", vcsRepoPath, version, pkgInnermostDir,
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
		if filename != "pallet-package.yml" {
			continue
		}
		pkg, err := loadCachedPkg(cacheFS, pkgConfigFilePath)
		if err != nil {
			return CachedPkg{}, errors.Wrapf(
				err, "couldn't check cached pkg defined at %s", pkgConfigFilePath,
			)
		}
		if pkg.Path == pkgPath {
			if len(candidatePkgs) > 0 {
				return CachedPkg{}, errors.Errorf(
					"package %s repeatedly defined in the same version of the same Github repo: %s, %s",
					pkgPath, candidatePkgs[0].ConfigPath, pkg.ConfigPath,
				)
			}
			candidatePkgs = append(candidatePkgs, pkg)
		}
	}
	if len(candidatePkgs) == 0 {
		return CachedPkg{}, errors.Errorf(
			"no cached repos were found matching %s@%s", pkgPath, version,
		)
	}
	return candidatePkgs[0], nil
}
