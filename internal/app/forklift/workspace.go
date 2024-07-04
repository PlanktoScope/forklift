package forklift

import (
	"os"
	"path"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/core"
)

func FileExists(filePath string) bool {
	results, err := os.Stat(filePath)
	if err == nil && !results.IsDir() {
		return true
	}
	return false
}

func DirExists(dirPath string) bool {
	dir, err := os.Stat(dirPath)
	if err == nil && dir.IsDir() {
		return true
	}
	return false
}

func EnsureExists(dirPath string) error {
	const perm = 0o755 // owner rwx, group rx, public rx
	return os.MkdirAll(dirPath, perm)
}

// FSWorkspace

// LoadWorkspace loads the workspace at the specified path.
// The workspace is usually just a home directory, e.g. $HOME; directories in the workspace are
// organized with the same structure as the default structure described by the
// [XDG base directory spec](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html).
// The provided path must use the host OS's path separators.
func LoadWorkspace(dirPath string) (*FSWorkspace, error) {
	if !DirExists(dirPath) {
		return nil, errors.Errorf("couldn't find workspace at %s", dirPath)
	}
	return &FSWorkspace{
		FS: core.AttachPath(os.DirFS(dirPath), dirPath),
	}, nil
}

// Data

func (w *FSWorkspace) GetDataPath() string {
	return path.Join(w.FS.Path(), dataDirPath)
}

func (w *FSWorkspace) getDataFS() (core.PathedFS, error) {
	if err := EnsureExists(w.GetDataPath()); err != nil {
		return nil, errors.Wrapf(err, "couldn't ensure the existence of %s", w.GetDataPath())
	}

	fsys, err := w.FS.Sub(dataDirPath)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get data directory from workspace")
	}
	return fsys, nil
}

// Data: Current Pallet

func (w *FSWorkspace) GetCurrentPalletPath() string {
	return path.Join(w.GetDataPath(), dataCurrentPalletDirName)
}

func (w *FSWorkspace) GetCurrentPallet() (*FSPallet, error) {
	fsys, err := w.getDataFS()
	if err != nil {
		return nil, err
	}
	return LoadFSPallet(fsys, dataCurrentPalletDirName)
}

// Data: Stages (i.e. pallet bundles which have been staged to be applied)

func (w *FSWorkspace) GetStageStorePath() string {
	return path.Join(w.GetDataPath(), dataStageStoreDirName)
}

// GetStageStore loads the workspace's stage store from the path, initializing a state file (which
// has the specified minimum supported Forklift tool version) if it does not already exist.
func (w *FSWorkspace) GetStageStore(newStateStoreVersion string) (*FSStageStore, error) {
	fsys, err := w.getDataFS()
	if err != nil {
		return nil, err
	}
	if err = EnsureFSStageStore(
		w.FS, path.Join(dataDirPath, dataStageStoreDirName), newStateStoreVersion,
	); err != nil {
		return nil, err
	}
	return LoadFSStageStore(fsys, dataStageStoreDirName)
}

// Cache

func (w *FSWorkspace) getCachePath() string {
	return path.Join(w.FS.Path(), cacheDirPath)
}

func (w *FSWorkspace) getCacheFS() (core.PathedFS, error) {
	if err := EnsureExists(w.getCachePath()); err != nil {
		return nil, errors.Wrapf(err, "couldn't ensure the existence of %s", w.getCachePath())
	}

	fsys, err := w.FS.Sub(cacheDirPath)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get cache directory from workspace")
	}
	return fsys, nil
}

// Cache: Repos

func (w *FSWorkspace) GetRepoCachePath() string {
	return path.Join(w.getCachePath(), cacheReposDirName)
}

func (w *FSWorkspace) GetRepoCache() (*FSRepoCache, error) {
	fsys, err := w.getCacheFS()
	if err != nil {
		return nil, err
	}
	pathedFS, err := fsys.Sub(cacheReposDirName)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get repos cache from workspace")
	}
	return &FSRepoCache{
		FS: pathedFS,
	}, nil
}

// Cache: Pallets

func (w *FSWorkspace) GetPalletCachePath() string {
	return path.Join(w.getCachePath(), cachePalletsDirName)
}

func (w *FSWorkspace) GetPalletCache() (*FSPalletCache, error) {
	fsys, err := w.getCacheFS()
	if err != nil {
		return nil, err
	}
	pathedFS, err := fsys.Sub(cachePalletsDirName)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get pallets cache from workspace")
	}
	return &FSPalletCache{
		FS: pathedFS,
	}, nil
}

// Cache: Downloads

func (w *FSWorkspace) GetDownloadCachePath() string {
	return path.Join(w.getCachePath(), cacheDownloadsDirName)
}

func (w *FSWorkspace) GetDownloadCache() (*FSDownloadCache, error) {
	fsys, err := w.getCacheFS()
	if err != nil {
		return nil, err
	}
	pathedFS, err := fsys.Sub(cacheDownloadsDirName)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get downloads cache from workspace")
	}
	return &FSDownloadCache{
		FS: pathedFS,
	}, nil
}
