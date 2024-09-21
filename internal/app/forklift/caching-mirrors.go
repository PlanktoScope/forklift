package forklift

import (
	"os"
	"path/filepath"
)

// FSMirrorCache

// Exists checks whether the cache actually exists on the OS's filesystem.
func (c *FSMirrorCache) Exists() bool {
	return DirExists(filepath.FromSlash(c.FS.Path()))
}

// Remove deletes the cache from the OS's filesystem, if it exists.
func (c *FSMirrorCache) Remove() error {
	return os.RemoveAll(filepath.FromSlash(c.FS.Path()))
}

// Path returns the path of the cache's filesystem.
func (c *FSMirrorCache) Path() string {
	return c.FS.Path()
}
