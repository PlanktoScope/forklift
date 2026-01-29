package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/forklift-run/forklift/internal/app/forklift"
	"github.com/forklift-run/forklift/internal/clients/cli"
	"github.com/forklift-run/forklift/internal/clients/docker"
	"github.com/forklift-run/forklift/internal/clients/git"
	fbun "github.com/forklift-run/forklift/pkg/bundling"
	"github.com/forklift-run/forklift/pkg/caching"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
	"github.com/forklift-run/forklift/pkg/structures"
)

func GetStageStore(
	workspace *forklift.FSWorkspace, stageStorePath, newStageStoreVersion string,
) (*forklift.FSStageStore, error) {
	if stageStorePath == "" {
		return workspace.GetStageStore(newStageStoreVersion)
	}

	fsys := ffs.DirFS(stageStorePath)
	if err := forklift.EnsureFSStageStore(fsys, ".", newStageStoreVersion); err != nil {
		return nil, err
	}
	return forklift.LoadFSStageStore(fsys, ".")
}

func SetNextStagedBundle(
	indent int, store *forklift.FSStageStore, index int, exportPath,
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
	Core               Versions
	MinSupportedBundle string
	NewBundle          string
}

type StagingCaches struct {
	Mirrors   ffs.Pather
	Pallets   caching.PathedPalletCache
	Downloads *caching.FSDownloadCache
}

func StagePallet(
	indent int, merged *fplt.FSPallet, stageStore *forklift.FSStageStore, caches StagingCaches,
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

	index, err = stageStore.AllocateNew()
	if err != nil {
		return 0, errors.Wrap(err, "couldn't allocate a directory for staging")
	}
	IndentedFprintf(
		indent, os.Stderr, "Bundling pallet as stage %d for staged application...\n", index,
	)
	if err = buildBundle(
		merged, palletCacheWithMerged, caches.Downloads,
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

func buildBundle(
	merged *fplt.FSPallet,
	palletCache caching.PathedPalletCache,
	dlCache *caching.FSDownloadCache,
	forkliftVersion, outputPath string,
) (err error) {
	outputBundle, err := fbun.NewFSBundle(outputPath)
	if err != nil {
		return errors.Errorf("couldn't initialize new bundle at %s", outputPath)
	}
	outputBundle.Manifest, err = newBundleManifest(merged, palletCache, forkliftVersion)
	if err != nil {
		return errors.Wrapf(err, "couldn't create bundle manifest for %s", outputBundle.FS.Path())
	}

	overlayCache, err := MakeOverlayCache(merged, palletCache)
	if err != nil {
		return err
	}
	depls, _, err := Check(0, merged, overlayCache)
	if err != nil {
		return errors.Wrap(err, "couldn't ensure pallet validity")
	}
	for _, depl := range depls {
		if err := outputBundle.AddResolvedDepl(depl); err != nil {
			return errors.Wrapf(err, "couldn't add deployment %s to bundle", depl.Name)
		}
	}

	if err := outputBundle.SetBundledPallet(merged); err != nil {
		return errors.Wrapf(err, "couldn't write pallet %s into bundle", merged.Decl.Pallet.Path)
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
	merged *fplt.FSPallet, palletCache caching.PathedPalletCache, forkliftVersion string,
) (fbun.BundleManifest, error) {
	desc := fbun.BundleManifest{
		ForkliftVersion: forkliftVersion,
		Pallet: fbun.BundlePallet{
			Path:        merged.Path(),
			Description: merged.Decl.Pallet.Description,
		},
		Includes: fbun.BundleInclusions{
			Pallets: make(map[string]fbun.BundlePalletInclusion),
		},
		Deploys:   make(map[string]fplt.DeplDecl),
		Downloads: make(map[string]fbun.BundleDeplDownloads),
		Exports:   make(map[string]fbun.BundleDeplExports),
	}
	desc.Pallet.Version, desc.Pallet.Clean = CheckGitRepoVersion(merged.FS.Path())
	palletReqs, err := merged.LoadFSPalletReqs("**")
	if err != nil {
		return desc, errors.Wrapf(
			err, "couldn't determine pallets required by pallet %s", merged.Path(),
		)
	}
	for _, req := range palletReqs {
		if desc.Includes.Pallets[req.RequiredPath], err = newBundlePalletInclusion(
			merged, req, palletCache, true,
		); err != nil {
			return desc, errors.Wrapf(
				err, "couldn't generate description of requirement for pallet %s", req.RequiredPath,
			)
		}
	}
	if mergeFS, ok := merged.FS.(*ffs.MergeFS); ok {
		imports, err := mergeFS.ListImports()
		if err != nil {
			return desc, errors.Wrapf(err, "couldn't list pallet file import groups")
		}
		desc.Imports = make(map[string][]string)
		for target, sourceRef := range imports {
			sources := make([]string, 0, len(sourceRef.Sources))
			for _, source := range sourceRef.Sources {
				sources = append(sources, path.Join(source, sourceRef.Path))
			}
			desc.Imports[target] = sources
		}
	}
	return desc, nil
}

func CheckGitRepoVersion(palletPath string) (version string, clean bool) {
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
	pallet *fplt.FSPallet, req *fplt.FSPalletReq, palletCache caching.PathedPalletCache,
	describeImports bool,
) (inclusion fbun.BundlePalletInclusion, err error) {
	inclusion = fbun.BundlePalletInclusion{
		Req:      req.PalletReq,
		Includes: make(map[string]fbun.BundlePalletInclusion),
	}
	for {
		if palletCache == nil {
			break
		}
		layeredCache, ok := palletCache.(*caching.LayeredPalletCache)
		if !ok {
			break
		}
		overlay := layeredCache.Overlay
		if overlay == nil {
			palletCache = layeredCache.Underlay
			continue
		}

		if loaded, err := overlay.LoadFSPallet(req.RequiredPath, req.VersionLock.Version); err == nil {
			// i.e. the pallet was overridden
			inclusion.Override.Path = loaded.FS.Path()
			inclusion.Override.Version, inclusion.Override.Clean = CheckGitRepoVersion(loaded.FS.Path())
			break
		}
		palletCache = layeredCache.Underlay
	}

	loaded, err := palletCache.LoadFSPallet(req.RequiredPath, req.VersionLock.Version)
	if err != nil {
		return inclusion, errors.Wrapf(err, "couldn't load pallet %s", req.RequiredPath)
	}
	palletReqs, err := loaded.LoadFSPalletReqs("**")
	if err != nil {
		return inclusion, errors.Wrapf(
			err, "couldn't determine pallets required by pallet %s", loaded.Path(),
		)
	}
	for _, req := range palletReqs {
		if inclusion.Includes[req.RequiredPath], err = newBundlePalletInclusion(
			loaded, req, palletCache, false,
		); err != nil {
			return inclusion, errors.Wrapf(
				err, "couldn't generate description of transitive requirement for pallet %s", loaded.Path(),
			)
		}
	}

	if !describeImports {
		return inclusion, nil
	}
	if inclusion.Imports, err = describePalletImports(pallet, req, palletCache); err != nil {
		return inclusion, errors.Wrapf(err, "couldn't describe file imports for %s", req.RequiredPath)
	}
	return inclusion, nil
}

func describePalletImports(
	pallet *fplt.FSPallet, req *fplt.FSPalletReq, palletCache caching.PathedPalletCache,
) (fileMappings map[string]map[string]string, err error) {
	imports, err := pallet.LoadImports(path.Join(req.RequiredPath, "**/*"))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load file import groups")
	}
	allResolved, err := fplt.ResolveImports(pallet, palletCache, imports)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't resolve file import groups")
	}
	requiredPallets := make(map[string]*fplt.FSPallet) // pallet path -> pallet
	for _, resolved := range allResolved {
		requiredPallets[resolved.Pallet.Path()] = resolved.Pallet
	}
	for palletPath, requiredPallet := range requiredPallets {
		if requiredPallets[palletPath], err = fplt.MergeFSPallet(
			requiredPallet, palletCache, nil,
		); err != nil {
			return nil, errors.Wrapf(
				err, "couldn't compute merged pallet for required pallet %s", palletPath,
			)
		}
	}

	fileMappings = make(map[string]map[string]string)
	for _, resolved := range allResolved {
		resolved.Pallet = requiredPallets[req.RequiredPath]
		importName := strings.TrimPrefix(resolved.Name, req.RequiredPath+"/")
		if fileMappings[importName], err = resolved.Evaluate(palletCache); err != nil {
			return nil, errors.Wrapf(err, "couldn't evaluate file import group %s", importName)
		}
	}
	return fileMappings, nil
}

// Apply

func ApplyNextOrCurrentBundle(
	indent int, store *forklift.FSStageStore, bundle *fbun.FSBundle, parallel bool,
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

func applyChangesSerially(indent int, plan []*ReconciliationChange) error {
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
	ctx context.Context, indent int, change *ReconciliationChange, dc *docker.Client,
) error {
	switch change.Type {
	default:
		return errors.Errorf("unknown change type '%s'", change.Type)
	case addReconciliationChange:
		IndentedFprintf(
			indent, os.Stderr,
			"Adding package deployment %s as Compose app %s...\n", change.Depl.Name, change.Name,
		)
		if err := deployApp(ctx, indent, change.Depl, change.Name, dc); err != nil {
			return errors.Wrapf(err, "couldn't add %s", change.Name)
		}
		return nil
	case removeReconciliationChange:
		// Note: removeReconciliationChange has a nil Depl field
		IndentedFprintf(
			indent, os.Stderr, "Removing Compose app %s (unknown deployment)...\n", change.Name,
		)
		if err := dc.RemoveApps(ctx, []string{change.Name}); err != nil {
			return errors.Wrapf(err, "couldn't remove %s", change.Name)
		}
		return nil
	case updateReconciliationChange:
		IndentedFprintf(
			indent, os.Stderr, "Updating package deployment %s as Compose app %s...\n",
			change.Depl.Name, change.Name,
		)
		if err := deployApp(ctx, indent, change.Depl, change.Name, dc); err != nil {
			return errors.Wrapf(err, "couldn't add %s", change.Name)
		}
		return nil
	}
}

func deployApp(
	ctx context.Context, indent int, depl *fplt.ResolvedDepl, name string, dc *docker.Client,
) error {
	definesApp, err := depl.DefinesComposeApp()
	if err != nil {
		return errors.Wrapf(
			err, "couldn't determine whether package deployment %s defines a Compose app", depl.Name,
		)
	}
	if !definesApp {
		IndentedFprintln(indent, os.Stderr, "No Docker Compose app to deploy!")
		return nil
	}

	appDef, err := depl.LoadComposeAppDefinition(true)
	if err != nil {
		return errors.Wrap(err, "couldn't load Compose app definition")
	}
	if err = dc.DeployApp(ctx, appDef, 0); err != nil {
		return errors.Wrapf(err, "couldn't deploy Compose app '%s'", name)
	}
	return nil
}

func applyChangesConcurrently(indent int, plan structures.Digraph[*ReconciliationChange]) error {
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
