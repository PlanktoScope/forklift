package forklift

import (
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

// Pseudo-versions

const Timestamp = "20060102150405"

func ToTimestamp(t time.Time) string {
	return t.UTC().Format(Timestamp)
}

const shortCommitLength = 12

func ShortCommit(commit string) string {
	return commit[:shortCommitLength]
}

// Repository versioning

func (l RepoVersionLock) IsCommitLocked() bool {
	return l.Commit != ""
}

func (l RepoVersionLock) ShortCommit() string {
	return ShortCommit(l.Commit)
}

func (l RepoVersionLock) IsVersion() bool {
	return l.Version != "" && l.Timestamp != ""
}

func (l RepoVersionLock) ParseVersion() (semver.Version, error) {
	if !strings.HasPrefix(l.Version, "v") {
		return semver.Version{}, errors.Errorf(
			"invalid repo lock version `%s` doesn't start with `v`", l.Version,
		)
	}
	version, err := semver.Parse(strings.TrimPrefix(l.Version, "v"))
	if err != nil {
		return semver.Version{}, errors.Errorf(
			"repo lock version `%s` couldn't be parsed as a semantic version", l.Version,
		)
	}
	return version, nil
}

func (l RepoVersionLock) PseudoVersion() (string, error) {
	// This implements the specification described at https://go.dev/ref/mod#pseudo-versions
	if l.Commit == "" {
		return "", errors.Errorf("repo lock missing commit hash")
	}
	if l.Timestamp == "" {
		return "", errors.Errorf("repo lock missing commit timestamp")
	}
	revisionID := ShortCommit(l.Commit)
	if l.Version == "" {
		return fmt.Sprintf("v0.0.0-%s-%s", l.Timestamp, revisionID), nil
	}
	version, err := l.ParseVersion()
	if err != nil {
		return "", err
	}
	version.Build = nil
	if len(version.Pre) > 0 {
		return fmt.Sprintf("%s.0.%s-%s", version.String(), l.Timestamp, revisionID), nil
	}
	return fmt.Sprintf(
		"v%d.%d.%d-0.%s-%s", version.Major, version.Minor, version.Patch+1, l.Timestamp, revisionID,
	), nil
}

func (r VersionedRepo) Path() string {
	return fmt.Sprintf("%s/%s", r.VCSRepoPath, r.RepoSubdir)
}

func (r VersionedRepo) VCSRepoVersion() (string, error) {
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

const versionedReposDirName = "repos"

func VersionedReposFS(envFS fs.FS) (fs.FS, error) {
	return fs.Sub(envFS, versionedReposDirName)
}

func splitRepoPathSubdir(repoPath string) (vcsRepoPath, repoSubdir string, err error) {
	const sep = "/"
	pathParts := strings.Split(repoPath, sep)
	if pathParts[0] != "github.com" {
		return "", "", errors.Errorf(
			"pallet repository %s does not begin with github.com, but support for non-Github "+
				"repositories is not yet implemented",
			repoPath,
		)
	}
	const splitIndex = 3
	return strings.Join(pathParts[0:splitIndex], sep), strings.Join(pathParts[splitIndex:], sep), nil
}

func loadRepoVersionConfig(reposFS fs.FS, filePath string) (RepoVersionConfig, error) {
	file, err := reposFS.Open(filePath)
	if err != nil {
		return RepoVersionConfig{}, errors.Wrapf(err, "couldn't open file %s", filePath)
	}
	buf, err := loadFile(file)
	if err != nil {
		return RepoVersionConfig{}, errors.Wrap(err, "couldn't read repo version config file")
	}
	config := RepoVersionConfig{}
	if err = yaml.Unmarshal(buf.Bytes(), &config); err != nil {
		return RepoVersionConfig{}, errors.Wrap(err, "couldn't parse repo version config")
	}
	return config, nil
}

func loadRepoVersionLock(reposFS fs.FS, filePath string) (RepoVersionLock, error) {
	file, err := reposFS.Open(filePath)
	if err != nil {
		return RepoVersionLock{}, errors.Wrapf(err, "couldn't open file %s", filePath)
	}
	buf, err := loadFile(file)
	if err != nil {
		return RepoVersionLock{}, errors.Wrap(err, "couldn't read repo version lock file")
	}
	lock := RepoVersionLock{}
	if err = yaml.Unmarshal(buf.Bytes(), &lock); err != nil {
		return RepoVersionLock{}, errors.Wrap(err, "couldn't parse repo version lock")
	}
	return lock, nil
}

func ListVersionedRepos(envFS fs.FS) ([]VersionedRepo, error) {
	reposFS, err := VersionedReposFS(envFS)
	if err != nil {
		return nil, err
	}
	files, err := doublestar.Glob(reposFS, "**/forklift-repo*.yml")
	if err != nil {
		return nil, err
	}
	repoPaths := make([]string, 0, len(files))
	repoMap := make(map[string]VersionedRepo)
	for _, filePath := range files {
		repoPath := filepath.Dir(filePath)
		vcsRepoPath, repoSubdir, err := splitRepoPathSubdir(repoPath)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't parse path of version-locked Pallet repo %s", repoPath,
			)
		}
		if _, ok := repoMap[repoPath]; !ok {
			repoPaths = append(repoPaths, repoPath)
			repoMap[repoPath] = VersionedRepo{
				VCSRepoPath: vcsRepoPath,
				RepoSubdir:  repoSubdir,
			}
		}
		filename := filepath.Base(filePath)
		switch filename {
		case "forklift-repo.yml":
			config, err := loadRepoVersionConfig(reposFS, filePath)
			if err != nil {
				return nil, errors.Wrapf(err, "couldn't load repo version config for %s", repoPath)
			}
			repo := repoMap[repoPath]
			repo.Config = config
			repoMap[repoPath] = repo
		case "forklift-repo-lock.yml":
			lock, err := loadRepoVersionLock(reposFS, filePath)
			if err != nil {
				return nil, errors.Wrapf(err, "couldn't load repo version lock for %s", repoPath)
			}
			repo := repoMap[repoPath]
			repo.Lock = lock
			repoMap[repoPath] = repo
		}
	}

	orderedRepos := make([]VersionedRepo, 0, len(repoPaths))
	for _, repoPath := range repoPaths {
		orderedRepos = append(orderedRepos, repoMap[repoPath])
	}
	return orderedRepos, nil
}
