package forklift

import ffs "github.com/forklift-run/forklift/pkg/fs"

type FSWorkspace struct {
	FS ffs.PathedFS
}

// in $HOME/.cache/forklift:

const (
	cacheDirPath          = ".cache/forklift"
	cacheMirrorsDirName   = "mirrors"
	cachePkgTreesDirName  = "packages"
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
