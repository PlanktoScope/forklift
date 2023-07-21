package forklift

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Loading

func FindCachedRepo(cacheFS fs.FS, repoPath string, version string) (*pallets.FSRepo, error) {
	vcsRepoPath, _, err := SplitRepoPathSubdir(repoPath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't parse path of Pallet repo %s", repoPath)
	}
	// The repo subdirectory path in the repo path (under the VCS repo path) might not match the
	// filesystem directory path with the pallet-repository.yml file, so we must check every
	// pallet-repository.yml file to find the actual repo path
	searchPattern := fmt.Sprintf("%s@%s/**/%s", vcsRepoPath, version, pallets.RepoSpecFile)
	candidateRepoConfigFiles, err := doublestar.Glob(cacheFS, searchPattern)
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
		repo, err := loadCachedRepo(cacheFS, filepath.Dir(repoConfigFilePath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't check cached repo defined at %s", repoConfigFilePath)
		}
		if repo.Config.Repository.Path == repoPath {
			if len(candidateRepos) > 0 {
				return nil, errors.Errorf(
					"repository %s repeatedly defined in the same version of the same Github repo: %s, %s",
					repoPath, candidateRepos[0].FS.Path(), repo.FS.Path(),
				)
			}
			candidateRepos = append(candidateRepos, repo)
		}
	}
	if len(candidateRepos) == 0 {
		return nil, errors.Errorf(
			"no cached repos were found matching %s@%s", repoPath, version,
		)
	}
	return candidateRepos[0], nil
}

func loadCachedRepo(cacheFS fs.FS, repoConfigPath string) (*pallets.FSRepo, error) {
	repo, err := pallets.LoadFSRepo(pallets.AttachPath(cacheFS, ""), repoConfigPath)
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

// splitRepoPathVersion splits paths of form github.com/user-name/git-repo-name@version into
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

// Listing

func ListCachedRepos(cacheFS fs.FS) ([]*pallets.FSRepo, error) {
	repoConfigFiles, err := doublestar.Glob(cacheFS, fmt.Sprintf("**/%s", pallets.RepoSpecFile))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't search for cached repo configs")
	}

	versionedRepoPaths := make([]string, 0, len(repoConfigFiles))
	repoMap := make(map[string]*pallets.FSRepo)
	for _, repoConfigFilePath := range repoConfigFiles {
		if filepath.Base(repoConfigFilePath) != pallets.RepoSpecFile {
			continue
		}
		repo, err := loadCachedRepo(cacheFS, filepath.Dir(repoConfigFilePath))
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
