package cli

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/internal/app/forklift"
	"github.com/forklift-run/forklift/pkg/caching"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
	"github.com/forklift-run/forklift/pkg/structures"
)

// Printing

func FprintRequiredPallets(indent int, out io.Writer, pallet *fplt.FSPallet) error {
	loadedPallets, err := pallet.LoadFSPalletReqs("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't identify pallets")
	}
	slices.SortFunc(loadedPallets, func(a, b *fplt.FSPalletReq) int {
		return fplt.CompareGitRepoReqs(a.PalletReq.GitRepoReq, b.PalletReq.GitRepoReq)
	})
	for _, pallet := range loadedPallets {
		IndentedFprintf(indent, out, "%s\n", pallet.Path())
	}
	return nil
}

func FprintRequiredPalletInfo(
	indent int, out io.Writer,
	pallet *fplt.FSPallet, cache caching.PathedPalletCache, requiredPalletPath string,
) error {
	req, err := pallet.LoadFSPalletReq(requiredPalletPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load pallet version lock definition %s from pallet %s",
			requiredPalletPath, pallet.FS.Path(),
		)
	}
	fprintPalletReq(indent, out, req.PalletReq)
	indent++

	version := req.VersionLock.Version
	cachedPallet, err := cache.LoadFSPallet(requiredPalletPath, version)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find pallet %s@%s in cache, please update the local cache of pallets",
			requiredPalletPath, version,
		)
	}
	// We must merge the required pallet to get an accurate list of its deployments & packages:
	mergedPallet, err := fplt.MergeFSPallet(cachedPallet, cache, nil)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't merge pallet %s with file imports from any pallets required by it",
			cachedPallet.Path(),
		)
	}
	return FprintCachedPallet(indent, out, cache, mergedPallet, false)
}

func fprintPalletReq(indent int, out io.Writer, req fplt.PalletReq) {
	IndentedFprintf(indent, out, "Pallet: %s\n", req.Path())
	indent++
	IndentedFprintf(indent, out, "Locked pallet version: %s\n", req.VersionLock.Version)
}

func FprintRequiredPalletVersion(
	indent int, out io.Writer,
	pallet *fplt.FSPallet, cache caching.PathedPalletCache, requiredPalletPath string,
) error {
	req, err := pallet.LoadFSPalletReq(requiredPalletPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load pallet version lock definition %s from pallet %s",
			requiredPalletPath, pallet.FS.Path(),
		)
	}
	IndentedFprintln(indent, out, req.VersionLock.Version)
	return nil
}

// Add

func AddPalletReqs(
	indent int, pallet *fplt.FSPallet, mirrorsPath string, palletQueries []string,
) error {
	if err := forklift.ValidateGitRepoQueries(palletQueries); err != nil {
		return errors.Wrap(err, "one or more pallet queries is invalid")
	}
	resolved, err := ResolveQueriesUsingLocalMirrors(0, mirrorsPath, palletQueries, true)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr)
	IndentedFprintf(indent, os.Stderr, "Saving configurations to %s...\n", pallet.FS.Path())
	for _, palletQuery := range palletQueries {
		req, ok := resolved[palletQuery]
		if !ok {
			return errors.Errorf("couldn't find configuration for %s", palletQuery)
		}
		if err = pallet.WriteFSPalletReq(req); err != nil {
			return err
		}
	}
	return nil
}

// Remove

func RemovePalletReqs(
	indent int, pallet *fplt.FSPallet, palletPaths []string, force bool,
) error {
	usedPalletReqs, err := determineUsedPalletReqs(indent, pallet, force)
	if err != nil {
		return errors.Wrap(
			err,
			"couldn't determine pallets have declared file imports, to check which pallets to remove "+
				"still have declared file imports; to skip this check, enable the --force flag",
		)
	}

	IndentedFprintf(indent, os.Stderr, "Removing requirements from %s...\n", pallet.FS.Path())
	for _, palletPath := range palletPaths {
		if actualPalletPath, _, ok := strings.Cut(palletPath, "@"); ok {
			IndentedFprintf(
				indent, os.Stderr,
				"Warning: provided pallet path %s is actually a pallet query; removing %s instead...\n",
				palletPath, actualPalletPath,
			)
			palletPath = actualPalletPath
		}
		if !force && len(usedPalletReqs[palletPath]) > 0 {
			return errors.Errorf(
				"couldn't remove requirement for pallet %s because it's needed by file imports %+v; to "+
					"skip this check, enable the --force flag",
				palletPath, usedPalletReqs[palletPath],
			)
		}
		if err = pallet.RemoveFSPalletReq(palletPath); err != nil {
			return err
		}
	}
	// TODO: maybe it'd be better to remove everything we can remove and then report errors at the
	// end?
	return nil
}

func determineUsedPalletReqs(
	indent int, pallet *fplt.FSPallet, force bool,
) (map[string][]string, error) {
	imports, err := pallet.LoadImports("**/*")
	if err != nil {
		err = errors.Wrap(err, "couldn't load import groups")
		if !force {
			return nil, err
		}
		IndentedFprintf(indent, os.Stderr, "Warning: %s\n", err.Error())
	}
	usedPalletReqs := make(map[string][]string)
	if len(imports) == 0 {
		return usedPalletReqs, nil
	}
	palletReqsFS, err := pallet.GetPalletReqsFS()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open directory for pallet requirements from pallet")
	}

	for _, imp := range imports {
		fsPalletReq, err := fplt.LoadFSPalletReqContaining(palletReqsFS, imp.Name)
		if err != nil {
			err = errors.Wrapf(
				err, "couldn't find pallet requirement needed for import group %s of package %s",
				imp.Name, imp.Name,
			)
			if !force {
				return nil, err
			}
			IndentedFprintf(indent, os.Stderr, "Warning: %s\n", err.Error())
		}
		usedPalletReqs[fsPalletReq.Path()] = append(usedPalletReqs[fsPalletReq.Path()], imp.Name)
	}
	return usedPalletReqs, nil
}

// Download

func DownloadAllRequiredPallets(
	indent int, pallet *fplt.FSPallet,
	mirrorsCache ffs.Pather, palletsCache caching.PathedPalletCache,
	skipPalletQueries structures.Set[string],
) (downloadedPallets structures.Set[string], err error) {
	loadedPalletReqs, err := pallet.LoadFSPalletReqs("**")
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't identify pallets")
	}
	if len(loadedPalletReqs) == 0 {
		return nil, nil
	}

	IndentedFprintln(indent, os.Stderr, "Downloading required pallets...")
	return downloadRequiredPallets(
		indent+1, loadedPalletReqs, mirrorsCache, palletsCache, skipPalletQueries,
	)
}

func downloadRequiredPallets(
	indent int, reqs []*fplt.FSPalletReq,
	mirrorsCache ffs.Pather, palletsCache caching.PathedPalletCache,
	skipPalletQueries structures.Set[string],
) (downloadedPallets structures.Set[string], err error) {
	allSkip := make(structures.Set[string])
	maps.Insert(allSkip, maps.All(skipPalletQueries))
	downloadedPallets = make(structures.Set[string])
	for _, req := range reqs {
		IndentedFprintf(indent, os.Stderr, "Caching required pallet %s...\n", req.GetQueryPath())
		palletIndent := indent + 1
		if !allSkip.Has(req.GetQueryPath()) {
			downloaded, err := DownloadLockedGitRepoUsingLocalMirror(
				palletIndent, mirrorsCache.Path(), palletsCache.Path(), req.Path(), req.VersionLock,
			)
			if downloaded {
				downloadedPallets.Add(req.GetQueryPath())
			}
			if err != nil {
				return downloadedPallets, errors.Wrapf(
					err, "couldn't download %s at commit %s", req.Path(), req.VersionLock.Decl.ShortCommit(),
				)
			}
		} else {
			IndentedFprintln(palletIndent, os.Stderr, "Skipped download of pallet!")
		}

		plt, err := palletsCache.LoadFSPallet(req.Path(), req.VersionLock.Version)
		if err != nil {
			return downloadedPallets, errors.Wrapf(
				err, "couldn't load downloaded pallet for %s to download its own required pallets",
				req.Path(),
			)
		}
		allSkip.Add(req.GetQueryPath())
		recurseDownloaded, err := DownloadAllRequiredPallets(
			palletIndent, plt, mirrorsCache, palletsCache, allSkip,
		)
		maps.Insert(downloadedPallets, maps.All(recurseDownloaded))
		maps.Insert(allSkip, maps.All(recurseDownloaded))
		if err != nil {
			return downloadedPallets, errors.Wrapf(
				err, "couldn't download pallets required by pallet %s", req.GetQueryPath(),
			)
		}
	}
	return downloadedPallets, nil
}
