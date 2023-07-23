package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// FSCache is a local cache with copies of Pallet repositories (and thus of Pallet packages too),
// stored in a [fs.FS] filesystem.
type FSCache struct {
	// FS is the filesystem which corresponds to the cache.
	FS pallets.PathedFS
}
