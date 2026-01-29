package fs

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

func DirExists(dirPath string) bool {
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

func (f dirFS) Sub(name string) (PathedFS, error) {
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
