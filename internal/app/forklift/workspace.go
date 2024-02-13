package forklift

import (
	"os"
	"path"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/core"
)

func Exists(dirPath string) bool {
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
	if !Exists(dirPath) {
		return nil, errors.Errorf("couldn't find workspace at %s", dirPath)
	}
	return &FSWorkspace{
		FS: core.AttachPath(os.DirFS(dirPath), dirPath),
	}, nil
}

func (w *FSWorkspace) GetCurrentPalletPath() string {
	return path.Join(w.getDataPath(), currentPalletDirName)
}

func (w *FSWorkspace) getDataPath() string {
	return path.Join(w.FS.Path(), dataDirPath)
}

func (w *FSWorkspace) GetCurrentPallet() (*FSPallet, error) {
	if err := EnsureExists(w.getDataPath()); err != nil {
		return nil, errors.Wrapf(err, "couldn't ensure the existence of %s", w.getDataPath())
	}
	return LoadFSPallet(w.FS, path.Join(dataDirPath, currentPalletDirName))
}

func (w *FSWorkspace) GetRepoCachePath() string {
	return path.Join(w.getCachePath(), cacheReposDirName)
}

func (w *FSWorkspace) GetPalletCachePath() string {
	return path.Join(w.getCachePath(), cachePalletsDirName)
}

func (w *FSWorkspace) getCachePath() string {
	return path.Join(w.FS.Path(), cacheDirPath)
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

func (w *FSWorkspace) getCacheFS() (core.PathedFS, error) {
	if err := EnsureExists(w.getCachePath()); err != nil {
		return nil, errors.Wrapf(err, "couldn't ensure the existence of %s", w.getCachePath())
	}

	fsys, err := w.FS.Sub(cacheDirPath)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get cache from workspace")
	}
	return fsys, nil
}
