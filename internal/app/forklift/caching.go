package forklift

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// FSCache

func (c *FSCache) Exists() bool {
	return Exists(c.FS.Path())
}

func (c *FSCache) Remove() error {
	return os.RemoveAll(c.FS.Path())
}

// FSCache: Loading Repos

func (c *FSCache) FindRepo(repoPath string, version string) (*pallets.FSRepo, error) {
	vcsRepoPath, _, err := SplitRepoPathSubdir(repoPath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't parse path of Pallet repo %s", repoPath)
	}
	// The repo subdirectory path in the repo path (under the VCS repo path) might not match the
	// filesystem directory path with the pallet-repository.yml file, so we must check every
	// pallet-repository.yml file to find the actual repo path
	searchPattern := fmt.Sprintf("%s@%s/**/%s", vcsRepoPath, version, pallets.RepoSpecFile)
	candidateRepoConfigFiles, err := doublestar.Glob(c.FS, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for cached Pallet repo configs matching pattern %s", searchPattern,
		)
	}
	if len(candidateRepoConfigFiles) == 0 {
		return nil, errors.Errorf(
			"no Pallet repo configs were found in %s@%s", vcsRepoPath, version,
		)
	}
	candidateRepos := make([]*pallets.FSRepo, 0)
	for _, repoConfigFilePath := range candidateRepoConfigFiles {
		if filepath.Base(repoConfigFilePath) != pallets.RepoSpecFile {
			continue
		}

		repo, err := c.loadRepo(filepath.Dir(repoConfigFilePath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't check cached repo defined at %s", repoConfigFilePath)
		}
		if repo.Config.Repository.Path != repoPath {
			continue
		}

		if len(candidateRepos) > 0 {
			return nil, errors.Errorf(
				"repository %s repeatedly defined in the same version of the same Github repo: %s, %s",
				repoPath, candidateRepos[0].FS.Path(), repo.FS.Path(),
			)
		}
		candidateRepos = append(candidateRepos, repo)
	}
	if len(candidateRepos) == 0 {
		return nil, errors.Errorf(
			"no cached repos were found matching %s@%s", repoPath, version,
		)
	}
	return candidateRepos[0], nil
}

func (c *FSCache) loadRepo(repoConfigPath string) (*pallets.FSRepo, error) {
	repo, err := pallets.LoadFSRepo(pallets.AttachPath(c.FS, ""), repoConfigPath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't load repo config from %s", repoConfigPath)
	}
	if repo.VCSRepoPath, repo.Version, err = splitRepoPathVersion(repo.FS.Path()); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't parse path of cached repo configured at %s", repo.FS.Path(),
		)
	}
	repo.Subdir = strings.TrimPrefix(
		repo.Config.Repository.Path, fmt.Sprintf("%s/", repo.VCSRepoPath),
	)
	return repo, nil
}

// splitRepoPathVersion splits paths of form github.com/user-name/git-repo-name/etc@version into
// github.com/user-name/git-repo-name and version.
func splitRepoPathVersion(repoPath string) (vcsRepoPath, version string, err error) {
	const sep = "/"
	pathParts := strings.Split(repoPath, sep)
	if pathParts[0] != "github.com" {
		return "", "", errors.Errorf(
			"pallet repo %s does not begin with github.com, and handling of non-Github repositories is "+
				"not yet implemented",
			repoPath,
		)
	}
	vcsRepoName, version, ok := strings.Cut(pathParts[2], "@")
	if !ok {
		return "", "", errors.Errorf(
			"Couldn't parse Github repository name %s as name@version", pathParts[2],
		)
	}
	vcsRepoPath = strings.Join([]string{pathParts[0], pathParts[1], vcsRepoName}, sep)
	return vcsRepoPath, version, nil
}

// FSCache: Listing Repos

func (c *FSCache) ListRepos() ([]*pallets.FSRepo, error) {
	repoConfigFiles, err := doublestar.Glob(c.FS, fmt.Sprintf("**/%s", pallets.RepoSpecFile))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't search for cached repo configs")
	}

	versionedRepoPaths := make([]string, 0, len(repoConfigFiles))
	repoMap := make(map[string]*pallets.FSRepo)
	for _, repoConfigFilePath := range repoConfigFiles {
		if filepath.Base(repoConfigFilePath) != pallets.RepoSpecFile {
			continue
		}
		repo, err := c.loadRepo(filepath.Dir(repoConfigFilePath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load cached repo from %s", repoConfigFilePath)
		}

		versionedRepoPath := fmt.Sprintf("%s@%s", repo.Config.Repository.Path, repo.Version)
		if prevRepo, ok := repoMap[versionedRepoPath]; ok {
			if prevRepo.FromSameVCSRepo(repo.Repo) && prevRepo.FS.Path() != repo.FS.Path() {
				return nil, errors.Errorf(
					"repository repeatedly defined in the same version of the same Github repo: %s, %s",
					prevRepo.FS.Path(), repo.FS.Path(),
				)
			}
		}
		versionedRepoPaths = append(versionedRepoPaths, versionedRepoPath)
		repoMap[versionedRepoPath] = repo
	}

	orderedRepos := make([]*pallets.FSRepo, 0, len(versionedRepoPaths))
	for _, path := range versionedRepoPaths {
		orderedRepos = append(orderedRepos, repoMap[path])
	}
	return orderedRepos, nil
}

// FSCache: Loading Pkgs

func (c *FSCache) FindPkg(pkgPath string, version string) (*pallets.FSPkg, error) {
	vcsRepoPath, _, err := SplitRepoPathSubdir(pkgPath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't parse path of Pallet package %s", pkgPath)
	}
	pkgInnermostDir := filepath.Base(pkgPath)
	// The package subdirectory path in the package path (under the VCS repo path) might not match the
	// filesystem directory path with the pallet-package.yml file, so we must check every
	// directory whose name matches the last part of the package path to look for the package
	searchPattern := fmt.Sprintf(
		"%s@%s/**/%s/%s", vcsRepoPath, version, pkgInnermostDir, pallets.PkgSpecFile,
	)
	candidatePkgConfigFiles, err := doublestar.Glob(c.FS, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for cached Pallet package configs matching pattern %s", searchPattern,
		)
	}
	if len(candidatePkgConfigFiles) == 0 {
		return nil, errors.Errorf(
			"no matching Pallet package configs were found in %s@%s", vcsRepoPath, version,
		)
	}
	candidatePkgs := make([]*pallets.FSPkg, 0)
	for _, pkgConfigFilePath := range candidatePkgConfigFiles {
		if filepath.Base(pkgConfigFilePath) != pallets.PkgSpecFile {
			continue
		}

		pkg, err := c.loadPkg(filepath.Dir(pkgConfigFilePath))
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't check cached pkg defined at %s", pkgConfigFilePath,
			)
		}
		if pkg.Path() != pkgPath {
			continue
		}

		if len(candidatePkgs) > 0 {
			return nil, errors.Errorf(
				"package %s repeatedly defined in the same version of the same Github repo: %s, %s",
				pkgPath, candidatePkgs[0].FS.Path(), pkg.FS.Path(),
			)
		}
		candidatePkgs = append(candidatePkgs, pkg)
	}
	if len(candidatePkgs) == 0 {
		return nil, errors.Errorf(
			"no cached packages were found matching %s@%s", pkgPath, version,
		)
	}
	return candidatePkgs[0], nil
}

func (c *FSCache) loadPkg(subdirPath string) (*pallets.FSPkg, error) {
	repo, err := c.findRepoContaining(subdirPath)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't identify cached repository for package from %s", subdirPath,
		)
	}
	return repo.LoadPkg(strings.TrimPrefix(subdirPath, fmt.Sprintf("%s/", repo.FS.Path())))
}

func (c *FSCache) findRepoContaining(subdirPath string) (*pallets.FSRepo, error) {
	repoCandidatePath := subdirPath
	for repoCandidatePath != "." {
		repo, err := c.loadRepo(repoCandidatePath)
		if err == nil {
			return repo, nil
		}
		repoCandidatePath = filepath.Dir(repoCandidatePath)
	}
	return nil, errors.Errorf(
		"no repository config file found in any parent directory of %s", subdirPath,
	)
}

// FSCache: Listing Pkgs

func (c *FSCache) ListPkgs(cachedPrefix string) ([]*pallets.FSPkg, error) {
	searchPattern := fmt.Sprintf("**/%s", pallets.PkgSpecFile)
	if cachedPrefix != "" {
		searchPattern = filepath.Join(cachedPrefix, searchPattern)
	}
	pkgConfigFiles, err := doublestar.Glob(c.FS, searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't search for cached package configs")
	}

	repoVersionedPkgPaths := make([]string, 0, len(pkgConfigFiles))
	pkgMap := make(map[string]*pallets.FSPkg)
	for _, pkgConfigFilePath := range pkgConfigFiles {
		if filepath.Base(pkgConfigFilePath) != pallets.PkgSpecFile {
			continue
		}
		pkg, err := c.loadPkg(filepath.Dir(pkgConfigFilePath))
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

	orderedPkgs := make([]*pallets.FSPkg, 0, len(repoVersionedPkgPaths))
	for _, path := range repoVersionedPkgPaths {
		orderedPkgs = append(orderedPkgs, pkgMap[path])
	}
	return orderedPkgs, nil
}
