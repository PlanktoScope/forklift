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

func (r CachedRepo) FromSameVCSRepo(cr CachedRepo) bool {
	return r.VCSRepoPath == cr.VCSRepoPath && r.Version == cr.Version
}

func (r CachedRepo) Path() string {
	return filepath.Join(r.VCSRepoPath, r.RepoSubdir)
}

const (
	compareLT = -1
	compareEQ = 0
	compareGT = 1
)

func CompareCachedRepoPaths(r, s CachedRepo) int {
	if r.VCSRepoPath != s.VCSRepoPath {
		if r.VCSRepoPath < s.VCSRepoPath {
			return compareLT
		}
		return compareGT
	}
	if r.RepoSubdir != s.RepoSubdir {
		if r.RepoSubdir < s.RepoSubdir {
			return compareLT
		}
		return compareGT
	}
	return compareEQ
}

func CompareCachedRepos(r, s CachedRepo) int {
	pathComparison := CompareCachedRepoPaths(r, s)
	if pathComparison != compareEQ {
		return pathComparison
	}
	versionComparison := gosemver.Compare(r.Version, s.Version)
	if versionComparison != compareEQ {
		return versionComparison
	}
	return compareEQ
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

func loadRepoConfig(cacheFS fs.FS, filePath string) (RepoConfig, error) {
	bytes, err := fs.ReadFile(cacheFS, filePath)
	if err != nil {
		return RepoConfig{}, errors.Wrapf(err, "couldn't read repo config file %s", filePath)
	}
	config := RepoConfig{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return RepoConfig{}, errors.Wrap(err, "couldn't parse repo config")
	}
	if config.Repository.Path == "" {
		return RepoConfig{}, errors.Errorf("repo config at %s is missing `path` parameter", filePath)
	}
	return config, nil
}

func LoadCachedRepo(cacheFS fs.FS, repoConfigFilePath string) (CachedRepo, error) {
	config, err := loadRepoConfig(cacheFS, repoConfigFilePath)
	if err != nil {
		return CachedRepo{}, errors.Wrapf(
			err, "couldn't load cached repo config from %s", repoConfigFilePath,
		)
	}

	repo := CachedRepo{
		ConfigPath: filepath.Dir(repoConfigFilePath),
		Config:     config,
	}
	if repo.VCSRepoPath, repo.Version, err = splitRepoPathVersion(repo.ConfigPath); err != nil {
		return CachedRepo{}, errors.Wrapf(
			err, "couldn't parse path of cached repo configured at %s", repo.ConfigPath,
		)
	}
	repo.RepoSubdir = strings.TrimPrefix(config.Repository.Path, fmt.Sprintf("%s/", repo.VCSRepoPath))
	return repo, nil
}

func ListCachedRepos(cacheFS fs.FS) ([]CachedRepo, error) {
	repoConfigFiles, err := doublestar.Glob(cacheFS, "**/pallet-repository.yml")
	if err != nil {
		return nil, errors.Wrap(err, "couldn't search for cached repo configs")
	}

	versionedRepoPaths := make([]string, 0, len(repoConfigFiles))
	repoMap := make(map[string]CachedRepo)
	for _, repoConfigFilePath := range repoConfigFiles {
		filename := filepath.Base(repoConfigFilePath)
		if filename != "pallet-repository.yml" {
			continue
		}
		repo, err := LoadCachedRepo(cacheFS, repoConfigFilePath)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load cached repo from %s", repoConfigFilePath)
		}

		versionedRepoPath := fmt.Sprintf("%s@%s", repo.Config.Repository.Path, repo.Version)
		if prevRepo, ok := repoMap[versionedRepoPath]; ok {
			if prevRepo.FromSameVCSRepo(repo) && prevRepo.ConfigPath != repo.ConfigPath {
				return nil, errors.Errorf(
					"repository repeatedly defined in the same version of the same Github repo: %s, %s",
					prevRepo.ConfigPath, repo.ConfigPath,
				)
			}
		}
		versionedRepoPaths = append(versionedRepoPaths, versionedRepoPath)
		repoMap[versionedRepoPath] = repo
	}

	orderedRepos := make([]CachedRepo, 0, len(versionedRepoPaths))
	for _, path := range versionedRepoPaths {
		orderedRepos = append(orderedRepos, repoMap[path])
	}
	return orderedRepos, nil
}

func FindCachedRepo(cacheFS fs.FS, repoPath string, version string) (CachedRepo, error) {
	vcsRepoPath, _, err := SplitRepoPathSubdir(repoPath)
	if err != nil {
		return CachedRepo{}, errors.Wrapf(err, "couldn't parse path of Pallet repo %s", repoPath)
	}
	// The repo subdirectory path in the repo path (under the VCS repo path) might not match the
	// filesystem directory path with the pallet-repository.yml file, so we must check every
	// pallet-repository.yml file to find the actual repo path
	searchPattern := fmt.Sprintf("%s@%s/**/pallet-repository.yml", vcsRepoPath, version)
	candidateRepoConfigFiles, err := doublestar.Glob(cacheFS, searchPattern)
	if err != nil {
		return CachedRepo{}, errors.Wrapf(
			err, "couldn't search for cached Pallet repo configs matching pattern %s", searchPattern,
		)
	}
	if len(candidateRepoConfigFiles) == 0 {
		return CachedRepo{}, errors.Errorf(
			"no Pallet repo configs were found in %s@%s", vcsRepoPath, version,
		)
	}
	candidateRepos := make([]CachedRepo, 0)
	for _, repoConfigFilePath := range candidateRepoConfigFiles {
		filename := filepath.Base(repoConfigFilePath)
		if filename != "pallet-repository.yml" {
			continue
		}
		repo, err := LoadCachedRepo(cacheFS, repoConfigFilePath)
		if err != nil {
			return CachedRepo{}, errors.Wrapf(
				err, "couldn't check cached repo defined at %s", repoConfigFilePath,
			)
		}
		if repo.Config.Repository.Path == repoPath {
			if len(candidateRepos) > 0 {
				return CachedRepo{}, errors.Errorf(
					"repository %s repeatedly defined in the same version of the same Github repo: %s, %s",
					repoPath, candidateRepos[0].ConfigPath, repo.ConfigPath,
				)
			}
			candidateRepos = append(candidateRepos, repo)
		}
	}
	if len(candidateRepos) == 0 {
		return CachedRepo{}, errors.Errorf(
			"no cached repos were found matching %s@%s", repoPath, version,
		)
	}
	return candidateRepos[0], nil
}
