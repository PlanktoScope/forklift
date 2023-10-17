package cache

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/pkg/core"
)

// ls-repo

func lsRepoAction(c *cli.Context) error {
	cache, err := getCache(c.String("workspace"))
	if err != nil {
		return err
	}
	if !cache.Exists() {
		return errMissingCache
	}

	// TODO: add a --pattern cli flag for the pattern
	loadedRepos, err := cache.LoadFSRepos("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't identify repos")
	}
	sort.Slice(loadedRepos, func(i, j int) bool {
		return core.CompareRepos(loadedRepos[i].Repo, loadedRepos[j].Repo) < 0
	})
	for _, repo := range loadedRepos {
		fmt.Printf("%s@%s\n", repo.Def.Repo.Path, repo.Version)
	}
	return nil
}

// show-repo

func showRepoAction(c *cli.Context) error {
	cache, err := getCache(c.String("workspace"))
	if err != nil {
		return err
	}
	if !cache.Exists() {
		return errMissingCache
	}

	versionedRepoPath := c.Args().First()
	repoPath, version, ok := strings.Cut(versionedRepoPath, "@")
	if !ok {
		return errors.Errorf(
			"Couldn't parse repo query %s as repo_path@version", versionedRepoPath,
		)
	}
	repo, err := cache.LoadFSRepo(repoPath, version)
	if err != nil {
		return errors.Wrapf(err, "couldn't find repo %s@%s", repoPath, version)
	}
	return printCachedRepo(0, cache, repo)
}

func printCachedRepo(indent int, cache *forklift.FSRepoCache, repo *core.FSRepo) error {
	fcli.IndentedPrintf(indent, "Cached repo: %s\n", repo.Path())
	indent++

	fcli.IndentedPrintf(indent, "Forklift version: %s\n", repo.Def.ForkliftVersion)
	fmt.Println()

	fcli.IndentedPrintf(indent, "Version: %s\n", repo.Version)
	fcli.IndentedPrintf(indent, "Path in cache: %s\n", core.GetSubdirPath(cache, repo.FS.Path()))
	fcli.IndentedPrintf(indent, "Description: %s\n", repo.Def.Repo.Description)

	readme, err := repo.LoadReadme()
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load readme file for repo %s@%s from cache", repo.Path(), repo.Version,
		)
	}
	fcli.IndentedPrintln(indent, "Readme:")
	const widthLimit = 100
	fcli.PrintReadme(indent+1, readme, widthLimit)
	return nil
}
