package forklift

import (
	"path/filepath"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

func AsVersionedPkg(pkg *pallets.FSPkg) *VersionedPkg {
	return &VersionedPkg{
		FSPkg: pkg,
		VersionedRepo: VersionedRepo{
			VCSRepoPath: pkg.Repo.VCSRepoPath,
			RepoSubdir:  pkg.Repo.Subdir,
		},
	}
}

func FindExternalRepoOfPkg(
	repos map[string]*pallets.FSRepo, pkgPath string,
) (repo *pallets.FSRepo, ok bool) {
	repoCandidatePath := filepath.Dir(pkgPath)
	for repoCandidatePath != "." {
		if repo, ok = repos[repoCandidatePath]; ok {
			return repo, true
		}
		repoCandidatePath = filepath.Dir(repoCandidatePath)
	}
	return nil, false
}
