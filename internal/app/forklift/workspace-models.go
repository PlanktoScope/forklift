package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/core"
)

type FSWorkspace struct {
	FS core.PathedFS
}

// in $HOME/.cache/forklift:

const (
	cacheDirPath        = ".cache/forklift"
	cacheReposDirName   = "repositories"
	cachePalletsDirName = "pallets"
)

// in $HOME/.local/share/forklift:

const (
	dataDirPath              = ".local/share/forklift"
	dataCurrentPalletDirName = "pallet" // TODO: cache pallets and track the "current" one in a file?
	dataStageStoreDirName    = "stages"
)
