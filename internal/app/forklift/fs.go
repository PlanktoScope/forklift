package forklift

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/core"
)

// ReadLinkFS is the interface implemented by a file system that supports symbolic links.
// This is a stopgap until https://github.com/golang/go/issues/49580 is implemented.
type ReadLinkFS interface {
	core.PathedFS

	// ReadLink returns the destination of the named symbolic link.
	ReadLink(name string) (string, error)

	// StatLink returns a [fs.FileInfo] describing the file without following any symbolic links.
	// If there is an error, it should be of type [*fs.PathError].
	StatLink(name string) (fs.FileInfo, error)
}

// ReadLink returns the destination of the named symbolic link.
//
// If fsys does not implement ReadLinkFS, then ReadLink returns an error.
func ReadLink(fsys fs.FS, name string) (string, error) {
	if fsys, ok := fsys.(ReadLinkFS); ok {
		return fsys.ReadLink(name)
	}
	return "", errors.New("filesystem does not support ReadLink")
}

// StatLink returns a [fs.FileInfo] describing the file without following any symbolic links.
//
// If fsys does not implement ReadLinkFS, then ReadLink returns an error.
func StatLink(fsys fs.FS, name string) (fs.FileInfo, error) {
	if fsys, ok := fsys.(ReadLinkFS); ok {
		return fsys.StatLink(name)
	}
	return nil, errors.New("filesystem does not support ReadLink")
}

// DirFS returns a filesystem (a ReadLinkFS) for a tree of files rooted at the directory dir.
func DirFS(dir string) ReadLinkFS {
	return &dirFS{
		path: dir,
		fsys: os.DirFS(dir),
	}
}

// dirFS

type dirFS struct {
	path string
	fsys fs.FS
}

func (f dirFS) Path() string {
	return f.path
}

func (f dirFS) Open(name string) (fs.File, error) {
	return f.fsys.Open(name)
}

func (f dirFS) Sub(name string) (core.PathedFS, error) {
	return DirFS(path.Join(f.Path(), name)), nil
}

// dirFS: fs.ReadDirFS

func (f dirFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(f.fsys, name)
}

// dirFS: fs.ReadFileFS

func (f dirFS) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(f.fsys, name)
}

// dirFS: fs.StatFS

func (f dirFS) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(f.fsys, name)
}

// dirFS: ReadLinkFS

func (f dirFS) ReadLink(name string) (string, error) {
	return os.Readlink(filepath.FromSlash(path.Join(f.Path(), name)))
}

func (f dirFS) StatLink(name string) (fs.FileInfo, error) {
	return os.Lstat(filepath.FromSlash(path.Join(f.Path(), name)))
}
