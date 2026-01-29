package forklift

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/internal/clients/git"
	fbun "github.com/forklift-run/forklift/pkg/bundling"
	"github.com/forklift-run/forklift/pkg/caching"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
	"github.com/forklift-run/forklift/pkg/staging"
	fws "github.com/forklift-run/forklift/pkg/workspaces"
)

func GetStageStore(
	workspace *fws.FSWorkspace, stageStorePath, newStageStoreVersion string,
) (*staging.FSStageStore, error) {
	if stageStorePath == "" {
		return workspace.GetStageStore(newStageStoreVersion)
	}

	fsys := ffs.DirFS(stageStorePath)
	if err := staging.EnsureFSStageStore(fsys, ".", newStageStoreVersion); err != nil {
		return nil, err
	}
	return staging.LoadFSStageStore(fsys, ".")
}

// Stage

func NewBundleManifest(
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
	versionLock, err := LockCommit(gitRepo, commit)
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
