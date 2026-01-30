package pallets

import (
	"cmp"
	"fmt"

	"golang.org/x/mod/semver"

	"github.com/forklift-run/forklift/exp/versioning"
)

// A GitRepoReq is a requirement for a specific Git repository (e.g. pallet) at a specific version.
type GitRepoReq struct {
	// GitRepoPath is the path of the required Git repository.
	RequiredPath string `yaml:"-"`
	// VersionLock specifies the version of the required Git repository.
	VersionLock versioning.Lock `yaml:"version-lock"`
}

// Path returns the path of the required Git repo.
func (r GitRepoReq) Path() string {
	return r.RequiredPath
}

// GetQueryPath returns the path of the Git repo in version queries, which is of form
// gitRepoPath@version (e.g. github.com/PlanktoScope/pallet-standard@v2024.0.0).
func (r GitRepoReq) GetQueryPath() string {
	return fmt.Sprintf("%s@%s", r.Path(), r.VersionLock.Version)
}

// CompareGitRepoReqs returns an integer comparing two [RepoReq] instances according to their paths
// and versions. The result will be 0 if the r and s have the same paths and versions; -1 if r has a
// path which alphabetically comes before the path of s or if the paths are the same but r has a
// lower version than s; or +1 if r has a path which alphabetically comes after the path of s or if
// the paths are the same but r has a higher version than s.
func CompareGitRepoReqs(r, s GitRepoReq) int {
	if result := cmp.Compare(r.Path(), s.Path()); result != 0 {
		return result
	}
	if result := semver.Compare(r.VersionLock.Version, s.VersionLock.Version); result != 0 {
		return result
	}
	return 0
}
