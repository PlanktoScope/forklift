package cli

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	ggit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/clients/git"
	"github.com/PlanktoScope/forklift/pkg/core"
)

func GetPalletCache(
	wpath string, pallet *forklift.FSPallet, ensureCache bool,
) (*forklift.FSPalletCache, error) {
	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	cache, err := workspace.GetPalletCache()
	if err != nil {
		return nil, err
	}

	if ensureCache && !cache.Exists() && pallet != nil {
		palletReqs, err := pallet.LoadFSPalletReqs("**")
		if err != nil {
			return nil, errors.Wrap(err, "couldn't check whether the pallet requires any pallets")
		}
		if len(palletReqs) > 0 {
			return nil, errors.New("you first need to cache the pallets specified by your pallet")
		}
	}
	return cache, nil
}

// Print

func PrintCachedPallet(indent int, cache core.Pather, pallet *forklift.FSPallet) error {
	IndentedPrintf(indent, "Cached pallet: %s\n", pallet.Path())
	indent++

	IndentedPrintf(indent, "Forklift version: %s\n", pallet.Def.ForkliftVersion)
	fmt.Println()

	IndentedPrintf(indent, "Version: %s\n", pallet.Version)
	IndentedPrintf(indent, "Path in cache: %s\n", core.GetSubdirPath(cache, pallet.FS.Path()))
	IndentedPrintf(indent, "Description: %s\n", pallet.Def.Pallet.Description)

	readme, err := pallet.LoadReadme()
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load readme file for pallet %s@%s from cache", pallet.Path(), pallet.Version,
		)
	}
	IndentedPrintln(indent, "Readme:")
	const widthLimit = 100
	PrintReadme(indent+1, readme, widthLimit)
	return nil
}

func PrintPalletInfo(indent int, pallet *forklift.FSPallet) error {
	IndentedPrintf(indent, "Pallet: %s\n", pallet.Path())
	indent++

	IndentedPrintf(indent, "Forklift version: %s\n", pallet.Def.ForkliftVersion)
	fmt.Println()

	if pallet.Def.Pallet.Path != "" {
		IndentedPrintf(indent, "Path in filesystem: %s\n", pallet.FS.Path())
	}
	IndentedPrintf(indent, "Description: %s\n", pallet.Def.Pallet.Description)
	if pallet.Def.Pallet.ReadmeFile == "" {
		fmt.Println()
	} else {
		readme, err := pallet.LoadReadme()
		if err != nil {
			return errors.Wrapf(err, "couldn't load readme file for pallet %s", pallet.FS.Path())
		}
		IndentedPrintln(indent, "Readme:")
		const widthLimit = 100
		PrintReadme(indent+1, readme, widthLimit)
	}

	return printGitRepoInfo(indent, pallet.FS.Path())
}

func printGitRepoInfo(indent int, palletPath string) error {
	ref, err := git.Head(palletPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query pallet %s for its HEAD", palletPath)
	}
	IndentedPrintf(indent, "Currently on: %s\n", git.StringifyRef(ref))
	// TODO: report any divergence between head and remotes
	if err := printUncommittedChanges(indent+1, palletPath); err != nil {
		return err
	}
	if err := printLocalRefsInfo(indent, palletPath); err != nil {
		return err
	}
	if err := printRemotesInfo(indent, palletPath); err != nil {
		return err
	}
	return nil
}

func printUncommittedChanges(indent int, palletPath string) error {
	status, err := git.Status(palletPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query the pallet %s for its status", palletPath)
	}
	IndentedPrint(indent, "Uncommitted changes:")
	if len(status) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for file, status := range status {
		if status.Staging == git.StatusUnmodified && status.Worktree == git.StatusUnmodified {
			continue
		}
		if status.Staging == git.StatusRenamed {
			file = fmt.Sprintf("%s -> %s", file, status.Extra)
		}
		BulletedPrintf(indent, "%c%c %s\n", status.Staging, status.Worktree, file)
	}
	return nil
}

func printLocalRefsInfo(indent int, palletPath string) error {
	refs, err := git.Refs(palletPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query pallet %s for its refs", palletPath)
	}

	IndentedPrintf(indent, "References:")
	if len(refs) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, ref := range refs {
		BulletedPrintf(indent, "%s\n", git.StringifyRef(ref))
	}

	return nil
}

func printRemotesInfo(indent int, palletPath string) error {
	remotes, err := git.Remotes(palletPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't query pallet %s for its remotes", palletPath)
	}

	IndentedPrintf(indent, "Remotes:")
	if len(remotes) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	SortRemotes(remotes)
	printCacheMirrorRemote := false
	for _, remote := range remotes {
		if remote.Config().Name == ForkliftCacheMirrorRemoteName && !printCacheMirrorRemote {
			IndentedPrintf(
				indent, "%s: (skipped because origin was successfully queried)\n", remote.Config().Name,
			)
			continue
		}

		if err := printRemoteInfo(
			indent, remote,
		); err != nil && remote.Config().Name == OriginRemoteName {
			printCacheMirrorRemote = true
		}
	}
	return nil
}

func SortRemotes(remotes []*ggit.Remote) {
	slices.SortFunc(remotes, func(a, b *ggit.Remote) int {
		if a.Config().Name == OriginRemoteName {
			return -1
		}
		if b.Config().Name == OriginRemoteName {
			return 1
		}
		return cmp.Compare(a.Config().Name, b.Config().Name)
	})
}

func printRemoteInfo(indent int, remote *ggit.Remote) error {
	config := remote.Config()
	IndentedPrintf(indent, "%s:\n", config.Name)
	indent++

	IndentedPrintf(indent, "URLs:")
	if len(config.URLs) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for i, url := range config.URLs {
		BulletedPrintf(indent+1, "%s: ", url)
		if i == 0 {
			fmt.Print("fetch, ")
		}
		fmt.Println("push")
	}

	IndentedPrintf(indent, "Up-to-date references:")
	refs, err := remote.List(git.EmptyListOptions())
	if err != nil {
		fmt.Printf(" (couldn't retrieve references: %s)\n", err)
		return err
	}

	if len(refs) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	slices.SortFunc(refs, func(a, b *plumbing.Reference) int {
		return cmp.Compare(git.StringifyRef(a), git.StringifyRef(b))
	})
	for _, ref := range refs {
		if strings.HasPrefix(git.StringifyRef(ref), "pull/") {
			continue
		}
		BulletedPrintf(indent+1, "%s\n", git.StringifyRef(ref))
	}

	return nil
}

// Download

func DownloadRequiredPallets(
	indent int, pallet *forklift.FSPallet, cachePath string,
) (changed bool, err error) {
	loadedPalletReqs, err := pallet.LoadFSPalletReqs("**")
	if err != nil {
		return false, errors.Wrapf(err, "couldn't identify pallets")
	}
	changed = false
	for _, req := range loadedPalletReqs {
		downloaded, err := DownloadLockedGitRepoUsingLocalMirror(
			indent, cachePath, req.Path(), req.VersionLock,
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

// Cache

func CacheAllRequirements(
	pallet *forklift.FSPallet, repoCachePath string,
	pkgLoader forklift.FSPkgLoader, dlCache *forklift.FSDownloadCache,
	includeDisabled, parallel bool,
) error {
	if err := CacheStagingRequirements(
		pallet, repoCachePath, pkgLoader, dlCache, includeDisabled, parallel,
	); err != nil {
		return err
	}

	fmt.Println("Downloading Docker container images to be deployed by the local pallet...")
	if err := DownloadImages(1, pallet, pkgLoader, includeDisabled, parallel); err != nil {
		return err
	}
	return nil
}

func CacheStagingRequirements(
	pallet *forklift.FSPallet, repoCachePath string,
	pkgLoader forklift.FSPkgLoader, dlCache *forklift.FSDownloadCache,
	includeDisabled, parallel bool,
) error {
	// TODO: download required pallets, once we allow layering pallets; then merge the pallets into
	// a composite before downloading required repos

	fmt.Println("Downloading repos required by the local pallet...")
	if _, err := DownloadRequiredRepos(0, pallet, repoCachePath); err != nil {
		return err
	}

	fmt.Println("Downloading files for export by the local pallet...")
	if err := DownloadExportFiles(
		1, pallet, pkgLoader, dlCache, includeDisabled, parallel,
	); err != nil {
		return err
	}

	// TODO: warn if any downloaded repo doesn't appear to be an actual repo, or if any repo's
	// forklift version is incompatible or ahead of the pallet version

	return nil
}
