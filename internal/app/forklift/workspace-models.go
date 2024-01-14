package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/core"
)

type FSWorkspace struct {
	FS core.PathedFS
}

// in $HOME/.cache/forklift:

const (
	cacheDirPath      = ".cache/forklift"
	cacheReposDirName = "repositories"
)

// in $HOME/.local/share/forklift:

const (
	dataDirPath          = ".local/share/forklift"
	currentPalletDirName = "pallet" // TODO: cache pallets and track the "current" one in a file?
)
