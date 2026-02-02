// Package workspaces implements storage and organization of Forklift-related data in the user's
// HOME directory.
package workspaces

import (
	"path"

	"github.com/pkg/errors"

	ffs "github.com/forklift-run/forklift/exp/fs"
)

type FSWorkspace struct {
	FS ffs.PathedFS
}

// in $HOME/.local/share/forklift:

const (
	dataDirPath = ".local/share/forklift"
)

// in $HOME/.config/forklift:

const (
	configDirPath = ".config/forklift"
)

// FSWorkspace

// LoadWorkspace loads the workspace at the specified path.
// The workspace is usually just a home directory, e.g. $HOME; directories in the workspace are
// organized with the same structure as the default structure described by the
// [XDG base directory spec](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html).
// The provided path must use the host OS's path separators.
func LoadWorkspace(dirPath string) (*FSWorkspace, error) {
	if !ffs.DirExists(dirPath) {
		return nil, errors.Errorf("couldn't find workspace at %s", dirPath)
	}
	return &FSWorkspace{
		FS: ffs.DirFS(dirPath),
	}, nil
}

// Data

func (w *FSWorkspace) GetDataPath() string {
	return path.Join(w.FS.Path(), dataDirPath)
}

func (w *FSWorkspace) getDataFS() (ffs.PathedFS, error) {
	if err := ffs.EnsureExists(w.GetDataPath()); err != nil {
		return nil, errors.Wrapf(err, "couldn't ensure the existence of %s", w.GetDataPath())
	}

	fsys, err := w.FS.Sub(dataDirPath)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get data directory from workspace")
	}
	return fsys, nil
}

// Config

func (w *FSWorkspace) getConfigPath() string {
	return path.Join(w.FS.Path(), configDirPath)
}

func (w *FSWorkspace) getConfigFS() (ffs.PathedFS, error) {
	if err := ffs.EnsureExists(w.getConfigPath()); err != nil {
		return nil, errors.Wrapf(err, "couldn't ensure the existence of %s", w.getConfigPath())
	}

	fsys, err := w.FS.Sub(configDirPath)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get config directory from workspace")
	}
	return fsys, nil
}
