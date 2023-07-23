package forklift

import (
	"path/filepath"
)

// Path returns the Pallet package path of the required package.
func (r PkgReq) Path() string {
	return filepath.Join(r.Repo.Path(), r.PkgSubdir)
}
