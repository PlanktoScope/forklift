package cli

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
)

func SetNextStagedBundle(
	store *forklift.FSStageStore, index int, exportPath, toolVersion, bundleMinVersion string,
	parallel, ignoreToolVersion bool,
) error {
	store.SetNext(index)
	fmt.Printf(
		"Committing update to the stage store for stage %d as the next stage to be applied...\n", index,
	)
	if err := store.CommitState(); err != nil {
		return errors.Wrap(err, "couldn't commit updated stage store state")
	}

	if err := DownloadImagesForStoreApply(
		store, toolVersion, bundleMinVersion, parallel, ignoreToolVersion,
	); err != nil {
		return errors.Wrap(err, "couldn't cache Docker container images required by staged pallet")
	}
	return nil
}
