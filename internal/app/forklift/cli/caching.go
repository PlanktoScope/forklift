package cli

import (
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
)

func CacheAllReqs(
	indent int, pallet *forklift.FSPallet, palletCache forklift.PathedPalletCache,
	repoCache forklift.PathedRepoCache, dlCache *forklift.FSDownloadCache,
	includeDisabled, parallel bool,
) error {
	pallet, repoCacheWithMerged, err := CacheStagingReqs(
		indent, pallet, palletCache, repoCache, dlCache, includeDisabled, parallel,
	)
	if err != nil {
		return err
	}

	IndentedPrintln(indent, "Downloading Docker container images to be deployed by the local pallet...")
	if err := DownloadImages(1, pallet, repoCacheWithMerged, includeDisabled, parallel); err != nil {
		return err
	}
	return nil
}

func CacheStagingReqs(
	indent int, pallet *forklift.FSPallet, palletCache forklift.PathedPalletCache,
	repoCache forklift.PathedRepoCache, dlCache *forklift.FSDownloadCache,
	includeDisabled, parallel bool,
) (merged *forklift.FSPallet, repoCacheWithMerged *forklift.LayeredRepoCache, err error) {
	IndentedPrintln(indent, "Caching everything needed to stage the pallet...")
	indent++

	if _, err = DownloadRequiredPallets(indent, pallet, palletCache); err != nil {
		return nil, nil, err
	}

	if merged, err = forklift.MergeFSPallet(pallet, palletCache, nil); err != nil {
		return nil, nil, errors.Wrap(
			err, "couldn't merge pallet with file imports from any pallets required by it",
		)
	}

	repoCacheWithMerged = &forklift.LayeredRepoCache{
		Underlay: repoCache,
	}
	if repoCacheWithMerged.Overlay, err = makeRepoOverrideCacheFromPallet(
		merged, true,
	); err != nil {
		return merged, nil, err
	}

	if _, err = DownloadRequiredRepos(indent, merged, repoCache.Path()); err != nil {
		return merged, repoCacheWithMerged, err
	}

	if err = DownloadExportFiles(
		indent, merged, repoCacheWithMerged, dlCache, includeDisabled, parallel,
	); err != nil {
		return merged, repoCacheWithMerged, err
	}

	// TODO: warn if any downloaded repo doesn't appear to be an actual repo, or if any repo's
	// forklift version is incompatible or ahead of the pallet version

	return merged, repoCacheWithMerged, nil
}
