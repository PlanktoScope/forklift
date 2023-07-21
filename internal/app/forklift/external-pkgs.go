package forklift

import (
	"fmt"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Loading

func FindExternalPkg(repo pallets.FSRepo, pkgPath string) (CachedPkg, error) {
	pkgInnermostDir := filepath.Base(pkgPath)
	// The package subdirectory path in the package path (under the VCS repo path) might not match the
	// filesystem directory path with the pallet-package.yml file, so we must check every
	// directory whose name matches the last part of the package path to look for the package
	searchPattern := fmt.Sprintf("**/%s/%s", pkgInnermostDir, pallets.PkgSpecFile)
	candidatePkgConfigFiles, err := doublestar.Glob(repo.FS, searchPattern)
	if err != nil {
		return CachedPkg{}, errors.Wrapf(
			err, "couldn't search for external Pallet package configs matching pattern %s", searchPattern,
		)
	}
	if len(candidatePkgConfigFiles) == 0 {
		return CachedPkg{}, errors.New("no matching Pallet package configs were found")
	}
	candidatePkgs := make([]CachedPkg, 0)
	for _, pkgConfigFilePath := range candidatePkgConfigFiles {
		if filepath.Base(pkgConfigFilePath) != pallets.PkgSpecFile {
			continue
		}

		pkg, err := loadPkgFromRepo(repo, filepath.Dir(pkgConfigFilePath))
		if err != nil {
			return CachedPkg{}, errors.Wrapf(
				err, "couldn't check external pkg defined at %s", pkgConfigFilePath,
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
		return CachedPkg{}, errors.Errorf("no external repos were found matching %s", pkgPath)
	}
	return candidatePkgs[0], nil
}

func loadPkgFromRepo(repo pallets.FSRepo, subdirPath string) (CachedPkg, error) {
	fsPkg, err := repo.LoadPkg(subdirPath)
	if err != nil {
		return CachedPkg{}, err
	}

	return CachedPkg{
		FSPkg: fsPkg,
		Repo:  repo,
	}, nil
}

func AsVersionedPkg(pkg CachedPkg) VersionedPkg {
	return VersionedPkg{
		Path: pkg.Path(),
		Repo: VersionedRepo{
			VCSRepoPath: pkg.Repo.VCSRepoPath,
			RepoSubdir:  pkg.Repo.Subdir,
		},
		Cached: pkg,
	}
}

// Listing

func ListExternalPkgs(repo pallets.FSRepo, cachedPrefix string) ([]CachedPkg, error) {
	searchPattern := fmt.Sprintf("**/%s", pallets.PkgSpecFile)
	if cachedPrefix != "" {
		searchPattern = filepath.Join(cachedPrefix, searchPattern)
	}
	pkgConfigFiles, err := doublestar.Glob(repo.FS, searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't search for external package configs")
	}

	repoPkgPaths := make([]string, 0, len(pkgConfigFiles))
	pkgMap := make(map[string]CachedPkg)
	for _, pkgConfigFilePath := range pkgConfigFiles {
		if filepath.Base(pkgConfigFilePath) != pallets.PkgSpecFile {
			continue
		}
		pkg, err := loadPkgFromRepo(repo, filepath.Dir(pkgConfigFilePath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load cached package from %s", pkgConfigFilePath)
		}

		if prevPkg, ok := pkgMap[pkg.Path()]; ok {
			if prevPkg.Repo.FromSameVCSRepo(pkg.Repo.Repo) && prevPkg.FS.Path() != pkg.FS.Path() {
				return nil, errors.Errorf(
					"package repeatedly defined in the same version of the same Github repo: %s, %s",
					prevPkg.FS.Path(), pkg.FS.Path(),
				)
			}
		}
		repoPkgPaths = append(repoPkgPaths, pkg.Path())
		pkgMap[pkg.Path()] = pkg
	}

	orderedPkgs := make([]CachedPkg, 0, len(repoPkgPaths))
	for _, path := range repoPkgPaths {
		orderedPkgs = append(orderedPkgs, pkgMap[path])
	}
	return orderedPkgs, nil
}
