package cache

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/pkg/core"
)

// ls-*

type versionQuerier interface {
	VersionQuery() string
}

func lsGitRepo[GitRepo versionQuerier](
	gitRepoType, searchPattern string,
	loader func(searchPattern string) ([]GitRepo, error),
	comparer func(r, s GitRepo) int,
) error {
	allLoaded, err := loader(searchPattern)
	if err != nil {
		return errors.Wrapf(err, "couldn't identify %ss", gitRepoType)
	}
	sort.Slice(allLoaded, func(i, j int) bool {
		return comparer(allLoaded[i], allLoaded[j]) < 0
	})
	for _, loaded := range allLoaded {
		fmt.Println(loaded.VersionQuery())
	}
	return nil
}

// show-*

func showGitRepo[GitRepo any](
	cache core.Pather, versionQuery string,
	loader func(path, version string) (GitRepo, error),
	printer func(indent int, cache core.Pather, gitRepo GitRepo) error,
) error {
	gitRepoPath, version, ok := strings.Cut(versionQuery, "@")
	if !ok {
		return errors.Errorf(
			"Couldn't parse query %s as git_repo_path@version", versionQuery,
		)
	}
	gitRepo, err := loader(gitRepoPath, version)
	if err != nil {
		return errors.Wrapf(err, "couldn't find %s@%s", gitRepoPath, version)
	}
	return printer(0, cache, gitRepo)
}

// add-*

func addGitRepoAction[Cache core.Pather](
	cacheGetter func(wpath string, ensureWorkspace bool) (Cache, error),
) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		cache, err := cacheGetter(c.String("workspace"), true)
		if err != nil {
			return err
		}

		queries := c.Args().Slice()
		if _, _, err = fcli.DownloadQueriedGitReposUsingLocalMirrors(
			0, cache.Path(), queries,
		); err != nil {
			return err
		}
		fmt.Println()
		fmt.Println("Done!")
		return nil
	}
}

type remover interface {
	Remove() error
}

// rm-*

func rmGitRepoAction[Cache remover](
	gitRepoType string, cacheGetter func(wpath string, ensureWorkspace bool) (Cache, error),
) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		cache, err := cacheGetter(c.String("workspace"), false)
		if err != nil {
			return err
		}

		fmt.Printf("Clearing %s cache...", gitRepoType)
		if err = cache.Remove(); err != nil {
			return errors.Wrapf(err, "couldn't clear %s cache", gitRepoType)
		}
		return nil
	}
}
