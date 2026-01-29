package forklift

import (
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/forklift-run/forklift/pkg/caching"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
	"github.com/forklift-run/forklift/pkg/versioning"
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

// Add

func WriteVersionLock(lock versioning.Lock, writePath string) error {
	marshaled, err := yaml.Marshal(lock.Decl)
	if err != nil {
		return errors.Wrapf(err, "couldn't marshal version lock")
	}
	parentDir := filepath.FromSlash(path.Dir(writePath))
	if err := ffs.EnsureExists(parentDir); err != nil {
		return errors.Wrapf(err, "couldn't make directory %s", parentDir)
	}
	const perm = 0o644 // owner rw, group r, public r
	if err := os.WriteFile(filepath.FromSlash(writePath), marshaled, perm); err != nil {
		return errors.Wrapf(err, "couldn't save version lock to %s", filepath.FromSlash(writePath))
	}
	return nil
}
