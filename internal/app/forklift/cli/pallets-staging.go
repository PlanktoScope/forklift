package cli

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/clients/git"
)

type StagingVersions struct {
	Core               Versions
	MinSupportedBundle string
	NewBundle          string
}

type StagingCaches struct {
	Repos     forklift.PathedRepoCache
	Pallets   forklift.PathedPalletCache
	Downloads *forklift.FSDownloadCache
}

func StagePallet(
	pallet *forklift.FSPallet, stageStore *forklift.FSStageStore, caches StagingCaches,
	exportPath string, versions StagingVersions,
	skipImageCaching, parallel, ignoreToolVersion bool,
) (index int, err error) {
	if err = CacheStagingReqs(
		0, pallet, caches.Repos.Path(), caches.Pallets.Path(), caches.Repos, caches.Downloads,
		false, parallel,
	); err != nil {
		return 0, errors.Wrap(err, "couldn't cache requirements for staging the pallet")
	}
	// Note: we must have all requirements in the cache before we can check their compatibility with
	// the Forklift tool version
	if err = CheckDeepCompat(pallet, caches.Repos, versions.Core, ignoreToolVersion); err != nil {
		return 0, err
	}
	fmt.Println()

	index, err = stageStore.AllocateNew()
	if err != nil {
		return 0, errors.Wrap(err, "couldn't allocate a directory for staging")
	}
	fmt.Printf("Bundling pallet as stage %d for staged application...\n", index)
	if err = buildBundle(
		pallet, caches.Repos, caches.Downloads,
		versions.NewBundle, path.Join(stageStore.FS.Path(), fmt.Sprintf("%d", index)),
	); err != nil {
		return index, errors.Wrapf(err, "couldn't bundle pallet %s as stage %d", pallet.Path(), index)
	}
	if err = SetNextStagedBundle(
		stageStore, index, exportPath, versions.Core.Tool, versions.MinSupportedBundle,
		skipImageCaching, parallel, ignoreToolVersion,
	); err != nil {
		return index, errors.Wrapf(
			err, "couldn't prepare staged pallet bundle %d to be applied next", index,
		)
	}
	return index, nil
}

func buildBundle(
	pallet *forklift.FSPallet, repoCache forklift.PathedRepoCache, dlCache *forklift.FSDownloadCache,
	forkliftVersion, outputPath string,
) (err error) {
	outputBundle := forklift.NewFSBundle(outputPath)
	// TODO: once we can overlay pallets, save the result of overlaying the pallets to a `overlay`
	// subdir
	outputBundle.Manifest, err = newBundleManifest(pallet, repoCache, forkliftVersion)
	if err != nil {
		return errors.Wrapf(err, "couldn't create bundle manifest for %s", outputBundle.FS.Path())
	}

	depls, _, err := Check(0, pallet, repoCache)
	if err != nil {
		return errors.Wrap(err, "couldn't ensure pallet validity")
	}
	for _, depl := range depls {
		if err := outputBundle.AddResolvedDepl(depl); err != nil {
			return err
		}
	}

	if err := outputBundle.SetBundledPallet(pallet); err != nil {
		return err
	}
	if err = outputBundle.WriteRepoDefFile(); err != nil {
		return err
	}
	if err = outputBundle.WriteFileExports(dlCache); err != nil {
		return err
	}
	return outputBundle.WriteManifestFile()
}

func newBundleManifest(
	pallet *forklift.FSPallet, repoCache forklift.PathedRepoCache, forkliftVersion string,
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
	// TODO: once we can overlay pallets, the description of pallet & repo inclusions should probably
	// be made from the result of overlaying. We could also describe pre-overlay requirements from the
	// bundled pallet, in desc.Pallet.Requires.
	for _, req := range palletReqs {
		inclusion := forklift.BundlePalletInclusion{Req: req.PalletReq}
		// TODO: also check for overridden pallets
		desc.Includes.Pallets[req.RequiredPath] = inclusion
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
