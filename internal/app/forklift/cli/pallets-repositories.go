package cli

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/pkg/core"
)

func GetRepoCache(
	wpath string, pallet *forklift.FSPallet, ensureCache bool,
) (*forklift.LayeredRepoCache, *forklift.RepoOverrideCache, error) {
	cache := &forklift.LayeredRepoCache{}
	override, err := makeRepoOverrideCacheFromPallet(pallet)
	if err != nil {
		return nil, nil, err
	}
	cache.Overlay = override

	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, nil, err
	}
	fsCache, err := workspace.GetRepoCache()
	if err != nil && override == nil {
		return nil, nil, err
	}
	cache.Underlay = fsCache

	if ensureCache && !fsCache.Exists() {
		repoReqs, err := pallet.LoadFSRepoReqs("**")
		if err != nil {
			return nil, nil, errors.Wrap(err, "couldn't check whether the pallet requires any repos")
		}
		if len(repoReqs) > 0 {
			return nil, nil, errors.New("you first need to cache the repos specified by your pallet")
		}
	}
	return cache, override, nil
}

func makeRepoOverrideCacheFromPallet(
	pallet *forklift.FSPallet,
) (*forklift.RepoOverrideCache, error) {
	palletAsRepo, err := core.LoadFSRepo(pallet.FS, ".")
	if err != nil {
		// The common case is that the pallet is not a repo (and thus can't be loaded as one), so we
		// mask the error:
		return nil, nil
	}
	return forklift.NewRepoOverrideCache(
		[]*core.FSRepo{palletAsRepo}, map[string][]string{
			// In a pallet which is a repo, the implicit repo requirement is for an empty version string
			palletAsRepo.Path(): {""},
		},
	)
}

// Print

func PrintPalletRepos(indent int, pallet *forklift.FSPallet) error {
	loadedRepos, err := pallet.LoadFSRepoReqs("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't identify repos")
	}
	sort.Slice(loadedRepos, func(i, j int) bool {
		return forklift.CompareGitRepoReqs(
			loadedRepos[i].RepoReq.GitRepoReq, loadedRepos[j].RepoReq.GitRepoReq,
		) < 0
	})
	for _, repo := range loadedRepos {
		IndentedPrintf(indent, "%s\n", repo.Path())
	}
	return nil
}

func PrintRepoInfo(
	indent int, pallet *forklift.FSPallet, cache forklift.PathedRepoCache, repoPath string,
) error {
	req, err := pallet.LoadFSRepoReq(repoPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load repo version lock definition %s from pallet %s",
			repoPath, pallet.FS.Path(),
		)
	}
	printRepoReq(indent, req.RepoReq)
	indent++

	version := req.VersionLock.Version
	cachedRepo, err := cache.LoadFSRepo(repoPath, version)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find repo %s@%s in cache, please update the local cache of repos",
			repoPath, version,
		)
	}
	IndentedPrintf(indent, "Forklift version: %s\n", cachedRepo.Def.ForkliftVersion)
	fmt.Println()

	if core.CoversPath(cache, cachedRepo.FS.Path()) {
		IndentedPrintf(
			indent, "Path in cache: %s\n", core.GetSubdirPath(cache, cachedRepo.FS.Path()),
		)
	} else {
		IndentedPrintf(indent, "Absolute path (replacing any cached copy): %s\n", cachedRepo.FS.Path())
	}
	IndentedPrintf(indent, "Description: %s\n", cachedRepo.Def.Repo.Description)

	readme, err := cachedRepo.LoadReadme()
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load readme file for repo %s@%s from cache", repoPath, version,
		)
	}
	IndentedPrintln(indent, "Readme:")
	const widthLimit = 100
	PrintReadme(indent+1, readme, widthLimit)
	return nil
}

func printRepoReq(indent int, req forklift.RepoReq) {
	IndentedPrintf(indent, "Repo: %s\n", req.Path())
	indent++
	IndentedPrintf(indent, "Locked repo version: %s\n", req.VersionLock.Version)
}

// Download

func DownloadRequiredRepos(
	indent int, pallet *forklift.FSPallet, cache forklift.PathedRepoCache,
) (changed bool, err error) {
	loadedRepoReqs, err := pallet.LoadFSRepoReqs("**")
	if err != nil {
		return false, errors.Wrapf(err, "couldn't identify repos")
	}
	changed = false
	for _, req := range loadedRepoReqs {
		downloaded, err := DownloadLockedGitRepoUsingLocalMirror(
			indent, cache.Path(), req.Path(), req.VersionLock,
		)
		changed = changed || downloaded
		if err != nil {
			return false, errors.Wrapf(
				err, "couldn't download %s at commit %s", req.Path(), req.VersionLock.Def.ShortCommit(),
			)
		}
	}
	return changed, nil
}
