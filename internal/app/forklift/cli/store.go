package cli

import (
	"context"
	"fmt"
	"os"

	dct "github.com/compose-spec/compose-go/v2/types"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/clients/docker"
	"github.com/PlanktoScope/forklift/pkg/core"
	"github.com/PlanktoScope/forklift/pkg/structures"
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

// Apply

func ApplyNextOrCurrentBundle(
	indent int, store *forklift.FSStageStore, bundle *forklift.FSBundle, parallel bool,
) error {
	applyingFallback := store.NextFailed()
	applyErr := applyBundle(0, bundle, parallel)
	current, _ := store.GetCurrent()
	next, _ := store.GetNext()
	fmt.Println()
	if !applyingFallback || current == next {
		store.RecordNextSuccess(applyErr == nil)
	}
	if applyErr != nil {
		if applyingFallback {
			IndentedPrintln(
				indent,
				"Failed to apply the fallback pallet bundle, even though it was successfully applied "+
					"in the past! You may need to try resetting your host, with `forklift host rm`.",
			)
			return applyErr
		}
		if err := store.CommitState(); err != nil {
			IndentedPrintf(
				indent,
				"Error: couldn't record failure of the next staged pallet bundle: %s\n", err.Error(),
			)
		}
		IndentedPrintln(
			indent,
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

func applyBundle(indent int, bundle *forklift.FSBundle, parallel bool) error {
	concurrentPlan, serialPlan, err := Plan(indent, bundle, bundle, parallel)
	if err != nil {
		return err
	}

	if serialPlan != nil {
		return applyChangesSerially(indent, serialPlan)
	}
	return applyChangesConcurrently(indent, concurrentPlan)
}

func applyChangesSerially(indent int, plan []*ReconciliationChange) error {
	dc, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}

	fmt.Println()
	fmt.Println("Applying changes serially...")
	for _, change := range plan {
		fmt.Println()
		if err := applyReconciliationChange(context.Background(), indent+1, change, dc); err != nil {
			return errors.Wrapf(err, "couldn't apply change '%s'", change.PlanString())
		}
	}
	return nil
}

func applyReconciliationChange(
	ctx context.Context, indent int, change *ReconciliationChange, dc *docker.Client,
) error {
	switch change.Type {
	default:
		return errors.Errorf("unknown change type '%s'", change.Type)
	case addReconciliationChange:
		IndentedPrintf(
			indent, "Adding package deployment %s as Compose app %s...\n", change.Depl.Name, change.Name,
		)
		if err := deployApp(ctx, indent, change.Depl, change.Name, dc); err != nil {
			return errors.Wrapf(err, "couldn't add %s", change.Name)
		}
		return nil
	case removeReconciliationChange:
		// Note: removeReconciliationChange has a nil Depl field
		IndentedPrintf(indent, "Removing Compose app %s (unknown deployment)...\n", change.Name)
		if err := dc.RemoveApps(ctx, []string{change.Name}); err != nil {
			return errors.Wrapf(err, "couldn't remove %s", change.Name)
		}
		return nil
	case updateReconciliationChange:
		IndentedPrintf(
			indent, "Updating package deployment %s as Compose app %s...\n",
			change.Depl.Name, change.Name,
		)
		if err := deployApp(ctx, indent, change.Depl, change.Name, dc); err != nil {
			return errors.Wrapf(err, "couldn't add %s", change.Name)
		}
		return nil
	}
}

func deployApp(
	ctx context.Context, indent int, depl *forklift.ResolvedDepl, name string, dc *docker.Client,
) error {
	definesApp, err := depl.DefinesApp()
	if err != nil {
		return errors.Wrapf(
			err, "couldn't determine whether package deployment %s defines a Compose app", depl.Name,
		)
	}
	if !definesApp {
		IndentedPrintln(indent, "No Docker Compose app to deploy!")
		return nil
	}

	appDef, err := loadAppDefinition(depl)
	if err != nil {
		return errors.Wrap(err, "couldn't load Compose app definition")
	}
	if err = dc.DeployApp(ctx, appDef, 0); err != nil {
		return errors.Wrapf(err, "couldn't deploy Compose app '%s'", name)
	}
	return nil
}

func loadAppDefinition(depl *forklift.ResolvedDepl) (*dct.Project, error) {
	composeFiles, err := depl.GetComposeFilenames()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't determine Compose files for deployment")
	}

	appDef, err := docker.LoadAppDefinition(
		depl.Pkg.FS, getAppName(depl.Name), composeFiles, nil,
	)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load Docker Compose app definition for deployment %s of %s",
			depl.Name, depl.Pkg.FS.Path(),
		)
	}
	return appDef, nil
}

func applyChangesConcurrently(indent int, plan structures.Digraph[*ReconciliationChange]) error {
	dc, err := docker.NewClient(docker.WithConcurrencySafeOutput())
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}
	fmt.Println()
	fmt.Println("Applying changes concurrently...")
	changeDone := make(map[*ReconciliationChange]chan struct{})
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
			if err := applyReconciliationChange(
				context.Background(), indent, change, dc,
			); err != nil {
				return errors.Wrapf(err, "couldn't apply change '%s'", change.PlanString())
			}
			return nil
		})
	}
	return eg.Wait()
}
