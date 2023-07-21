package pallets

import (
	"io/fs"
)

// A PathedFS provides access to a hierarchical file system locatable at some path.
type PathedFS interface {
	fs.FS
	// Path returns the path where the file system is located.
	Path() string
	// Sub returns a PathedFS corresponding to the subtree rooted at dir.
	Sub(dir string) (PathedFS, error)
}
