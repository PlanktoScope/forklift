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

func ListCachedRepos(cacheFS fs.FS) ([]CachedRepo, error) {
	files, err := doublestar.Glob(cacheFS, "**/pallet-repository.yml")
	if err != nil {
		return nil, err
	}
	repoPaths := make([]string, 0, len(files))
	repoMap := make(map[string]CachedRepo)
	for _, filePath := range files {
		repoPath := filepath.Dir(filePath)
		vcsRepoPath, version, err := splitRepoPathVersion(repoPath)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't parse path of cached Pallet repo %s", repoPath)
		}
		repoSubdir := strings.TrimPrefix(filePath, fmt.Sprintf("%s/", vcsRepoPath))
		if _, ok := repoMap[repoPath]; !ok {
			repoPaths = append(repoPaths, repoPath)
			repoMap[repoPath] = CachedRepo{
				VCSRepoPath: vcsRepoPath,
				Version:     version,
				RepoSubdir:  repoSubdir,
			}
		}
		filename := filepath.Base(filePath)
		if filename != "pallet-repository.yml" {
			continue
		}
		config, err := loadRepoConfig(cacheFS, filePath)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load cached repo config for %s", repoPath)
		}
		repo := repoMap[repoPath]
		repo.Config = config
		repoMap[repoPath] = repo
	}

	orderedRepos := make([]CachedRepo, 0, len(repoPaths))
	for _, repoPath := range repoPaths {
		orderedRepos = append(orderedRepos, repoMap[repoPath])
	}
	return orderedRepos, nil
}
