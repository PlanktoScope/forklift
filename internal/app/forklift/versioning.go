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

func ToTimestamp(t time.Time) string {
	return t.UTC().Format(Timestamp)
}

const shortCommitLength = 12

// TODO: move this somewhere else?
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

// VersionLock

// loadVersionLock loads a VersionLock from a specified file path in the provided base filesystem.
// The loaded version lock is fully initialized, including the version field.
func loadVersionLock(fsys pallets.PathedFS, filePath string) (lock VersionLock, err error) {
	if lock.Config, err = loadVersionLockConfig(fsys, filePath); err != nil {
		return VersionLock{}, errors.Wrapf(err, "couldn't load version lock config")
	}
	if lock.Version, err = lock.Config.Version(); err != nil {
		return VersionLock{}, errors.Wrapf(
			err, "couldn't determine version specified by version lock configuration",
		)
	}
	return lock, nil
}

// Check looks for errors in the construction of the version lock.
func (l VersionLock) Check() (errs []error) {
	configVersion, err := l.Config.Version()
	if err != nil {
		errs = append(errs, errors.Wrap(
			err, "couldn't determine version specified by version lock configuration",
		))
		return errs
	}
	if l.Version != configVersion {
		errs = append(errs, errors.Wrapf(
			err, "version %s is inconsistent with version %s specified by version lock configuration",
			l.Version, configVersion,
		))
	}
	return errs
}

// VersionLockConfig

// loadVersionLockConfig loads a VersionLockConfig from a specified file path in the provided base
// filesystem.
func loadVersionLockConfig(fsys pallets.PathedFS, filePath string) (VersionLockConfig, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return VersionLockConfig{}, errors.Wrapf(
			err, "couldn't read version lock config file %s/%s", fsys.Path(), filePath,
		)
	}
	config := VersionLockConfig{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return VersionLockConfig{}, errors.Wrap(err, "couldn't parse repository version config")
	}
	return config, nil
}

func (c VersionLockConfig) IsCommitLocked() bool {
	return c.Commit != ""
}

func (c VersionLockConfig) ShortCommit() string {
	return ShortCommit(c.Commit)
}

func (c VersionLockConfig) IsVersion() bool {
	return c.BaseVersion != "" && c.Commit != ""
}

func (c VersionLockConfig) ParseVersion() (semver.Version, error) {
	if !strings.HasPrefix(c.BaseVersion, "v") {
		return semver.Version{}, errors.Errorf(
			"invalid repo version `%s` doesn't start with `v`", c.BaseVersion,
		)
	}
	version, err := semver.Parse(strings.TrimPrefix(c.BaseVersion, "v"))
	if err != nil {
		return semver.Version{}, errors.Errorf(
			"repo version `%s` couldn't be parsed as a semantic version", c.BaseVersion,
		)
	}
	return version, nil
}

func (c VersionLockConfig) Pseudoversion() (string, error) {
	// This implements the specification described at https://go.dev/ref/mod#pseudo-versions
	if c.Commit == "" {
		return "", errors.Errorf("repo version missing commit hash")
	}
	if c.Timestamp == "" {
		return "", errors.Errorf("repo version missing commit timestamp")
	}
	revisionID := ShortCommit(c.Commit)
	if c.BaseVersion == "" {
		return fmt.Sprintf("v0.0.0-%s-%s", c.Timestamp, revisionID), nil
	}
	version, err := c.ParseVersion()
	if err != nil {
		return "", err
	}
	version.Build = nil
	if len(version.Pre) > 0 {
		return fmt.Sprintf("%s.0.%s-%s", version.String(), c.Timestamp, revisionID), nil
	}
	return fmt.Sprintf(
		"v%d.%d.%d-0.%s-%s", version.Major, version.Minor, version.Patch+1, c.Timestamp, revisionID,
	), nil
}

func (c VersionLockConfig) Version() (string, error) {
	if c.IsVersion() {
		version, err := c.ParseVersion()
		if err != nil {
			return "", errors.Wrap(err, "invalid repo version")
		}
		return version.String(), nil
	}
	pseudoversion, err := c.Pseudoversion()
	if err != nil {
		return "", errors.Wrap(err, "couldn't determine pseudo-version")
	}
	return pseudoversion, nil
}
