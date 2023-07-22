package forklift

import (
	"fmt"
	"io/fs"
	"path/filepath"
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

// RepoVersionRequirement

func (r RepoVersionRequirement) Path() string {
	return filepath.Join(r.VCSRepoPath, r.RepoSubdir)
}

// TODO: rename this method
func (r RepoVersionRequirement) listVersionedPkgs(
	cache *FSCache,
) (pkgMap map[string]*pallets.FSPkg, versionedPkgPaths []string, err error) {
	repoVersion, err := r.Config.Version()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't determine version of repo %s", r.Path())
	}
	repoCachePath := filepath.Join(fmt.Sprintf("%s@%s", r.VCSRepoPath, repoVersion), r.RepoSubdir)
	pkgs, err := cache.LoadFSPkgs(repoCachePath)
	if err != nil {
		return nil, nil, errors.Wrapf(
			err, "couldn't list packages from repo cached at %s", repoCachePath,
		)
	}

	pkgMap = make(map[string]*pallets.FSPkg)
	for _, pkg := range pkgs {
		versionedPkgPath := fmt.Sprintf(
			"%s@%s/%s", pkg.Repo.Config.Repository.Path, pkg.Repo.Version, pkg.Subdir,
		)
		if prevPkg, ok := pkgMap[versionedPkgPath]; ok {
			if prevPkg.Repo.FromSameVCSRepo(pkg.Repo.Repo) && prevPkg.FS.Path() != pkg.FS.Path() {
				return nil, nil, errors.Errorf(
					"package repeatedly defined in the same version of the same cached Github repo: %s, %s",
					prevPkg.FS.Path(), pkg.FS.Path(),
				)
			}
		}
		versionedPkgPaths = append(versionedPkgPaths, versionedPkgPath)
		pkgMap[versionedPkgPath] = pkg
	}

	return pkgMap, versionedPkgPaths, nil
}

func CompareRepoVersionRequirements(r, s RepoVersionRequirement) int {
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

// VersionLockConfig

// loadVersionLockConfig loads a VersionLockConfig from a specified file path in the
// provided base filesystem.
func loadVersionLockConfig(
	fsys pallets.PathedFS, filePath string,
) (VersionLockConfig, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return VersionLockConfig{}, errors.Wrapf(
			err, "couldn't read repository version config file %s/%s", fsys.Path(), filePath,
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
