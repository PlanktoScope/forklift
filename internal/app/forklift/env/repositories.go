// Package env provides an interface to the local environment
package env

import (
	"bytes"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type RepoConfig struct {
	Release string `yaml:"release"`
}

const Timestamp = "20060102150405"

func ToTimestamp(t time.Time) string {
	return t.UTC().Format(Timestamp)
}

const shortCommitLength = 12

func ShortCommit(commit string) string {
	return commit[:shortCommitLength]
}

type RepoLock struct {
	Version   string `yaml:"version"`
	Timestamp string `yaml:"timestamp"`
	Commit    string `yaml:"commit"`
}

func (l RepoLock) IsCommitLocked() bool {
	return l.Commit != ""
}

func (l RepoLock) IsVersion() bool {
	return l.Version != "" && l.Timestamp != ""
}

func (l RepoLock) ParseVersion() (semver.Version, error) {
	if !strings.HasPrefix(l.Version, "v") {
		return semver.Version{}, errors.Errorf(
			"invalid repo version `%s` doesn't start with `v`", l.Version,
		)
	}
	version, err := semver.Parse(strings.TrimPrefix(l.Version, "v"))
	if err != nil {
		return semver.Version{}, errors.Errorf(
			"repo version `%s` couldn't be parsed as a semantic version", l.Version,
		)
	}
	return version, nil
}

func (l RepoLock) PseudoVersion() (string, error) {
	// This implements the specification described at https://go.dev/ref/mod#pseudo-versions
	if l.Commit == "" {
		return "", errors.Errorf("missing commit hash")
	}
	if l.Timestamp == "" {
		return "", errors.Errorf("missing commit timestamp")
	}
	revisionID := ShortCommit(l.Commit)
	if l.Version == "" {
		return fmt.Sprintf("v0.0.0-%s-%s", l.Timestamp, revisionID), nil
	}
	version, err := l.ParseVersion()
	if err != nil {
		return "", errors.Wrap(err, "couldn't build pseudoversion with base version")
	}
	version.Build = nil
	if len(version.Pre) > 0 {
		return fmt.Sprintf("%s.0.%s-%s", version.String(), l.Timestamp, revisionID), nil
	}
	return fmt.Sprintf(
		"v%d.%d.%d-0.%s-%s", version.Major, version.Minor, version.Patch+1, l.Timestamp, revisionID,
	), nil
}

type Repo struct {
	VCSRepoPath string
	RepoSubdir  string
	Config      RepoConfig
	Lock        RepoLock
}

const sep = "/"

func (r Repo) Path() string {
	return fmt.Sprintf("%s/%s", r.VCSRepoPath, r.RepoSubdir)
}

func (r Repo) VCSRepoVersion() (string, error) {
	if r.Lock.IsVersion() {
		version, err := r.Lock.ParseVersion()
		if err != nil {
			return "", errors.Wrap(err, "invalid lock version")
		}
		return fmt.Sprintf("%s@%s", r.VCSRepoPath, version.String()), nil
	}
	pseudoversion, err := r.Lock.PseudoVersion()
	if err != nil {
		return "", errors.Wrap(err, "couldn't determine pseudo-version")
	}
	return fmt.Sprintf("%s@%s", r.VCSRepoPath, pseudoversion), nil
}

const reposDirName = "repos"

func ReposFS(envFS fs.FS) (fs.FS, error) {
	return fs.Sub(envFS, reposDirName)
}

func getVCSRepoPath(repoPath string) (vcsRepoPath, repoSubdir string, err error) {
	pathParts := strings.Split(repoPath, sep)
	if pathParts[0] != "github.com" {
		return "", "", errors.Errorf(
			"pallet repo %s does not begin with github.com, and handling of non-Github repositories is "+
				"not yet implemented",
			repoPath,
		)
	}
	const splitIndex = 3
	return strings.Join(pathParts[0:splitIndex], sep), strings.Join(pathParts[splitIndex:], sep), nil
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

func loadRepoLock(reposFS fs.FS, filePath string) (RepoLock, error) {
	file, err := reposFS.Open(filePath)
	if err != nil {
		return RepoLock{}, errors.Wrapf(err, "couldn't open file %s", filePath)
	}
	buf, err := loadFile(file)
	if err != nil {
		return RepoLock{}, errors.Wrap(err, "couldn't load repo lock")
	}
	lock := RepoLock{}
	if err = yaml.Unmarshal(buf.Bytes(), &lock); err != nil {
		return RepoLock{}, errors.Wrap(err, "couldn't parse repo lock")
	}
	return lock, nil
}

func ListRepos(envFS fs.FS) ([]Repo, error) {
	reposFS, err := ReposFS(envFS)
	if err != nil {
		return nil, err
	}
	files, err := doublestar.Glob(reposFS, "**/forklift-repo*.yml")
	if err != nil {
		return nil, err
	}
	repoPaths := make([]string, 0, len(files))
	repoMap := make(map[string]Repo)
	for _, filePath := range files {
		repoPath := filepath.Dir(filePath)
		vcsRepoPath, repoSubdir, err := getVCSRepoPath(repoPath)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't determine Github repo path of pallet repo %s", repoPath,
			)
		}
		if _, ok := repoMap[repoPath]; !ok {
			repoPaths = append(repoPaths, repoPath)
			repoMap[repoPath] = Repo{
				VCSRepoPath: vcsRepoPath,
				RepoSubdir:  repoSubdir,
			}
		}
		filename := filepath.Base(filePath)
		switch filename {
		case "forklift-repo.yml":
			config, err := loadRepoConfig(reposFS, filePath)
			if err != nil {
				return nil, errors.Wrapf(err, "couldn't load repo config for %s", repoPath)
			}
			repo := repoMap[repoPath]
			repo.Config = config
			repoMap[repoPath] = repo
		case "forklift-repo-lock.yml":
			lock, err := loadRepoLock(reposFS, filePath)
			if err != nil {
				return nil, errors.Wrapf(err, "couldn't load repo lock for %s", repoPath)
			}
			repo := repoMap[repoPath]
			repo.Lock = lock
			repoMap[repoPath] = repo
		}
	}

	orderedRepos := make([]Repo, 0, len(repoPaths))
	for _, repoPath := range repoPaths {
		orderedRepos = append(orderedRepos, repoMap[repoPath])
	}
	return orderedRepos, nil
}
