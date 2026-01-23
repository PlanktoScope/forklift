package forklift

import (
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ffs "github.com/forklift-run/forklift/pkg/fs"
)

// Pseudo-versions

func ToTimestamp(t time.Time) string {
	return t.UTC().Format(Timestamp)
}

func ShortCommit(commit string) string {
	const truncatedLength = 12
	return commit[:truncatedLength]
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
func loadVersionLock(fsys ffs.PathedFS, filePath string) (lock VersionLock, err error) {
	if lock.Decl, err = loadVersionLockDecl(fsys, filePath); err != nil {
		return VersionLock{}, errors.Wrapf(err, "couldn't load version lock config")
	}
	if lock.Version, err = lock.Decl.Version(); err != nil {
		return VersionLock{}, errors.Wrapf(
			err, "couldn't determine version specified by version lock configuration",
		)
	}
	return lock, nil
}

// Check looks for errors in the construction of the version lock.
func (l VersionLock) Check() (errs []error) {
	configVersion, err := l.Decl.Version()
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

// VersionLockDecl

// loadVersionLockDecl loads a VersionLockDecl from a specified file path in the provided base
// filesystem.
func loadVersionLockDecl(fsys ffs.PathedFS, filePath string) (VersionLockDecl, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return VersionLockDecl{}, errors.Wrapf(
			err, "couldn't read version lock definition file %s/%s", fsys.Path(), filePath,
		)
	}
	config := VersionLockDecl{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return VersionLockDecl{}, errors.Wrap(err, "couldn't parse version lock definition")
	}
	return config, nil
}

func (l VersionLockDecl) IsCommitLocked() bool {
	return l.Commit != ""
}

func (l VersionLockDecl) ShortCommit() string {
	return ShortCommit(l.Commit)
}

func (l VersionLockDecl) ParseVersion() (semver.Version, error) {
	if !strings.HasPrefix(l.Tag, "v") {
		return semver.Version{}, errors.Errorf("invalid tag `%s` doesn't start with `v`", l.Tag)
	}
	version, err := semver.Parse(strings.TrimPrefix(l.Tag, "v"))
	if err != nil {
		return semver.Version{}, errors.Errorf(
			"tag `%s` couldn't be parsed as a semantic version", l.Tag,
		)
	}
	return version, nil
}

func (l VersionLockDecl) Pseudoversion() (string, error) {
	// This implements the specification described at https://go.dev/ref/mod#pseudo-versions
	if l.Commit == "" {
		return "", errors.Errorf("pseudoversion missing commit hash")
	}
	if l.Timestamp == "" {
		return "", errors.Errorf("pseudoversion missing commit timestamp")
	}
	revisionID := ShortCommit(l.Commit)
	if l.Tag == "" {
		return fmt.Sprintf("v0.0.0-%s-%s", l.Timestamp, revisionID), nil
	}
	parsed, err := l.ParseVersion()
	if err != nil {
		return "", err
	}
	parsed.Build = nil
	if len(parsed.Pre) > 0 {
		return fmt.Sprintf("v%s.0.%s-%s", parsed.String(), l.Timestamp, revisionID), nil
	}
	return fmt.Sprintf(
		"v%d.%d.%d-0.%s-%s", parsed.Major, parsed.Minor, parsed.Patch+1, l.Timestamp, revisionID,
	), nil
}

const (
	LockTypeVersion       = "version"
	LockTypePseudoversion = "pseudoversion"
)

func (l VersionLockDecl) Version() (string, error) {
	switch l.Type {
	case LockTypeVersion:
		version, err := l.ParseVersion()
		if err != nil {
			return "", errors.Wrap(err, "invalid version")
		}
		return "v" + version.String(), nil
	case LockTypePseudoversion:
		pseudoversion, err := l.Pseudoversion()
		if err != nil {
			return "", errors.Wrap(err, "couldn't determine pseudo-version")
		}
		return pseudoversion, nil
	default:
		return "", errors.Errorf("unknown lock type %s", l.Type)
	}
}
