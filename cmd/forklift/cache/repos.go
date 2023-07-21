package cache

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// ls-repo

func lsRepoAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) {
		fmt.Printf("The cache is empty.")
		return nil
	}

	repos, err := forklift.ListCachedRepos(workspace.CacheFS(wpath))
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet repositories")
	}
	sort.Slice(repos, func(i, j int) bool {
		return pallets.CompareRepos(repos[i].Repo, repos[j].Repo) < 0
	})
	for _, repo := range repos {
		fmt.Printf("%s@%s\n", repo.Config.Repository.Path, repo.Version)
	}
	return nil
}

// show-repo

func showRepoAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) {
		fmt.Printf("The cache is empty.")
		return nil
	}

	versionedRepoPath := c.Args().First()
	repoPath, version, ok := strings.Cut(versionedRepoPath, "@")
	if !ok {
		return errors.Errorf(
			"Couldn't parse Pallet repo path %s as repo_path@version", versionedRepoPath,
		)
	}
	repo, err := forklift.FindCachedRepo(workspace.CacheFS(wpath), repoPath, version)
	if err != nil {
		return errors.Wrapf(err, "couldn't find Pallet repository %s@%s", repoPath, version)
	}
	printCachedRepo(0, repo)
	return nil
}

func printCachedRepo(indent int, repo *pallets.FSRepo) {
	fcli.IndentedPrintf(indent, "Cached Pallet repository: %s\n", repo.Config.Repository.Path)
	indent++

	fcli.IndentedPrintf(indent, "Version: %s\n", repo.Version)
	fcli.IndentedPrintf(indent, "Provided by Git repository: %s\n", repo.VCSRepoPath)
	fcli.IndentedPrintf(indent, "Path in cache: %s\n", repo.FS.Path())
	fcli.IndentedPrintf(indent, "Description: %s\n", repo.Config.Repository.Description)
	// TODO: show the README file
}
