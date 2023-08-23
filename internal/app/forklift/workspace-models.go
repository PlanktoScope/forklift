package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/core"
)

type FSWorkspace struct {
	FS core.PathedFS
}

const (
	currentPalletDirName = "pallet" // TODO: cache pallets and track the "current" one in a file?
	cacheDirName         = "cache"
	cacheReposDirName    = "repositories"
)
