package versioning

import (
	"io/fs"

	"github.com/blang/semver/v4"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Lock

// A Lock is a specification of a particular version of a repo or package.
type Lock struct {
	// Decl is the version lock definition.
	Decl LockDecl `yaml:",inline"`
	// Version is the version string corresponding to the configured version.
	Version string `yaml:"-"`
}

// LoadLock loads a Lock from a specified file path in the provided base filesystem.
// The loaded version lock is fully initialized, including the version field.
func LoadLock(fsys ffs.PathedFS, filePath string) (lock Lock, err error) {
	if lock.Decl, err = loadLockDecl(fsys, filePath); err != nil {
		return Lock{}, errors.Wrapf(err, "couldn't load version lock config")
	}
	if lock.Version, err = lock.Decl.Version(); err != nil {
		return Lock{}, errors.Wrapf(
			err, "couldn't determine version specified by version lock configuration",
		)
	}
	return lock, nil
}

// Check looks for errors in the construction of the version lock.
func (l Lock) Check() (errs []error) {
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

// LockDecl

// LockDeclFile is the name of the file defining each version lock of a repo.
const LockDeclFile = "forklift-version-lock.yml"

// A LockDecl defines a requirement for a repo or package at a specific
// version.
type LockDecl struct {
	// Type specifies the type of version lock (either "version" or "pseudoversion")
	Type string `yaml:"type"`
	// Tag specifies the VCS repository tag associated with the version or pseudoversion, if it
	// exists. If the type is "version", the tag should point to the commit corresponding to the
	// version; if the type is "pseudoversion", the tag should be the highest-versioned tag in the
	// ancestry of the commit corresponding to the version (and it is used as a "base version").
	Tag string `yaml:"tag,omitempty"`
	// Timestamp specifies the commit time (in UTC) of the commit corresponding to the version, as
	// a 14-character string.
	Timestamp string `yaml:"timestamp"`
	// Commit specifies the full hash of the commit corresponding to the version.
	Commit string `yaml:"commit"`
}

// loadLockDecl loads a LockDecl from a specified file path in the provided base
// filesystem.
func loadLockDecl(fsys ffs.PathedFS, filePath string) (d LockDecl, err error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return d, errors.Wrapf(
			err, "couldn't read version lock definition file %s/%s", fsys.Path(), filePath,
		)
	}
	if err = yaml.Unmarshal(bytes, &d); err != nil {
		return d, errors.Wrap(err, "couldn't parse version lock definition")
	}
	return d, nil
}

func (l LockDecl) IsCommitLocked() bool {
	return l.Commit != ""
}

func (l LockDecl) ShortCommit() string {
	return ShortCommit(l.Commit)
}

func (l LockDecl) ParseVersion() (v semver.Version, err error) {
	return parseTag(l.Tag)
}

func (l LockDecl) Pseudoversion() (string, error) {
	return Pseudoversion(l.Tag, l.Timestamp, l.Commit)
}

const (
	LockTypeVersion       = "version"
	LockTypePseudoversion = "pseudoversion"
)

func (l LockDecl) Version() (string, error) {
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
