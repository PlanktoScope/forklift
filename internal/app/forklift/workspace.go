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
	return path.Join(w.FS.Path(), currentPalletDirName)
}

func (w *FSWorkspace) GetCurrentPallet() (*FSPallet, error) {
	return LoadFSPallet(w.FS, currentPalletDirName)
}

func (w *FSWorkspace) getCachePath() string {
	return path.Join(w.FS.Path(), cacheDirName)
}

func (w *FSWorkspace) getCacheFS() (core.PathedFS, error) {
	fsys, err := w.FS.Sub(cacheDirName)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get cache from workspace")
	}
	return fsys, nil
}

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
