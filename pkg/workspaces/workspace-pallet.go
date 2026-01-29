package workspaces

import (
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"

	ffs "github.com/forklift-run/forklift/pkg/fs"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
)

// in $HOME/.local/share/forklift:

const dataCurrentPalletDirName = "pallet"

// in $HOME/.config/forklift:

const (
	configCurrentPalletUpgradesFile     = "pallet-upgrades.yml"
	configCurrentPalletUpgradesSwapFile = "pallet-upgrades-swap.yml"
)

// FSWorkspace: Data: Current Pallet

func (w *FSWorkspace) GetCurrentPalletPath() string {
	return path.Join(w.GetDataPath(), dataCurrentPalletDirName)
}

func (w *FSWorkspace) GetCurrentPallet() (*fplt.FSPallet, error) {
	fsys, err := w.getDataFS()
	if err != nil {
		return nil, err
	}
	return fplt.LoadFSPallet(fsys, dataCurrentPalletDirName)
}

// Config

func (w *FSWorkspace) GetCurrentPalletUpgrades() (GitRepoQuery, error) {
	fsys, err := w.getConfigFS()
	if err != nil {
		return GitRepoQuery{}, err
	}
	return loadGitRepoQuery(fsys, configCurrentPalletUpgradesFile)
}

// CommitCurrentPalletUpgrades atomically updates the current pallet upgrades file.
// Warning: on non-Unix platforms, the update is not entirely atomic!
func (w *FSWorkspace) CommitCurrentPalletUpgrades(query GitRepoQuery) error {
	// TODO: we might want to be less sloppy about read locks vs. write locks in the future. After
	// successfully acquiring a write lock, then we could just overwrite the swap file.
	swapPath := path.Join(w.getConfigPath(), configCurrentPalletUpgradesSwapFile)
	if ffs.FileExists(filepath.FromSlash(swapPath)) {
		return errors.Errorf(
			"current pallet upgrades swap file %s already exists, so either another operation is "+
				"currently running or the previous operation failed or was interrupted before it could "+
				"finish; please ensure that no other operations are currently running and delete the swap "+
				"file before retrying",
			swapPath,
		)
	}
	if err := query.Write(swapPath); err != nil {
		return errors.Wrapf(err, "couldn't save current pallet upgrades to swap file %s", swapPath)
	}
	outputPath := path.Join(w.getConfigPath(), configCurrentPalletUpgradesFile)
	// Warning: on non-Unix platforms, os.Rename is not an atomic operation! So if the program dies
	// during the os.Rename call, we could end up breaking the state of the stage store.
	if err := os.Rename(filepath.FromSlash(swapPath), filepath.FromSlash(outputPath)); err != nil {
		return errors.Wrapf(
			err, "couldn't commit current pallet upgrades update from %s to %s", swapPath, outputPath,
		)
	}
	return nil
}
