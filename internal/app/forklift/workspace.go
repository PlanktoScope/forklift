package forklift

import (
	"os"
	"path"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/pallets"
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
		FS: pallets.AttachPath(os.DirFS(dirPath), dirPath),
	}, nil
}

func (w *FSWorkspace) GetCurrentEnvPath() string {
	return path.Join(w.FS.Path(), currentEnvDirName)
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
	return path.Join(w.FS.Path(), cacheDirName)
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
