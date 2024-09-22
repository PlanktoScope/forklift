package cli

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	dct "github.com/compose-spec/compose-go/v2/types"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/clients/cli"
	"github.com/PlanktoScope/forklift/internal/clients/docker"
	"github.com/PlanktoScope/forklift/internal/clients/git"
	"github.com/PlanktoScope/forklift/pkg/core"
	"github.com/PlanktoScope/forklift/pkg/structures"
)

func GetStageStore(
	workspace *forklift.FSWorkspace, stageStorePath, newStageStoreVersion string,
) (*forklift.FSStageStore, error) {
	if stageStorePath == "" {
		return workspace.GetStageStore(newStageStoreVersion)
	}

	fsys := forklift.DirFS(stageStorePath)
	if err := forklift.EnsureFSStageStore(fsys, ".", newStageStoreVersion); err != nil {
		return nil, err
	}
	return forklift.LoadFSStageStore(fsys, ".")
}

func SetNextStagedBundle(
	indent int, store *forklift.FSStageStore, index int, exportPath,
	toolVersion, bundleMinVersion string, skipImageCaching, parallel, ignoreToolVersion bool,
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
		indent, store, toolVersion, bundleMinVersion, parallel, ignoreToolVersion,
	); err != nil {
		return errors.Wrap(err, "couldn't cache Docker container images required by staged pallet")
	}
	return nil
}

// Stage

type StagingVersions struct {
	Core               Versions
	MinSupportedBundle string
	NewBundle          string
}

type StagingCaches struct {
	Mirrors   core.Pather
	Pallets   forklift.PathedPalletCache
	Repos     forklift.PathedRepoCache
	Downloads *forklift.FSDownloadCache
}

func StagePallet(
	indent int, pallet *forklift.FSPallet, stageStore *forklift.FSStageStore, caches StagingCaches,
	exportPath string, versions StagingVersions,
	skipImageCaching, parallel, ignoreToolVersion bool,
) (index int, err error) {
	if _, isMerged := pallet.FS.(*forklift.MergeFS); isMerged {
		return 0, errors.Errorf("the pallet provided for staging should not be a merged pallet!")
	}

	pallet, repoCacheWithMerged, err := CacheStagingReqs(
		0, pallet, caches.Mirrors, caches.Pallets, caches.Repos, caches.Downloads, false, parallel,
	)
	if err != nil {
		return 0, errors.Wrap(err, "couldn't cache requirements for staging the pallet")
	}
	// Note: we must have all requirements in the cache before we can check their compatibility with
	// the Forklift tool version
	if err = CheckDeepCompat(
		pallet, caches.Pallets, repoCacheWithMerged, versions.Core, ignoreToolVersion,
	); err != nil {
		return 0, err
	}
	fmt.Println()

	index, err = stageStore.AllocateNew()
	if err != nil {
		return 0, errors.Wrap(err, "couldn't allocate a directory for staging")
	}
	fmt.Printf("Bundling pallet as stage %d for staged application...\n", index)
	if err = buildBundle(
		pallet, caches.Pallets, repoCacheWithMerged, caches.Downloads,
		versions.NewBundle, path.Join(stageStore.FS.Path(), fmt.Sprintf("%d", index)),
	); err != nil {
		return index, errors.Wrapf(err, "couldn't bundle pallet %s as stage %d", pallet.Path(), index)
	}
	if err = SetNextStagedBundle(
		indent, stageStore, index, exportPath, versions.Core.Tool, versions.MinSupportedBundle,
		skipImageCaching, parallel, ignoreToolVersion,
	); err != nil {
		return index, errors.Wrapf(
			err, "couldn't prepare staged pallet bundle %d to be applied next", index,
		)
	}
	return index, nil
}

func buildBundle(
	pallet *forklift.FSPallet,
	palletCache forklift.PathedPalletCache, repoCache forklift.PathedRepoCache,
	dlCache *forklift.FSDownloadCache,
	forkliftVersion, outputPath string,
) (err error) {
	outputBundle := forklift.NewFSBundle(outputPath)
	outputBundle.Manifest, err = newBundleManifest(pallet, palletCache, repoCache, forkliftVersion)
	if err != nil {
		return errors.Wrapf(err, "couldn't create bundle manifest for %s", outputBundle.FS.Path())
	}

	depls, _, err := Check(0, pallet, repoCache)
	if err != nil {
		return errors.Wrap(err, "couldn't ensure pallet validity")
	}
	for _, depl := range depls {
		if err := outputBundle.AddResolvedDepl(depl); err != nil {
			return errors.Wrapf(err, "couldn't add deployment %s to bundle", depl.Name)
		}
	}

	if err := outputBundle.SetBundledPallet(pallet); err != nil {
		return errors.Wrapf(err, "couldn't write pallet %s into bundle", pallet.Def.Pallet.Path)
	}
	if err = outputBundle.WriteRepoDefFile(); err != nil {
		return errors.Wrap(err, "couldn't write repo declaration into bundle")
	}
	if err = outputBundle.WriteFileExports(dlCache); err != nil {
		return errors.Wrap(err, "couldn't write file exports into bundle")
	}
	if err = outputBundle.WriteManifestFile(); err != nil {
		return errors.Wrap(err, "couldn't write bundle manifest file into bundle")
	}
	return nil
}

func newBundleManifest(
	pallet *forklift.FSPallet,
	palletCache forklift.PathedPalletCache, repoCache forklift.PathedRepoCache,
	forkliftVersion string,
) (forklift.BundleManifest, error) {
	desc := forklift.BundleManifest{
		ForkliftVersion: forkliftVersion,
		Pallet: forklift.BundlePallet{
			Path:        pallet.Path(),
			Description: pallet.Def.Pallet.Description,
		},
		Includes: forklift.BundleInclusions{
			Pallets: make(map[string]forklift.BundlePalletInclusion),
			Repos:   make(map[string]forklift.BundleRepoInclusion),
		},
		Deploys:   make(map[string]forklift.DeplDef),
		Downloads: make(map[string][]string),
		Exports:   make(map[string][]string),
	}
	desc.Pallet.Version, desc.Pallet.Clean = checkGitRepoVersion(pallet.FS.Path())
	palletReqs, err := pallet.LoadFSPalletReqs("**")
	if err != nil {
		return desc, errors.Wrapf(err, "couldn't determine pallets required by pallet %s", pallet.Path())
	}
	for _, req := range palletReqs {
		desc.Includes.Pallets[req.RequiredPath] = newBundlePalletInclusion(req, palletCache)
	}
	repoReqs, err := pallet.LoadFSRepoReqs("**")
	if err != nil {
		return desc, errors.Wrapf(err, "couldn't determine repos required by pallet %s", pallet.Path())
	}
	for _, req := range repoReqs {
		desc.Includes.Repos[req.RequiredPath] = newBundleRepoInclusion(req, repoCache)
	}
	return desc, nil
}

func checkGitRepoVersion(palletPath string) (version string, clean bool) {
	gitRepo, err := git.Open(filepath.FromSlash(palletPath))
	if err != nil {
		return "", false
	}
	commit, err := gitRepo.GetHead()
	if err != nil {
		return "", false
	}
	versionLock, err := lockCommit(gitRepo, commit)
	if err != nil {
		return "", false
	}
	versionString, err := versionLock.Version()
	if err != nil {
		return "", false
	}
	status, err := gitRepo.Status()
	if err != nil {
		return versionString, false
	}
	return versionString, status.IsClean()
}

func newBundlePalletInclusion(
	req *forklift.FSPalletReq, palletCache forklift.PathedPalletCache,
) forklift.BundlePalletInclusion {
	inclusion := forklift.BundlePalletInclusion{Req: req.PalletReq}
	for {
		if palletCache == nil {
			return inclusion
		}
		layeredCache, ok := palletCache.(*forklift.LayeredPalletCache)
		if !ok {
			return inclusion
		}
		overlay := layeredCache.Overlay
		if overlay == nil {
			palletCache = layeredCache.Underlay
			continue
		}

		if repo, err := overlay.LoadFSPallet(
			req.RequiredPath, req.VersionLock.Version,
		); err == nil { // i.e. the repo was overridden
			inclusion.Override.Path = repo.FS.Path()
			inclusion.Override.Version, inclusion.Override.Clean = checkGitRepoVersion(repo.FS.Path())
			return inclusion
		}
		palletCache = layeredCache.Underlay
	}
}

func newBundleRepoInclusion(
	req *forklift.FSRepoReq, repoCache forklift.PathedRepoCache,
) forklift.BundleRepoInclusion {
	inclusion := forklift.BundleRepoInclusion{Req: req.RepoReq}
	for {
		if repoCache == nil {
			return inclusion
		}
		layeredCache, ok := repoCache.(*forklift.LayeredRepoCache)
		if !ok {
			return inclusion
		}
		overlay := layeredCache.Overlay
		if overlay == nil {
			repoCache = layeredCache.Underlay
			continue
		}

		if repo, err := overlay.LoadFSRepo(
			req.RequiredPath, req.VersionLock.Version,
		); err == nil { // i.e. the repo was overridden
			inclusion.Override.Path = repo.FS.Path()
			inclusion.Override.Version, inclusion.Override.Clean = checkGitRepoVersion(repo.FS.Path())
			return inclusion
		}
		repoCache = layeredCache.Underlay
	}
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
	const dockerIndent = 2 // docker's indentation is flaky, so we indent extra
	dc, err := docker.NewClient(
		docker.WithOutputStream(cli.NewIndentedWriter(indent+dockerIndent, os.Stdout)),
		docker.WithErrorStream(cli.NewIndentedWriter(indent+dockerIndent, os.Stderr)),
	)
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}

	fmt.Println()
	fmt.Println("Applying changes serially...")
	indent++
	for _, change := range plan {
		fmt.Println()
		if err := applyReconciliationChange(context.Background(), indent, change, dc); err != nil {
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
	IndentedPrintln(indent, "Applying changes concurrently...")
	indent++

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
