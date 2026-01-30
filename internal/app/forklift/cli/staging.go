package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	fbun "github.com/forklift-run/forklift/exp/bundling"
	"github.com/forklift-run/forklift/exp/caching"
	ffs "github.com/forklift-run/forklift/exp/fs"
	fplt "github.com/forklift-run/forklift/exp/pallets"
	"github.com/forklift-run/forklift/exp/staging"
	"github.com/forklift-run/forklift/exp/structures"
	"github.com/forklift-run/forklift/internal/app/forklift"
	"github.com/forklift-run/forklift/internal/clients/cli"
	"github.com/forklift-run/forklift/internal/clients/docker"
)

func SetNextStagedBundle(
	indent int, store *staging.FSStageStore, index int, exportPath,
	toolVersion, bundleMinVersion string, skipImageCaching bool, platform string, parallel,
	ignoreToolVersion bool,
) error {
	store.SetNext(index)
	IndentedFprintf(
		indent, os.Stderr,
		"Committing update to the stage store for stage %d as the next stage to be applied...\n", index,
	)
	if err := store.CommitState(); err != nil {
		return errors.Wrap(err, "couldn't commit updated stage store state")
	}

	if skipImageCaching {
		return nil
	}

	if err := DownloadImagesForStoreApply(
		indent, store, platform, toolVersion, bundleMinVersion, parallel, ignoreToolVersion,
	); err != nil {
		return errors.Wrap(err, "couldn't cache Docker container images required by staged pallet")
	}
	return nil
}

// Stage

type StagingVersions struct {
	Core               forklift.Versions
	MinSupportedBundle string
	NewBundle          string
}

type StagingCaches struct {
	Mirrors   ffs.Pather
	Pallets   caching.PathedPalletCache
	Downloads *caching.FSDownloadCache
}

func StagePallet(
	indent int, merged *fplt.FSPallet, stageStore *staging.FSStageStore, caches StagingCaches,
	exportPath string, versions StagingVersions,
	skipImageCaching bool, platform string, parallel, ignoreToolVersion bool,
) (index int, err error) {
	if _, isMerged := merged.FS.(*ffs.MergeFS); isMerged {
		return 0, errors.Errorf("the pallet provided for staging should not be a merged pallet!")
	}

	merged, palletCacheWithMerged, err := CacheStagingReqs(
		0, merged, caches.Mirrors, caches.Pallets, caches.Downloads,
		platform, false, parallel,
	)
	if err != nil {
		return 0, errors.Wrap(err, "couldn't cache requirements for staging the pallet")
	}
	// Note: we must have all requirements in the cache before we can check their compatibility with
	// the Forklift tool version
	if err = CheckDeepCompat(merged, caches.Pallets, versions.Core, ignoreToolVersion); err != nil {
		return 0, err
	}
	fmt.Fprintln(os.Stderr)

	if _, _, err = Check(0, merged, palletCacheWithMerged); err != nil {
		return index, errors.Wrap(err, "couldn't ensure pallet validity")
	}

	index, err = stageStore.AllocateNew()
	if err != nil {
		return 0, errors.Wrap(err, "couldn't allocate a directory for staging")
	}
	IndentedFprintf(
		indent, os.Stderr, "Bundling pallet as stage %d for staged application...\n", index,
	)

	if err = forklift.BuildBundle(
		merged, caches.Pallets, caches.Downloads,
		versions.NewBundle, path.Join(stageStore.FS.Path(), fmt.Sprintf("%d", index)),
	); err != nil {
		return index, errors.Wrapf(err, "couldn't bundle pallet %s as stage %d", merged.Path(), index)
	}
	if err = SetNextStagedBundle(
		indent, stageStore, index, exportPath, versions.Core.Tool, versions.MinSupportedBundle,
		skipImageCaching, platform, parallel, ignoreToolVersion,
	); err != nil {
		return index, errors.Wrapf(
			err, "couldn't prepare staged pallet bundle %d to be applied next", index,
		)
	}
	return index, nil
}

// Apply

func ApplyNextOrCurrentBundle(
	indent int, store *staging.FSStageStore, bundle *fbun.FSBundle, parallel bool,
) error {
	applyingFallback := store.NextFailed()
	applyErr := applyBundle(0, bundle, parallel)
	current, _ := store.GetCurrent()
	next, _ := store.GetNext()
	fmt.Fprintln(os.Stderr)
	if !applyingFallback || current == next {
		store.RecordNextSuccess(applyErr == nil)
	}
	if applyErr != nil {
		if applyingFallback {
			IndentedFprintln(
				indent, os.Stderr,
				"Failed to apply the fallback pallet bundle, even though it was successfully applied "+
					"in the past! You may need to try resetting your host, with `forklift host rm`.",
			)
			return applyErr
		}
		if err := store.CommitState(); err != nil {
			IndentedFprintf(
				indent, os.Stderr,
				"Error: couldn't record failure of the next staged pallet bundle: %s\n", err.Error(),
			)
		}
		IndentedFprintln(
			indent, os.Stderr,
			"Failed to apply next staged bundle; if you run `forklift stage apply` again, it will "+
				"attempt to apply the last successfully-applied pallet bundle (if it exists) as a "+
				"fallback!",
		)
		return errors.Wrap(applyErr, "couldn't apply next staged bundle")
	}
	if err := store.CommitState(); err != nil {
		return errors.Wrap(err, "couldn't commit updated stage store state")
	}
	return nil
}

func applyBundle(indent int, bundle *fbun.FSBundle, parallel bool) error {
	concurrentPlan, serialPlan, err := Plan(indent, bundle, bundle, parallel)
	if err != nil {
		return err
	}

	if serialPlan != nil {
		return applyChangesSerially(indent, serialPlan)
	}
	return applyChangesConcurrently(indent, concurrentPlan)
}

func applyChangesSerially(indent int, plan []*forklift.ReconciliationChange) error {
	const dockerIndent = 2 // docker's indentation is flaky, so we indent extra
	dc, err := docker.NewClient(
		// we want to send all of Docker's log messages to stderr:
		docker.WithOutputStream(cli.NewIndentedWriter(indent+dockerIndent, os.Stderr)),
		docker.WithErrorStream(cli.NewIndentedWriter(indent+dockerIndent, os.Stderr)),
	)
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}

	fmt.Fprintln(os.Stderr)
	IndentedFprintln(indent, os.Stderr, os.Stderr, "Applying changes serially...")
	indent++
	for _, change := range plan {
		fmt.Fprintln(os.Stderr)
		if err := applyReconciliationChange(context.Background(), indent, change, dc); err != nil {
			return errors.Wrapf(err, "couldn't apply change '%s'", change.PlanString())
		}
	}
	return nil
}

func applyReconciliationChange(
	ctx context.Context, indent int, change *forklift.ReconciliationChange, dc *docker.Client,
) error {
	switch change.Type {
	default:
		return errors.Errorf("unknown change type '%s'", change.Type)
	case forklift.AddReconciliationChange:
		IndentedFprintf(
			indent, os.Stderr,
			"Adding package deployment %s as Compose app %s...\n", change.Depl.Name, change.Name,
		)
	case forklift.RemoveReconciliationChange:
		// Note: removeReconciliationChange has a nil Depl field
		IndentedFprintf(
			indent, os.Stderr, "Removing Compose app %s (unknown deployment)...\n", change.Name,
		)
	case forklift.UpdateReconciliationChange:
		IndentedFprintf(
			indent, os.Stderr, "Updating package deployment %s as Compose app %s...\n",
			change.Depl.Name, change.Name,
		)
	}
	return forklift.ApplyReconciliationChange(ctx, change, dc)
}

func applyChangesConcurrently(
	indent int, plan structures.Digraph[*forklift.ReconciliationChange],
) error {
	const dockerIndent = 2 // docker's indentation is flaky, so we indent extra
	dc, err := docker.NewClient(
		docker.WithConcurrencySafeOutput(),
		docker.WithOutputStream(cli.NewIndentedWriter(indent+dockerIndent, os.Stderr)),
		// Docker's usual stderr output looks weird with concurrency, so we discard it.
		// TODO: direct it to a concurrency-safe logger instead?
		docker.WithErrorStream(cli.NewIndentedWriter(indent+dockerIndent, io.Discard)),
	)
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}
	fmt.Fprintln(os.Stderr)
	IndentedFprintln(indent, os.Stderr, "Applying changes concurrently...")
	indent++

	changeDone := make(map[*forklift.ReconciliationChange]chan struct{})
	for change := range plan {
		changeDone[change] = make(chan struct{})
	}
	// We don't use the errgroup's context because we don't want one failing service to prevent
	// bringup of all other services.
	eg, _ := errgroup.WithContext(context.Background())
	for change, deps := range plan {
		eg.Go(func() error {
			defer close(changeDone[change])

			for dep := range deps {
				<-changeDone[dep]
			}
			if err := applyReconciliationChange(context.Background(), indent, change, dc); err != nil {
				return errors.Wrapf(err, "couldn't apply change '%s'", change.PlanString())
			}
			return nil
		})
	}
	return eg.Wait()
}
