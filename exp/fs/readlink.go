package fs

import (
	"io/fs"

	"github.com/pkg/errors"
)

// ReadLinkFS is the interface implemented by a file system that supports symbolic links.
// TODO: replace this with [fs.ReadLinkFS].
type ReadLinkFS interface {
	PathedFS

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
