package forklift

import (
	"slices"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/pkg/core"
)

// LoadFSRepos loads all FSRepos from the provided base filesystem matching the specified search
// pattern, appropriately handling repos defined (implicitly or explicitly) as potentially-layered
// pallets. The search pattern should be a [doublestar] pattern, such as `**`, matching repo
// directories to search for.
// In the embedded [Repo] of each loaded FSRepo, the version is *not* initialized.
func LoadFSRepos(
	fsys core.PathedFS, searchPattern string, palletLoader FSPalletLoader,
) ([]*core.FSRepo, error) {
	allRepos := make(map[string]*core.FSRepo) // repo FS path -> repo
	pallets, err := LoadFSPallets(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load pallets (which are also repos)")
	}
	for _, pallet := range pallets {
		merged, err := MergeFSPallet(pallet, palletLoader, nil)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't merge pallet %s with any pallets required by it, to use it as a repo",
				pallet.FS.Path(),
			)
		}
		allRepos[pallet.FS.Path()] = merged.Repo
	}

	repos, err := core.LoadFSRepos(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load repos")
	}
	for _, repo := range repos {
		if _, ok := allRepos[repo.FS.Path()]; ok { // the repo might've already been added as a pallet
			continue
		}
		allRepos[repo.FS.Path()] = repo
	}

	repos = make([]*core.FSRepo, 0, len(allRepos))
	for _, repo := range allRepos {
		repos = append(repos, repo)
	}
	slices.SortFunc(repos, func(a, b *core.FSRepo) int {
		return core.CompareRepos(a.Repo, b.Repo)
	})
	return repos, nil
}
