package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/core"
)

// Stage Store

// FSStageStore is a source of bundles rooted at a single path, with bundles stored as
// incrementally-numbered directories within a [core.PathedFS] filesystem.
type FSStageStore struct {
	// FS is the filesystem which corresponds to the store of staged pallets.
	FS core.PathedFS
}
