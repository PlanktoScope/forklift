package pallets

import (
	"io/fs"
	"path/filepath"
)

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
		path: filepath.Join(f.path, dir),
	}, err
}
