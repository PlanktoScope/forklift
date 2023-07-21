package pallets

import (
	"io/fs"
	"path/filepath"

	"github.com/pkg/errors"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// Repo

// Path returns the Pallet repository path of the Repo instance.
func (r Repo) Path() string {
	return filepath.Join(r.VCSRepoPath, r.Subdir)
}

// FromSameVCSRepo determines whether the candidate Pallet repository is provided by the same VCS
// repo as the Repo instance.
func (r Repo) FromSameVCSRepo(candidate Repo) bool {
	return r.VCSRepoPath == candidate.VCSRepoPath && r.Version == candidate.Version
}

// CompareRepoPaths returns an integer comparing two [Repo] instances according to their paths. The
// result will be 0 if the r and s have the same paths; -1 if r has a path which alphabetically
// comes before the path of s; or +1 if r has a path which alphabetically comes after the path of s.
// A path comes before another path if the VCS repository component comes before, or if the VCS
// repository components are equal but the subdirectory component comes before.
func CompareRepoPaths(r, s Repo) int {
	if r.VCSRepoPath != s.VCSRepoPath {
		if r.VCSRepoPath < s.VCSRepoPath {
			return CompareLT
		}
		return CompareGT
	}
	if r.Subdir != s.Subdir {
		if r.Subdir < s.Subdir {
			return CompareLT
		}
		return CompareGT
	}
	return CompareEQ
}

// CompareRepos returns an integer comparing two [Repo] instances according to their paths and
// versions. The result will be 0 if the r and s have the same paths and versions; -1 if r has a
// path which alphabetically comes before the path of s or if the paths are the same but r has a
// lower version than s; or +1 if r has a path which alphabetically comes after the path of s or if
// the paths are the same but r has a higher version than s.
func CompareRepos(r, s Repo) int {
	if result := CompareRepoPaths(r, s); result != CompareEQ {
		return result
	}
	if result := semver.Compare(r.Version, s.Version); result != CompareEQ {
		return result
	}
	return CompareEQ
}

// FSRepo

// LoadFSRepo loads a FSRepo from the specified directory path in the provided base filesystem.
// The base path should correspond to the location of the base filesystem. In the loaded FSRepo's
// embedded [Repo], the Pallet repository path is not initialized, nor is the Pallet repository
// subdirectory initialized, nor is the Pallet repository version initialized.
func LoadFSRepo(baseFS fs.FS, baseFSPath, repoFSPath string) (p FSRepo, err error) {
	p = FSRepo{
		FSPath: filepath.Join(baseFSPath, repoFSPath),
	}
	if p.FS, err = fs.Sub(baseFS, repoFSPath); err != nil {
		return FSRepo{}, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", repoFSPath, baseFSPath,
		)
	}
	if p.Repo.Config, err = LoadRepoConfig(p.FS, p.FSPath, RepoSpecFile); err != nil {
		return FSRepo{}, errors.Wrapf(err, "couldn't load repo config")
	}
	return p, nil
}

// RepoConfig

// LoadRepoConfig loads a RepoConfig from the specified file path in the provided base filesystem.
// The base path should correspond to the location of the base filesystem.
func LoadRepoConfig(baseFS fs.FS, baseFSPath, filePath string) (RepoConfig, error) {
	bytes, err := fs.ReadFile(baseFS, filePath)
	if err != nil {
		return RepoConfig{}, errors.Wrapf(
			err, "couldn't read repo config file %s/%s", baseFSPath, filePath,
		)
	}
	config := RepoConfig{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return RepoConfig{}, errors.Wrap(err, "couldn't parse repo config")
	}
	return config, nil
}

func (c RepoConfig) Check() (errs []error) {
	return ErrsWrap(c.Repository.Check(), "invalid repo spec")
}

// RepoSpec

func (s RepoSpec) Check() (errs []error) {
	if s.Path == "" {
		errs = append(errs, errors.Errorf("repo spec is missing `path` parameter"))
	}
	return errs
}
