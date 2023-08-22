package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/core"
)

type FSWorkspace struct {
	FS core.PathedFS
}

const (
	currentEnvDirName = "env"
	cacheDirName      = "cache"
	cacheReposDirName = "repositories"
)
