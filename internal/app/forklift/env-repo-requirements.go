package forklift

import (
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// RepoRequirement

// LoadFSRepo loads a FSRepoRequirement from the specified directory path in the provided base
// filesystem, assuming the directory path is also the Pallet repository path of the required
// repository.
// The loaded RepoRequirement is fully initialized.
func loadFSRepoRequirement(
	fsys pallets.PathedFS, repoPath string,
) (r *FSRepoRequirement, err error) {
	r = &FSRepoRequirement{}
	if r.FS, err = fsys.Sub(repoPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", repoPath, fsys.Path(),
		)
	}
	if r.VCSRepoPath, r.RepoSubdir, err = pallets.SplitRepoPathSubdir(
		repoPath,
	); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't parse path of requirement for Pallet repo %s", repoPath,
		)
	}
	r.VersionLock, err = loadVersionLock(r.FS, VersionLockSpecFile)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load version lock config of requirement for Pallet repo %s", repoPath,
		)
	}
	return r, nil
}

// loadFSRepoRequirements loads all FSRepoRequirements from the provided base filesystem matching
// the specified search pattern, assuming the directory paths in the base filesystem are also the
// Pallet repository paths of the required repositories. The search pattern should be a [doublestar]
// pattern, such as `**`, matching the repo paths to search for.
// With a nil processor function, in the embedded [Repo] of each loaded FSRepo, the VCS repository
// path and Pallet repository subdirectory are initialized from the Pallet repository path declared
// in the repository's configuration file, while the Pallet repository version is not initialized.
func loadFSRepoRequirements(
	fsys pallets.PathedFS, searchPattern string,
) ([]*FSRepoRequirement, error) {
	searchPattern = filepath.Join(searchPattern, VersionLockSpecFile)
	repoRequirementFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for Pallet repo requirement files matching %s/%s",
			fsys.Path(), searchPattern,
		)
	}

	requirements := make([]*FSRepoRequirement, 0, len(repoRequirementFiles))
	for _, repoRequirementConfigFilePath := range repoRequirementFiles {
		if filepath.Base(repoRequirementConfigFilePath) != VersionLockSpecFile {
			continue
		}

		requirement, err := loadFSRepoRequirement(fsys, filepath.Dir(repoRequirementConfigFilePath))
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't load repo requirement from %s", repoRequirementConfigFilePath,
			)
		}
		requirements = append(requirements, requirement)
	}
	return requirements, nil
}

func (r RepoRequirement) Path() string {
	return filepath.Join(r.VCSRepoPath, r.RepoSubdir)
}

// CompareRepos returns an integer comparing two [RepoRequirement] instances according to their
// paths and versions. The result will be 0 if the r and s have the same paths and versions; -1 if r
// has a path which alphabetically comes before the path of s or if the paths are the same but r has
// a lower version than s; or +1 if r has a path which alphabetically comes after the path of s or
// if the paths are the same but r has a higher version than s.
func CompareRepoRequirements(r, s RepoRequirement) int {
	if result := pallets.ComparePaths(r.Path(), s.Path()); result != pallets.CompareEQ {
		return result
	}
	if result := semver.Compare(
		r.VersionLock.Version, s.VersionLock.Version,
	); result != pallets.CompareEQ {
		return result
	}
	return pallets.CompareEQ
}
