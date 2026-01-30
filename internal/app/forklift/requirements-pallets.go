package forklift

import (
	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/pkg/caching"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
	fws "github.com/forklift-run/forklift/pkg/workspaces"
)

func GetPalletCache(
	wpath string, pallet *fplt.FSPallet, requireCache bool,
) (*caching.FSPalletCache, error) {
	workspace, err := fws.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	cache, err := workspace.GetPalletCache()
	if err != nil {
		return nil, err
	}

	if requireCache && !cache.Exists() && pallet != nil {
		palletReqs, err := pallet.LoadFSPalletReqs("**")
		if err != nil {
			return nil, errors.Wrap(err, "couldn't check whether the pallet requires any pallets")
		}
		if len(palletReqs) > 0 {
			return nil, errors.New("you first need to cache the pallets specified by your pallet")
		}
	}
	return cache, nil
}

func GetRequiredPallet(
	pallet *fplt.FSPallet, cache caching.PathedPalletCache, requiredPalletPath string,
) (*fplt.FSPallet, error) {
	req, err := pallet.LoadFSPalletReq(requiredPalletPath)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load pallet version lock definition %s from pallet %s",
			requiredPalletPath, pallet.Path(),
		)
	}
	version := req.VersionLock.Version
	cachedPallet, err := cache.LoadFSPallet(requiredPalletPath, version)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't find pallet %s@%s in cache, please update the local cache of pallets",
			requiredPalletPath, version,
		)
	}
	mergedPallet, err := fplt.MergeFSPallet(cachedPallet, cache, nil)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't merge pallet %s with file imports from any pallets required by it",
			cachedPallet.Path(),
		)
	}
	return mergedPallet, nil
}
