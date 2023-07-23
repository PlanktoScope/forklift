package forklift

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// RepoReq

// LoadFSRepo loads a FSRepoReq from the specified directory path in the provided base filesystem,
// assuming the directory path is also the Pallet repository path of the required repository.
func loadFSRepoReq(fsys pallets.PathedFS, repoPath string) (r *FSRepoReq, err error) {
	r = &FSRepoReq{}
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

// loadFSRepoReqContaining loads the FSRepoReq containing the specified sub-directory path in the
// provided base filesystem.
// The sub-directory path does not have to actually exist; however, it would usually be provided
// as a Pallet package path.
func loadFSRepoReqContaining(fsys pallets.PathedFS, subdirPath string) (*FSRepoReq, error) {
	repoCandidatePath := subdirPath
	for {
		if repo, err := loadFSRepoReq(fsys, repoCandidatePath); err == nil {
			return repo, nil
		}
		repoCandidatePath = filepath.Dir(repoCandidatePath)
		if repoCandidatePath == "/" || repoCandidatePath == "." {
			// we can't go up anymore!
			return nil, errors.Errorf(
				"no repo requirement config file found in any parent directory of %s", subdirPath,
			)
		}
	}
}

// loadFSRepoReqs loads all FSRepoReqs from the provided base filesystem matching the specified
// search pattern, assuming the directory paths in the base filesystem are also the Pallet
// repository paths of the required repositories. The search pattern should be a [doublestar]
// pattern, such as `**`, matching the repo paths to search for.
// With a nil processor function, in the embedded [Repo] of each loaded FSRepo, the VCS repository
// path and Pallet repository subdirectory are initialized from the Pallet repository path declared
// in the repository's configuration file, while the Pallet repository version is not initialized.
func loadFSRepoReqs(fsys pallets.PathedFS, searchPattern string) ([]*FSRepoReq, error) {
	searchPattern = filepath.Join(searchPattern, VersionLockSpecFile)
	repoReqFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for Pallet repo requirement files matching %s/%s",
			fsys.Path(), searchPattern,
		)
	}

	requirements := make([]*FSRepoReq, 0, len(repoReqFiles))
	for _, repoReqConfigFilePath := range repoReqFiles {
		if filepath.Base(repoReqConfigFilePath) != VersionLockSpecFile {
			continue
		}

		requirement, err := loadFSRepoReq(fsys, filepath.Dir(repoReqConfigFilePath))
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't load repo requirement from %s", repoReqConfigFilePath,
			)
		}
		requirements = append(requirements, requirement)
	}
	return requirements, nil
}

// Path returns the Pallet repository path of the required repository.
func (r RepoReq) Path() string {
	return filepath.Join(r.VCSRepoPath, r.RepoSubdir)
}

// GetPkgSubdir returns the Pallet package subdirectory within the required repo for the provided
// Pallet package path.
func (r RepoReq) GetPkgSubdir(pkgPath string) string {
	return strings.TrimPrefix(pkgPath, fmt.Sprintf("%s/", r.Path()))
}

// CompareRepoReqs returns an integer comparing two [RepoReq] instances according to their paths and
// versions. The result will be 0 if the r and s have the same paths and versions; -1 if r has a
// path which alphabetically comes before the path of s or if the paths are the same but r has a
// lower version than s; or +1 if r has a path which alphabetically comes after the path of s or if
// the paths are the same but r has a higher version than s.
func CompareRepoReqs(r, s RepoReq) int {
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
