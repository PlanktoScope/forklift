package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

type CachedRepo struct {
	VCSRepoPath string
	Version     string
	RepoSubdir  string
	ConfigPath  string
	Config      pallets.RepoConfig
}

type CachedPkg struct {
	pallets.FSPkg
	Repo CachedRepo
}
