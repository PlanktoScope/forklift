package workspaces

import (
	"path"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/exp/caching"
	ffs "github.com/forklift-run/forklift/exp/fs"
)

// in $HOME/.cache/forklift:

const (
	cacheDirPath          = ".cache/forklift"
	cacheMirrorsDirName   = "mirrors"
	cachePalletsDirName   = "pallets"
	cacheDownloadsDirName = "downloads"
)

// FSWorkspace: Caching

func (w *FSWorkspace) getCachePath() string {
	return path.Join(w.FS.Path(), cacheDirPath)
}

func (w *FSWorkspace) getCacheFS() (ffs.PathedFS, error) {
	if err := ffs.EnsureExists(w.getCachePath()); err != nil {
		return nil, errors.Wrapf(err, "couldn't ensure the existence of %s", w.getCachePath())
	}

	fsys, err := w.FS.Sub(cacheDirPath)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get cache directory from workspace")
	}
	return fsys, nil
}

// FSWorkspace: Caching: Mirrors

func (w *FSWorkspace) GetMirrorCachePath() string {
	return path.Join(w.getCachePath(), cacheMirrorsDirName)
}

func (w *FSWorkspace) GetMirrorCache() (*caching.FSMirrorCache, error) {
	fsys, err := w.getCacheFS()
	if err != nil {
		return nil, err
	}
	pathedFS, err := fsys.Sub(cacheMirrorsDirName)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get mirrors cache from workspace")
	}
	return &caching.FSMirrorCache{
		FS: pathedFS,
	}, nil
}

// FSWorkspace: Caching: Pallets

func (w *FSWorkspace) GetPalletCachePath() string {
	return path.Join(w.getCachePath(), cachePalletsDirName)
}

func (w *FSWorkspace) GetPalletCache() (*caching.FSPalletCache, error) {
	fsys, err := w.getCacheFS()
	if err != nil {
		return nil, err
	}
	pathedFS, err := fsys.Sub(cachePalletsDirName)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get pallets cache from workspace")
	}
	return caching.NewFSPalletCache(pathedFS), nil
}

// FSWorkspace: Caching: Downloads

func (w *FSWorkspace) GetDownloadCachePath() string {
	return path.Join(w.getCachePath(), cacheDownloadsDirName)
}

func (w *FSWorkspace) GetDownloadCache() (*caching.FSDownloadCache, error) {
	fsys, err := w.getCacheFS()
	if err != nil {
		return nil, err
	}
	pathedFS, err := fsys.Sub(cacheDownloadsDirName)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get downloads cache from workspace")
	}
	return &caching.FSDownloadCache{
		FS: pathedFS,
	}, nil
}
