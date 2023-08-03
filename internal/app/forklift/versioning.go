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

func ShortCommit(commit string) string {
	const shortCommitLength = 12
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
	if lock.Def, err = loadVersionLockDef(fsys, filePath); err != nil {
		return VersionLock{}, errors.Wrapf(err, "couldn't load version lock config")
	}
	if lock.Version, err = lock.Def.Version(); err != nil {
		return VersionLock{}, errors.Wrapf(
			err, "couldn't determine version specified by version lock configuration",
		)
	}
	return lock, nil
}

// Check looks for errors in the construction of the version lock.
func (l VersionLock) Check() (errs []error) {
	configVersion, err := l.Def.Version()
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

// VersionLockDef

// loadVersionLockDef loads a VersionLockDef from a specified file path in the provided base
// filesystem.
func loadVersionLockDef(fsys pallets.PathedFS, filePath string) (VersionLockDef, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return VersionLockDef{}, errors.Wrapf(
			err, "couldn't read version lock definition file %s/%s", fsys.Path(), filePath,
		)
	}
	config := VersionLockDef{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return VersionLockDef{}, errors.Wrap(err, "couldn't parse version lock definition")
	}
	return config, nil
}

func (c VersionLockDef) IsCommitLocked() bool {
	return c.Commit != ""
}

func (c VersionLockDef) ShortCommit() string {
	return ShortCommit(c.Commit)
}

func (c VersionLockDef) IsVersion() bool {
	return c.BaseVersion != "" && c.Commit != ""
}

func (c VersionLockDef) ParseVersion() (semver.Version, error) {
	if !strings.HasPrefix(c.BaseVersion, "v") {
		return semver.Version{}, errors.Errorf(
			"invalid version `%s` doesn't start with `v`", c.BaseVersion,
		)
	}
	version, err := semver.Parse(strings.TrimPrefix(c.BaseVersion, "v"))
	if err != nil {
		return semver.Version{}, errors.Errorf(
			"version `%s` couldn't be parsed as a semantic version", c.BaseVersion,
		)
	}
	return version, nil
}

func (c VersionLockDef) Pseudoversion() (string, error) {
	// This implements the specification described at https://go.dev/ref/mod#pseudo-versions
	if c.Commit == "" {
		return "", errors.Errorf("version missing commit hash")
	}
	if c.Timestamp == "" {
		return "", errors.Errorf("version missing commit timestamp")
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

func (c VersionLockDef) Version() (string, error) {
	if c.IsVersion() {
		version, err := c.ParseVersion()
		if err != nil {
			return "", errors.Wrap(err, "invalid version")
		}
		return version.String(), nil
	}
	pseudoversion, err := c.Pseudoversion()
	if err != nil {
		return "", errors.Wrap(err, "couldn't determine pseudo-version")
	}
	return pseudoversion, nil
}
