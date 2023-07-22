package forklift

import (
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/pkg/pallets"
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

type CommitTimeGetter interface {
	GetCommitTime(hash string) (time.Time, error)
}

func GetCommitTimestamp(c CommitTimeGetter, hash string) (string, error) {
	commitTime, err := c.GetCommitTime(hash)
	if err != nil {
		return "", errors.Wrapf(err, "couldn't check time of commit %s", ShortCommit(hash))
	}
	return ToTimestamp(commitTime), nil
}

// VersionedRepo

func (r VersionedRepo) Path() string {
	return fmt.Sprintf("%s/%s", r.VCSRepoPath, r.RepoSubdir)
}

func CompareVersionedRepos(r, s VersionedRepo) int {
	if r.VCSRepoPath != s.VCSRepoPath {
		if r.VCSRepoPath < s.VCSRepoPath {
			return pallets.CompareLT
		}
		return pallets.CompareGT
	}
	if r.RepoSubdir != s.RepoSubdir {
		if r.RepoSubdir < s.RepoSubdir {
			return pallets.CompareLT
		}
		return pallets.CompareGT
	}
	return pallets.CompareEQ
}

// RepoVersionConfig

// loadRepoVersionConfig loads a RepoVersionConfig from a specified file path in the provided base
// filesystem.
func loadRepoVersionConfig(fsys pallets.PathedFS, filePath string) (RepoVersionConfig, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return RepoVersionConfig{}, errors.Wrapf(
			err, "couldn't read repository version config file %s/%s", fsys.Path(), filePath,
		)
	}
	config := RepoVersionConfig{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return RepoVersionConfig{}, errors.Wrap(err, "couldn't parse repository version config")
	}
	return config, nil
}

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
