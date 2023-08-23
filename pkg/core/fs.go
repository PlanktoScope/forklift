package core

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// Pather

// CoversPath checks whether the provided path would be within a child directory of the Pather if
// the Pather were a filesystem.
func CoversPath(pather Pather, path string) bool {
	if len(pather.Path()) == 0 {
		return true
	}
	// TODO: handle "/", ".", and ".."
	return strings.HasPrefix(path, fmt.Sprintf("%s/", pather.Path()))
}

// GetSubdirPath removes the path of the Pather instance from the start of the provided path,
// resulting in the subdirectory path of the provided path under the path of the Pather instance.
// If the Pather instance's path is not a parent directory of the provided path, the result is
// unchanged from the provided path.
func GetSubdirPath(pather Pather, path string) string {
	if len(pather.Path()) == 0 {
		return path
	}
	// TODO: handle "/", ".", and ".."...maybe use path.Rel?
	return strings.TrimPrefix(path, fmt.Sprintf("%s/", pather.Path()))
}

// PathedFS

// AttachPath makes a [PathedFS] for fsys with the specified path.
func AttachPath(fsys fs.FS, path string) PathedFS {
	return pathedFS{
		FS:   fsys,
		path: path,
	}
}

// pathedFS

// pathedFS is a basic implementation of the [PathedFS] interface.
type pathedFS struct {
	fs.FS
	path string
}

// Path returns the path where the file system is located.
func (f pathedFS) Path() string {
	return f.path
}

// Sub returns a PathedFS corresponding to the subtree rooted at fsys's dir.
func (f pathedFS) Sub(dir string) (PathedFS, error) {
	subFS, err := fs.Sub(f.FS, dir)
	return pathedFS{
		FS:   subFS,
		path: path.Join(f.path, dir),
	}, err
}
