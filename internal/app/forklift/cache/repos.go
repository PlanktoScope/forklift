// Package cache manages the downloaded cache of Pallet repositories and packages
package cache

import (
	"bytes"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type RepoConfig struct {
	Path string `yaml:"path"`
}

type Repo struct {
	VCSRepoPath string
	Version     string
	RepoSubdir  string
	Config      RepoConfig
}

const sep = "/"

func getVCSRepoPathVersion(repoPath string) (vcsRepoPath, release string, err error) {
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

func loadFile(file fs.File) (bytes.Buffer, error) {
	buf := bytes.Buffer{}
	_, err := buf.ReadFrom(file)
	return buf, errors.Wrap(err, "couldn't load file")
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

func ListRepos(cacheFS fs.FS) ([]Repo, error) {
	files, err := doublestar.Glob(cacheFS, "**/pallet-repository.yml")
	if err != nil {
		return nil, err
	}
	repoPaths := make([]string, 0, len(files))
	repoMap := make(map[string]Repo)
	for _, filePath := range files {
		repoPath := filepath.Dir(filePath)
		vcsRepoPath, version, err := getVCSRepoPathVersion(repoPath)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't determine Github repo path of pallet repo %s", repoPath,
			)
		}
		repoSubdir := strings.TrimPrefix(filePath, fmt.Sprintf("%s/", vcsRepoPath))
		if _, ok := repoMap[repoPath]; !ok {
			repoPaths = append(repoPaths, repoPath)
			repoMap[repoPath] = Repo{
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
			return nil, errors.Wrapf(err, "couldn't load repo config for %s", repoPath)
		}
		repo := repoMap[repoPath]
		repo.Config = config
		repoMap[repoPath] = repo
	}

	orderedRepos := make([]Repo, 0, len(repoPaths))
	for _, repoPath := range repoPaths {
		orderedRepos = append(orderedRepos, repoMap[repoPath])
	}
	return orderedRepos, nil
}
