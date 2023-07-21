package forklift

import (
	gosemver "golang.org/x/mod/semver"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Sorting

// TODO: move this into pkg/pallets
func CompareCachedPkgs(p, q *pallets.FSPkg) int {
	repoPathComparison := pallets.CompareRepoPaths(p.Repo.Repo, q.Repo.Repo)
	if repoPathComparison != pallets.CompareEQ {
		return repoPathComparison
	}
	if p.Subdir != q.Subdir {
		if p.Subdir < q.Subdir {
			return pallets.CompareLT
		}
		return pallets.CompareGT
	}
	repoVersionComparison := gosemver.Compare(p.Repo.Version, q.Repo.Version)
	if repoVersionComparison != pallets.CompareEQ {
		return repoVersionComparison
	}
	return pallets.CompareEQ
}
