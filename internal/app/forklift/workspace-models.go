package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

type FSWorkspace struct {
	FS pallets.PathedFS
}

const (
	currentEnvDirName   = "env"
	cacheDirName        = "cache"
	cachePalletsDirName = "pallets"
)
