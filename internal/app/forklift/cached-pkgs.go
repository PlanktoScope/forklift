package forklift

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

func loadPkgConfig(cacheFS fs.FS, filePath string) (PkgConfig, error) {
	file, err := cacheFS.Open(filePath)
	if err != nil {
		return PkgConfig{}, errors.Wrapf(err, "couldn't open file %s", filePath)
	}
	buf, err := loadFile(file)
	if err != nil {
		return PkgConfig{}, errors.Wrap(err, "couldn't read package config")
	}
	config := PkgConfig{}
	if err = yaml.Unmarshal(buf.Bytes(), &config); err != nil {
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

func LoadCachedPkg(cacheFS fs.FS, pkgConfigFilePath string) (CachedPkg, error) {
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
	pkg.Path = fmt.Sprintf("%s/%s", pkg.Repo.Config.Path, pkg.PkgSubdir)
	return pkg, nil
}

func ListCachedPkgs(cacheFS fs.FS) ([]CachedPkg, error) {
	pkgConfigFiles, err := doublestar.Glob(cacheFS, "**/pallet-package.yml")
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
		pkg, err := LoadCachedPkg(cacheFS, pkgConfigFilePath)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load cached package from %s", pkgConfigFilePath)
		}

		versionedPkgPath := fmt.Sprintf(
			"%s@%s/%s", pkg.Repo.Config.Path, pkg.Repo.Version, pkg.PkgSubdir,
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
