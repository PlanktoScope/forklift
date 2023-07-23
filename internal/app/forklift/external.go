package forklift

import (
	"path/filepath"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// TODO: see if we can remove the need for this
func AsVersionedPkg(pkg *pallets.FSPkg) *VersionedPkg {
	return &VersionedPkg{
		FSPkg: pkg,
		RepoRequirement: &FSRepoRequirement{
			RepoRequirement: RepoRequirement{
				VCSRepoPath: pkg.Repo.VCSRepoPath,
				RepoSubdir:  pkg.Repo.Subdir,
			},
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
