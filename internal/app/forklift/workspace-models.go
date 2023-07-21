package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

type FSWorkspace struct {
	FS pallets.PathedFS
}

// Env

const currentEnvDirName = "env"

type FSEnv struct {
	FS pallets.PathedFS
}

// Cache

const cacheDirName = "cache"

type FSCache struct {
	FS pallets.PathedFS
}
