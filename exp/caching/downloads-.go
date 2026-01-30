package caching

import (
	"os"
	"path/filepath"

	ffs "github.com/forklift-run/forklift/exp/fs"
)

// FSDownloadCache is a source of downloaded files saved on the filesystem.
type FSDownloadCache struct {
	// FS is the filesystem which corresponds to the cache of downloads.
	FS ffs.PathedFS
}

// Exists checks whether the cache actually exists on the OS's filesystem.
func (c *FSDownloadCache) Exists() bool {
	return ffs.DirExists(filepath.FromSlash(c.FS.Path()))
}

// Remove deletes the cache from the OS's filesystem, if it exists.
func (c *FSDownloadCache) Remove() error {
	return os.RemoveAll(filepath.FromSlash(c.FS.Path()))
}

// Path returns the path of the cache's filesystem.
func (c *FSDownloadCache) Path() string {
	return c.FS.Path()
}
