package cli

import (
	"os"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/internal/app/forklift"
	ffs "github.com/forklift-run/forklift/pkg/fs"
)

func CacheAllReqs(
	indent int, pallet *forklift.FSPallet, mirrorsCache ffs.Pather,
	palletCache forklift.PathedPalletCache,
	dlCache *forklift.FSDownloadCache,
	platform string, includeDisabled, parallel bool,
) error {
	pallet, palletCacheWithMerged, err := CacheStagingReqs(
		indent, pallet, mirrorsCache, palletCache, dlCache,
		platform, includeDisabled, parallel,
	)
	if err != nil {
		return err
	}

	IndentedFprintln(
		indent, os.Stderr,
		"Downloading Docker container images to be deployed by the local pallet...",
	)
	if err := DownloadImages(
		1, pallet, palletCacheWithMerged, platform, includeDisabled, parallel,
	); err != nil {
		return err
	}
	return nil
}

func CacheStagingReqs(
	indent int, pallet *forklift.FSPallet, mirrorsCache ffs.Pather,
	palletCache forklift.PathedPalletCache,
	dlCache *forklift.FSDownloadCache,
	platform string, includeDisabled, parallel bool,
) (merged *forklift.FSPallet, palletCacheWithMerged *forklift.LayeredPalletCache, err error) {
	IndentedFprintln(indent, os.Stderr, "Caching everything needed to stage the pallet...")
	indent++

	if _, err = DownloadAllRequiredPallets(
		indent, pallet, mirrorsCache, palletCache, nil,
	); err != nil {
		return nil, nil, err
	}

	if merged, err = forklift.MergeFSPallet(pallet, palletCache, nil); err != nil {
		return nil, nil, errors.Wrap(
			err, "couldn't merge pallet with file imports from any pallets required by it",
		)
	}

	if palletCacheWithMerged, err = MakeOverlayCache(merged, palletCache); err != nil {
		return nil, nil, err
	}

	if err = DownloadExportFiles(
		indent, merged, palletCacheWithMerged, dlCache, platform, includeDisabled, parallel,
	); err != nil {
		return merged, palletCacheWithMerged, err
	}

	// TODO: warn if any downloaded pallet doesn't appear to be an actual pallet, or if any pallet's
	// forklift version is incompatible or ahead of the pallet version

	return merged, palletCacheWithMerged, nil
}
