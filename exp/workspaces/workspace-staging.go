// Package workspaces implements storage and organization of Forklift-related data in the user's
// HOME directory.
package workspaces

import (
	"path"

	"github.com/forklift-run/forklift/exp/staging"
)

// in $HOME/.local/share/forklift:

const dataStageStoreDirName = "stages"

// FSWorkspace: Data: Staging

func (w *FSWorkspace) GetStageStorePath() string {
	return path.Join(w.GetDataPath(), dataStageStoreDirName)
}

// GetStageStore loads the workspace's stage store from the path, initializing a state file (which
// has the specified minimum supported Forklift tool version) if it does not already exist.
func (w *FSWorkspace) GetStageStore(newStateStoreVersion string) (*staging.FSStageStore, error) {
	fsys, err := w.getDataFS()
	if err != nil {
		return nil, err
	}
	if err = staging.EnsureFSStageStore(
		w.FS, path.Join(dataDirPath, dataStageStoreDirName), newStateStoreVersion,
	); err != nil {
		return nil, err
	}
	return staging.LoadFSStageStore(fsys, dataStageStoreDirName)
}
