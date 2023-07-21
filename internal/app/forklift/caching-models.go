package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

type CachedRepo struct {
	pallets.FSRepo
	// TODO: move version to pallets.Repo?
	Version string
}

type CachedPkg struct {
	pallets.FSPkg
	Repo CachedRepo
}
