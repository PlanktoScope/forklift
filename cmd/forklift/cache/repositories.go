package cache

import (
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/pkg/core"
)

// ls-repo

func lsRepoAction(c *cli.Context) error {
	cache, err := getRepoCache(c.String("workspace"), false)
	if err != nil {
		return err
	}
	if !cache.Exists() {
		return errMissingCache
	}

	// TODO: add a --pattern cli flag for the pattern
	return lsGitRepo("repo", "**", cache.LoadFSRepos, func(r, s *core.FSRepo) int {
		return core.CompareRepos(r.Repo, s.Repo)
	})
}

// show-repo

func showRepoAction(c *cli.Context) error {
	cache, err := getRepoCache(c.String("workspace"), false)
	if err != nil {
		return err
	}
	if !cache.Exists() {
		return errMissingCache
	}

	return showGitRepo(cache, c.Args().First(), cache.LoadFSRepo, fcli.PrintCachedRepo)
}
