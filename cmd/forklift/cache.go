package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
)

// ls-repo

func cacheLsRepoAction(c *cli.Context) error {
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
		return forklift.CompareCachedRepos(repos[i], repos[j]) < 0
	})
	for _, repo := range repos {
		fmt.Printf("%s@%s\n", repo.Config.Path, repo.Version)
	}
	return nil
}

// info-repo

func cacheInfoRepoAction(c *cli.Context) error {
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
	fmt.Printf("Repo path: %s\n", repo.Config.Path)
	fmt.Printf("Repo version: %s\n", repo.Version)
	fmt.Printf("Provided by Git repository: %s\n", repo.VCSRepoPath)
	fmt.Printf("File path in cache: %s\n", repo.ConfigPath)
	return nil
}

// ls-pkg

func cacheLsPkgAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) {
		fmt.Printf("The cache is empty.")
		return nil
	}

	pkgs, err := forklift.ListCachedPkgs(workspace.CacheFS(wpath))
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet packages")
	}
	sort.Slice(pkgs, func(i, j int) bool {
		return forklift.CompareCachedPkgs(pkgs[i], pkgs[j]) < 0
	})
	for _, pkg := range pkgs {
		fmt.Printf(
			"%s@%s from %s@%s\n",
			pkg.Path, pkg.Config.Package.Version, pkg.Repo.Config.Path, pkg.Repo.Version,
		)
	}
	return nil
}

// rm

func cacheRmAction(c *cli.Context) error {
	wpath := c.String("workspace")
	fmt.Printf("Removing cache from workspace %s...\n", wpath)
	return errors.Wrap(workspace.RemoveCache(wpath), "couldn't remove cache")
}

// info-pkg

func cacheInfoPkgAction(c *cli.Context) error {
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
	fmt.Printf("Repo path: %s\n", repo.Config.Path)
	fmt.Printf("Repo version: %s\n", repo.Version)
	fmt.Printf("Provided by Git repository: %s\n", repo.VCSRepoPath)
	fmt.Printf("File path in cache: %s\n", repo.ConfigPath)
	return nil
}
