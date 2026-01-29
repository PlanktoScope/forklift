package cache

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	fcli "github.com/forklift-run/forklift/internal/app/forklift/cli"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	fws "github.com/forklift-run/forklift/pkg/workspaces"
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
	out io.Writer,
	cache ffs.Pather, versionQuery string,
	loader func(path, version string) (GitRepo, error),
	fprinter func(
		indent int, out io.Writer, cache ffs.Pather, gitRepo GitRepo, printHeader bool,
	) error,
	printHeader bool,
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
	return fprinter(0, out, cache, gitRepo, printHeader)
}

// add-*

func addGitRepoAction[Cache ffs.Pather](
	cacheGetter func(wpath string, ensureWorkspace bool) (Cache, error),
) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		cache, err := cacheGetter(c.String("workspace"), true)
		if err != nil {
			return err
		}
		workspace, err := fws.LoadWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}

		queries := c.Args().Slice()
		if _, _, err = fcli.DownloadQueriedGitReposUsingLocalMirrors(
			0, workspace.GetMirrorCachePath(), cache.Path(), queries,
		); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Done!")
		return nil
	}
}

type remover interface {
	Remove() error
}

// del-*

func delGitRepoAction[Cache remover](
	gitRepoType string, cacheGetter func(wpath string, ensureWorkspace bool) (Cache, error),
) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		cache, err := cacheGetter(c.String("workspace"), false)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Clearing %s cache...\n", gitRepoType)
		if err = cache.Remove(); err != nil {
			return errors.Wrapf(err, "couldn't clear %s cache", gitRepoType)
		}
		return nil
	}
}
