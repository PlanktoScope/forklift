package cli

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
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

func PrintRequiredPallets(indent int, pallet *forklift.FSPallet) error {
	loadedPallets, err := pallet.LoadFSPalletReqs("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't identify pallets")
	}
	sort.Slice(loadedPallets, func(i, j int) bool {
		return forklift.CompareGitRepoReqs(
			loadedPallets[i].PalletReq.GitRepoReq, loadedPallets[j].PalletReq.GitRepoReq,
		) < 0
	})
	for _, pallet := range loadedPallets {
		IndentedPrintf(indent, "%s\n", pallet.Path())
	}
	return nil
}

func PrintRequiredPalletInfo(
	indent int, pallet *forklift.FSPallet, cache forklift.PathedPalletCache,
	requiredPalletPath string,
) error {
	req, err := pallet.LoadFSPalletReq(requiredPalletPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load pallet version lock definition %s from pallet %s",
			requiredPalletPath, pallet.FS.Path(),
		)
	}
	printPalletReq(indent, req.PalletReq)
	indent++

	version := req.VersionLock.Version
	cachedPallet, err := cache.LoadFSPallet(requiredPalletPath, version)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find pallet %s@%s in cache, please update the local cache of pallets",
			requiredPalletPath, version,
		)
	}
	// TODO: replace this with a call to PrintCachedPallet
	IndentedPrintf(indent, "Forklift version: %s\n", cachedPallet.Def.ForkliftVersion)
	fmt.Println()

	if core.CoversPath(cache, cachedPallet.FS.Path()) {
		IndentedPrintf(
			indent, "Path in cache: %s\n", core.GetSubdirPath(cache, cachedPallet.FS.Path()),
		)
	} else {
		IndentedPrintf(indent, "Absolute path (replacing any cached copy): %s\n", cachedPallet.FS.Path())
	}
	IndentedPrintf(indent, "Description: %s\n", cachedPallet.Def.Pallet.Description)

	readme, err := cachedPallet.LoadReadme()
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load readme file for pallet %s@%s from cache", requiredPalletPath, version,
		)
	}
	IndentedPrintln(indent, "Readme:")
	const widthLimit = 100
	PrintReadme(indent+1, readme, widthLimit)
	return nil
}

func printPalletReq(indent int, req forklift.PalletReq) {
	IndentedPrintf(indent, "Pallet: %s\n", req.Path())
	indent++
	IndentedPrintf(indent, "Locked pallet version: %s\n", req.VersionLock.Version)
}

// Add

func AddPalletReqs(
	indent int, pallet *forklift.FSPallet, cachePath string, palletQueries []string,
) error {
	if err := validateGitRepoQueries(palletQueries); err != nil {
		return errors.Wrap(err, "one or more pallet queries is invalid")
	}
	resolved, err := ResolveQueriesUsingLocalMirrors(0, cachePath, palletQueries, true)
	if err != nil {
		return err
	}
	fmt.Println()
	fmt.Printf("Saving configurations to %s...\n", pallet.FS.Path())
	for _, palletQuery := range palletQueries {
		req, ok := resolved[palletQuery]
		if !ok {
			return errors.Errorf("couldn't find configuration for %s", palletQuery)
		}
		reqsPalletsFS, err := pallet.GetPalletReqsFS()
		if err != nil {
			return err
		}
		palletReqPath := path.Join(reqsPalletsFS.Path(), req.Path(), forklift.VersionLockDefFile)
		if err = writeVersionLock(req.VersionLock, palletReqPath); err != nil {
			return errors.Wrapf(err, "couldn't write version lock for pallet requirement")
		}
	}
	return nil
}

// Remove

func RemovePalletReqs(
	indent int, pallet *forklift.FSPallet, palletPaths []string, force bool,
) error {
	usedPalletReqs, err := determineUsedPalletReqs(indent, pallet, force)
	if err != nil {
		return errors.Wrap(
			err,
			"couldn't determine pallets have declared file imports, to check which pallets to remove "+
				"still have declared file imports; to skip this check, enable the --force flag",
		)
	}

	fmt.Printf("Removing requirements from %s...\n", pallet.FS.Path())
	for _, palletPath := range palletPaths {
		if actualPalletPath, _, ok := strings.Cut(palletPath, "@"); ok {
			IndentedPrintf(
				indent,
				"Warning: provided pallet path %s is actually a pallet query; removing %s instead...\n",
				palletPath, actualPalletPath,
			)
			palletPath = actualPalletPath
		}
		reqsPalletsFS, err := pallet.GetPalletReqsFS()
		if err != nil {
			return err
		}
		palletReqPath := path.Join(reqsPalletsFS.Path(), palletPath)
		if !force && len(usedPalletReqs[palletPath]) > 0 {
			return errors.Errorf(
				"couldn't remove requirement for pallet %s because it's needed by file imports %+v; to "+
					"skip this check, enable the --force flag",
				palletPath, usedPalletReqs[palletPath],
			)
		}
		if err = os.RemoveAll(palletReqPath); err != nil {
			return errors.Wrapf(
				err, "couldn't remove requirement for pallet %s, at %s", palletPath, palletReqPath,
			)
		}
	}
	// TODO: maybe it'd be better to remove everything we can remove and then report errors at the
	// end?
	return nil
}

func determineUsedPalletReqs(
	indent int, pallet *forklift.FSPallet, force bool,
) (map[string][]string, error) {
	// FIXME: implement this by checking which pallet requirements have import files in them
	IndentedPrintln(
		indent,
		"Warning: we have not yet implemented a check for whether a pallet requirement has "+
			"any attached file imports!", pallet.Path(), force,
	)
	return nil, nil
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
