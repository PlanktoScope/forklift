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
		fmt.Printf("%s@%s\n", repo.Config.Repository.Path, repo.Version)
	}
	return nil
}

// info-repo

func printCachedRepo(repo forklift.CachedRepo) {
	fmt.Printf("Cached Pallet repository: %s\n", repo.Config.Repository.Path)
	fmt.Printf("  Version: %s\n", repo.Version)
	fmt.Printf("  Provided by Git repository: %s\n", repo.VCSRepoPath)
	fmt.Printf("  Path in cache: %s\n", repo.ConfigPath)
	fmt.Printf("  Description: %s\n", repo.Config.Repository.Description)
}

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
	printCachedRepo(repo)
	return nil
}

// ls-pkg

func cacheLsPkgAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) {
		fmt.Println("The cache is empty.")
		return nil
	}

	pkgs, err := forklift.ListCachedPkgs(workspace.CacheFS(wpath), "")
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet packages")
	}
	sort.Slice(pkgs, func(i, j int) bool {
		return forklift.CompareCachedPkgs(pkgs[i], pkgs[j]) < 0
	})
	for _, pkg := range pkgs {
		fmt.Printf("%s@%s\n", pkg.Path, pkg.Repo.Version)
	}
	return nil
}

// info-pkg

func printPkgSpec(spec forklift.PkgSpec) {
	fmt.Printf("  Description: %s\n", spec.Description)
	if len(spec.Maintainers) > 0 {
		fmt.Printf("  Maintainers:\n")
		for _, maintainer := range spec.Maintainers {
			if maintainer.Email != "" {
				fmt.Printf("    %s <%s>\n", maintainer.Name, maintainer.Email)
			} else {
				fmt.Printf("    %s\n", maintainer.Name)
			}
		}
	}
	if spec.License != "" {
		fmt.Printf("  License: %s\n", spec.License)
	} else {
		fmt.Printf("  License: (custom license)\n")
	}
	if len(spec.Maintainers) > 0 {
		fmt.Printf("  Sources:\n")
		for _, source := range spec.Sources {
			fmt.Printf("    %s\n", source)
		}
	}
}

func printDeplSpec(spec forklift.PkgDeplSpec) {
	fmt.Printf("  Deployment:\n")
	fmt.Printf("    Deploys as: %s\n", spec.Name)
}

func printFeatureSpecs(features map[string]forklift.PkgFeatureSpec) {
	fmt.Printf("  Optional features:\n")
	names := make([]string, 0, len(features))
	for name := range features {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if description := features[name].Description; description != "" {
			fmt.Printf("    %s: %s\n", name, description)
			continue
		}
		fmt.Printf("    %s\n", name)
	}
}

func printCachedPkg(pkg forklift.CachedPkg) {
	fmt.Printf("Pallet package: %s\n", pkg.Path)
	fmt.Printf("  Provided by Pallet repository: %s\n", pkg.Repo.Config.Repository.Path)
	fmt.Printf("    Version: %s\n", pkg.Repo.Version)
	fmt.Printf("    Description: %s\n", pkg.Repo.Config.Repository.Description)
	fmt.Printf("    Provided by Git repository: %s\n", pkg.Repo.VCSRepoPath)
	fmt.Printf("  Path in cache: %s\n", pkg.ConfigPath)
	fmt.Println()
	printPkgSpec(pkg.Config.Package)
	fmt.Println()
	printDeplSpec(pkg.Config.Deployment)
	fmt.Println()
	printFeatureSpecs(pkg.Config.Features)
}

func cacheInfoPkgAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) {
		fmt.Println("The cache is empty.")
		return nil
	}

	versionedPkgPath := c.Args().First()
	pkgPath, version, ok := strings.Cut(versionedPkgPath, "@")
	if !ok {
		return errors.Errorf(
			"Couldn't parse Pallet package path %s as repo_path@version", versionedPkgPath,
		)
	}
	pkg, err := forklift.FindCachedPkg(workspace.CacheFS(wpath), pkgPath, version)
	if err != nil {
		return errors.Wrapf(err, "couldn't find Pallet package %s@%s", pkgPath, version)
	}
	printCachedPkg(pkg)
	return nil
}

// rm

func cacheRmAction(c *cli.Context) error {
	wpath := c.String("workspace")
	fmt.Printf("Removing cache from workspace %s...\n", wpath)
	return errors.Wrap(workspace.RemoveCache(wpath), "couldn't remove cache")
}
