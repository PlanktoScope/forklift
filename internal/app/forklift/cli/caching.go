package cli

import (
	"github.com/PlanktoScope/forklift/internal/app/forklift"
)

func CacheAllReqs(
	indent int, pallet *forklift.FSPallet, repoCachePath, palletCachePath string,
	pkgLoader forklift.FSPkgLoader, dlCache *forklift.FSDownloadCache,
	includeDisabled, parallel bool,
) error {
	if err := CacheStagingReqs(
		indent, pallet, repoCachePath, palletCachePath, pkgLoader, dlCache, includeDisabled, parallel,
	); err != nil {
		return err
	}

	IndentedPrintln(indent, "Downloading Docker container images to be deployed by the local pallet...")
	if err := DownloadImages(1, pallet, pkgLoader, includeDisabled, parallel); err != nil {
		return err
	}
	return nil
}

func CacheStagingReqs(
	indent int, pallet *forklift.FSPallet, repoCachePath, palletCachePath string,
	pkgLoader forklift.FSPkgLoader, dlCache *forklift.FSDownloadCache,
	includeDisabled, parallel bool,
) error {
	IndentedPrintln(indent, "Caching everything needed to stage the pallet...")
	indent++

	IndentedPrintln(indent, "Downloading pallets required by the local pallet...")
	if _, err := DownloadRequiredPallets(0, pallet, palletCachePath); err != nil {
		return err
	}

	// FIXME: merge the pallets into before downloading required repos

	IndentedPrintln(indent, "Downloading repos required by the local pallet...")
	if _, err := DownloadRequiredRepos(0, pallet, repoCachePath); err != nil {
		return err
	}

	IndentedPrintln(indent, "Downloading files for export by the local pallet...")
	if err := DownloadExportFiles(
		1, pallet, pkgLoader, dlCache, includeDisabled, parallel,
	); err != nil {
		return err
	}

	// TODO: warn if any downloaded repo doesn't appear to be an actual repo, or if any repo's
	// forklift version is incompatible or ahead of the pallet version

	return nil
}
