package forklift

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

func Exists(path string) bool {
	dir, err := os.Stat(path)
	if err == nil && dir.IsDir() {
		return true
	}
	return false
}

func EnsureExists(path string) error {
	const perm = 0o755 // owner rwx, group rx, public rx
	return os.MkdirAll(path, perm)
}

// FSWorkspace

func LoadWorkspace(path string) (*FSWorkspace, error) {
	if !Exists(path) {
		return nil, errors.Errorf("couldn't find workspace at %s", path)
	}
	return &FSWorkspace{
		FS: pallets.AttachPath(os.DirFS(path), path),
	}, nil
}

func (w *FSWorkspace) GetCurrentEnvPath() string {
	return filepath.Join(w.FS.Path(), currentEnvDirName)
}

func (w *FSWorkspace) GetCurrentEnv() (*FSEnv, error) {
	pathedFS, err := w.FS.Sub(currentEnvDirName)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get local environment from workspace")
	}
	return &FSEnv{
		FS: pathedFS,
	}, nil
}

func (w *FSWorkspace) GetCachePath() string {
	return filepath.Join(w.FS.Path(), cacheDirName)
}

func (w *FSWorkspace) GetCache() (*FSCache, error) {
	pathedFS, err := w.FS.Sub(cacheDirName)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get cache from workspace")
	}
	return &FSCache{
		FS: pathedFS,
	}, nil
}
