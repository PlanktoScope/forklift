package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/core"
)

type FSWorkspace struct {
	FS core.PathedFS
}

// in $HOME/.cache/forklift:

const (
	cacheDirPath          = ".cache/forklift"
	cacheMirrorsDirName   = "mirrors"
	cacheReposDirName     = "repositories"
	cachePalletsDirName   = "pallets"
	cacheDownloadsDirName = "downloads"
)

// in $HOME/.local/share/forklift:

const (
	dataDirPath              = ".local/share/forklift"
	dataCurrentPalletDirName = "pallet"
	dataStageStoreDirName    = "stages"
)

// in $HOME/.config/forklift:

const (
	configDirPath                       = ".config/forklift"
	configCurrentPalletUpgradesFile     = "pallet-upgrades.yml"
	configCurrentPalletUpgradesSwapFile = "pallet-upgrades-swap.yml"
)
