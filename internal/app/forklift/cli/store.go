package cli

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/pkg/core"
)

func GetStageStore(
	workspace *forklift.FSWorkspace, stageStorePath, newStageStoreVersion string,
) (*forklift.FSStageStore, error) {
	if stageStorePath == "" {
		return workspace.GetStageStore(newStageStoreVersion)
	}

	fsys := core.AttachPath(os.DirFS(stageStorePath), stageStorePath)
	if err := forklift.EnsureFSStageStore(fsys, ".", newStageStoreVersion); err != nil {
		return nil, err
	}
	return forklift.LoadFSStageStore(fsys, ".")
}

func SetNextStagedBundle(
	store *forklift.FSStageStore, index int, exportPath, toolVersion, bundleMinVersion string,
	skipImageCaching, parallel, ignoreToolVersion bool,
) error {
	store.SetNext(index)
	fmt.Printf(
		"Committing update to the stage store for stage %d as the next stage to be applied...\n", index,
	)
	if err := store.CommitState(); err != nil {
		return errors.Wrap(err, "couldn't commit updated stage store state")
	}

	if skipImageCaching {
		return nil
	}

	if err := DownloadImagesForStoreApply(
		store, toolVersion, bundleMinVersion, parallel, ignoreToolVersion,
	); err != nil {
		return errors.Wrap(err, "couldn't cache Docker container images required by staged pallet")
	}
	return nil
}
