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

func (l RepoVersionConfig) IsCommitLocked() bool {
	return l.Commit != ""
}

func (l RepoVersionConfig) ShortCommit() string {
	return ShortCommit(l.Commit)
}

func (l RepoVersionConfig) IsVersion() bool {
	return l.BaseVersion != "" && l.Commit != ""
}

func (l RepoVersionConfig) ParseVersion() (semver.Version, error) {
	if !strings.HasPrefix(l.BaseVersion, "v") {
		return semver.Version{}, errors.Errorf(
			"invalid repo version `%s` doesn't start with `v`", l.BaseVersion,
		)
	}
	version, err := semver.Parse(strings.TrimPrefix(l.BaseVersion, "v"))
	if err != nil {
		return semver.Version{}, errors.Errorf(
			"repo version `%s` couldn't be parsed as a semantic version", l.BaseVersion,
		)
	}
	return version, nil
}

func (l RepoVersionConfig) Pseudoversion() (string, error) {
	// This implements the specification described at https://go.dev/ref/mod#pseudo-versions
	if l.Commit == "" {
		return "", errors.Errorf("repo version missing commit hash")
	}
	if l.Timestamp == "" {
		return "", errors.Errorf("repo version missing commit timestamp")
	}
	revisionID := ShortCommit(l.Commit)
	if l.BaseVersion == "" {
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

func (l RepoVersionConfig) Version() (string, error) {
	if l.IsVersion() {
		version, err := l.ParseVersion()
		if err != nil {
			return "", errors.Wrap(err, "invalid repo version")
		}
		return version.String(), nil
	}
	pseudoversion, err := l.Pseudoversion()
	if err != nil {
		return "", errors.Wrap(err, "couldn't determine pseudo-version")
	}
	return pseudoversion, nil
}

func (r VersionedRepo) Path() string {
	return fmt.Sprintf("%s/%s", r.VCSRepoPath, r.RepoSubdir)
}

func CompareVersionedRepos(r, s VersionedRepo) int {
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

const versionedReposDirName = "repositories"

func VersionedReposFS(envFS fs.FS) (fs.FS, error) {
	return fs.Sub(envFS, versionedReposDirName)
}

// SplitRepoPathSubdir splits paths of form github.com/user-name/git-repo-name/pallets-repo-subdir
// into github.com/user-name/git-repo-name and pallets-repo-subdir.
func SplitRepoPathSubdir(repoPath string) (vcsRepoPath, repoSubdir string, err error) {
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
	if len(pathParts) < splitIndex {
		return "", "", errors.Errorf(
			"pallet repository %s does not appear to be within a Github Git repository", repoPath,
		)
	}
	return strings.Join(pathParts[0:splitIndex], sep), strings.Join(pathParts[splitIndex:], sep), nil
}

func loadRepoVersionConfig(reposFS fs.FS, filePath string) (RepoVersionConfig, error) {
	bytes, err := fs.ReadFile(reposFS, filePath)
	if err != nil {
		return RepoVersionConfig{}, errors.Wrapf(
			err, "couldn't read repo version config file %s", filePath,
		)
	}
	config := RepoVersionConfig{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return RepoVersionConfig{}, errors.Wrap(err, "couldn't parse repo version config")
	}
	return config, nil
}

func LoadVersionedRepo(reposFS fs.FS, repoPath string) (VersionedRepo, error) {
	vcsRepoPath, repoSubdir, err := SplitRepoPathSubdir(repoPath)
	if err != nil {
		return VersionedRepo{}, errors.Wrapf(
			err, "couldn't parse path of version-locked Pallet repo %s", repoPath,
		)
	}
	repo := VersionedRepo{
		VCSRepoPath: vcsRepoPath,
		RepoSubdir:  repoSubdir,
	}

	repo.Config, err = loadRepoVersionConfig(
		reposFS, filepath.Join(repoPath, "forklift-repo.yml"),
	)
	if err != nil {
		return VersionedRepo{}, errors.Wrapf(err, "couldn't load repo version config for %s", repoPath)
	}

	return repo, nil
}

func ListVersionedRepos(envFS fs.FS) ([]VersionedRepo, error) {
	reposFS, err := VersionedReposFS(envFS)
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't open directory for Pallet repositories in local environment",
		)
	}
	files, err := doublestar.Glob(reposFS, "**/forklift-repo.yml")
	if err != nil {
		return nil, errors.Wrap(err, "couldn't search for Pallet repo versioning configs")
	}

	repoPaths := make([]string, 0, len(files))
	repoMap := make(map[string]VersionedRepo)
	for _, filePath := range files {
		repoPath := filepath.Dir(filePath)
		if _, ok := repoMap[repoPath]; ok {
			return nil, errors.Errorf(
				"versioned repository %s repeatedly defined in the local environment", repoPath,
			)
		}
		repoPaths = append(repoPaths, repoPath)
		repoMap[repoPath], err = LoadVersionedRepo(reposFS, repoPath)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load versioned repo from %s", repoPath)
		}
	}

	orderedRepos := make([]VersionedRepo, 0, len(repoPaths))
	for _, repoPath := range repoPaths {
		orderedRepos = append(orderedRepos, repoMap[repoPath])
	}
	return orderedRepos, nil
}
