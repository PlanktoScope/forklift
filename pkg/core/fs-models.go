package core

import (
	"io/fs"
)

// Pather is something with a path.
type Pather interface {
	// Path returns the path of the instance.
	Path() string
}

// A PathedFS provides access to a hierarchical file system locatable at some path.
type PathedFS interface {
	fs.FS
	Pather
	// Sub returns a PathedFS corresponding to the subtree rooted at dir.
	Sub(dir string) (PathedFS, error)
}
