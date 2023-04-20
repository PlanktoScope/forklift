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

func (r CachedRepo) FromSameVCSRepo(cr CachedRepo) bool {
	return r.VCSRepoPath == cr.VCSRepoPath && r.Version == cr.Version
}

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
	vcsRepoName, release, ok := strings.Cut(pathParts[2], "@")
	if !ok {
		return "", "", errors.Errorf(
			"Couldn't parse Github repository name %s as name@release", pathParts[2],
		)
	}
	vcsRepoPath = strings.Join([]string{pathParts[0], pathParts[1], vcsRepoName}, sep)
	return vcsRepoPath, release, nil
}

func loadRepoConfig(reposFS fs.FS, filePath string) (RepoConfig, error) {
	file, err := reposFS.Open(filePath)
	if err != nil {
		return RepoConfig{}, errors.Wrapf(err, "couldn't open file %s", filePath)
	}
	buf, err := loadFile(file)
	if err != nil {
		return RepoConfig{}, errors.Wrap(err, "couldn't read repo config")
	}
	config := RepoConfig{}
	if err = yaml.Unmarshal(buf.Bytes(), &config); err != nil {
		return RepoConfig{}, errors.Wrap(err, "couldn't parse repo config")
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
	repo.RepoSubdir = strings.TrimPrefix(config.Path, fmt.Sprintf("%s/", repo.VCSRepoPath))
	return repo, nil
}

func ListCachedRepos(cacheFS fs.FS) ([]CachedRepo, error) {
	repoConfigFiles, err := doublestar.Glob(cacheFS, "**/pallet-repository.yml")
	if err != nil {
		return nil, errors.Wrap(err, "couldn't search for cached package repo configs")
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

		versionedRepoPath := fmt.Sprintf("%s@%s", repo.Config.Path, repo.Version)
		if prevRepo, ok := repoMap[versionedRepoPath]; ok {
			if prevRepo.FromSameVCSRepo(repo) && prevRepo.ConfigPath != repo.ConfigPath {
				return nil, errors.Errorf(
					"repository defined in multiple places: %s, %s", prevRepo.ConfigPath, repo.ConfigPath,
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
