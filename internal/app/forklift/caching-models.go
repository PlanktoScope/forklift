package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

type CachedPkg struct {
	pallets.FSPkg
	Repo pallets.FSRepo
}
