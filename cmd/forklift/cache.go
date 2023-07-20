package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	units "github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
	"github.com/PlanktoScope/forklift/internal/clients/docker"
)

// CLI

var cacheCmd = &cli.Command{
	Name:  "cache",
	Usage: "Manages the local cache of Pallet repositories and packages",
	Subcommands: []*cli.Command{
		{
			Name:     "ls-repo",
			Aliases:  []string{"ls-r", "list-repositories"},
			Category: "Query the cache",
			Usage:    "Lists cached repositories",
			Action:   cacheLsRepoAction,
		},
		{
			Name:      "show-repo",
			Aliases:   []string{"s-r", "show-repository"},
			Category:  "Query the cache",
			Usage:     "Describes a cached repository",
			ArgsUsage: "repository_path@version",
			Action:    cacheShowRepoAction,
		},
		{
			Name:     "ls-pkg",
			Aliases:  []string{"ls-p", "list-packages"},
			Category: "Query the cache",
			Usage:    "Lists packages offered by cached repositories",
			Action:   cacheLsPkgAction,
		},
		{
			Name:      "show-pkg",
			Aliases:   []string{"s-p", "show-package"},
			Category:  "Query the cache",
			Usage:     "Describes a cached package",
			ArgsUsage: "package_path@version",
			Action:    cacheShowPkgAction,
		},
		{
			Name:     "ls-img",
			Aliases:  []string{"ls-i", "list-images"},
			Category: "Query the cache",
			Usage:    "Lists Docker container images in the local cache",
			Action:   cacheLsImgAction,
		},
		{
			Name:      "show-img",
			Aliases:   []string{"s-i", "show-image"},
			Category:  "Query the cache",
			Usage:     "Describes a cached Docker container image",
			ArgsUsage: "image_sha",
			Action:    cacheShowImgAction,
		},
		{
			Name:     "rm",
			Aliases:  []string{"remove"},
			Category: "Modify the cache",
			Usage:    "Removes the locally-cached repositories and Docker container images",
			Action:   cacheRmAction,
		},
	},
}

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

// show-repo

func cacheShowRepoAction(c *cli.Context) error {
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

func printCachedRepo(indent int, repo forklift.CachedRepo) {
	fcli.IndentedPrintf(indent, "Cached Pallet repository: %s\n", repo.Config.Repository.Path)
	indent++

	fcli.IndentedPrintf(indent, "Version: %s\n", repo.Version)
	fcli.IndentedPrintf(indent, "Provided by Git repository: %s\n", repo.VCSRepoPath)
	fcli.IndentedPrintf(indent, "Path in cache: %s\n", repo.ConfigPath)
	fcli.IndentedPrintf(indent, "Description: %s\n", repo.Config.Repository.Description)
	// TODO: show the README file
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

// show-pkg

func cacheShowPkgAction(c *cli.Context) error {
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
	printCachedPkg(0, pkg)
	return nil
}

func printCachedPkg(indent int, pkg forklift.CachedPkg) {
	fcli.IndentedPrintf(indent, "Pallet package: %s\n", pkg.Path)
	indent++

	printCachedPkgRepo(indent, pkg)
	fcli.IndentedPrintf(indent, "Path in cache: %s\n", pkg.ConfigPath)
	fmt.Println()
	printPkgSpec(indent, pkg.Config.Package)
	fmt.Println()
	printDeplSpec(indent, pkg.Config.Deployment)
	fmt.Println()
	printFeatureSpecs(indent, pkg.Config.Features)
}

func printCachedPkgRepo(indent int, pkg forklift.CachedPkg) {
	fcli.IndentedPrintf(
		indent, "Provided by Pallet repository: %s\n", pkg.Repo.Config.Repository.Path,
	)
	indent++

	fcli.IndentedPrintf(indent, "Version: %s\n", pkg.Repo.Version)
	fcli.IndentedPrintf(indent, "Description: %s\n", pkg.Repo.Config.Repository.Description)
	fcli.IndentedPrintf(indent, "Provided by Git repository: %s\n", pkg.Repo.VCSRepoPath)
}

func printPkgSpec(indent int, spec forklift.PkgSpec) {
	fcli.IndentedPrintf(indent, "Description: %s\n", spec.Description)

	fcli.IndentedPrint(indent, "Maintainers:")
	if len(spec.Maintainers) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for _, maintainer := range spec.Maintainers {
		printMaintainer(indent+1, maintainer)
	}

	if spec.License != "" {
		fcli.IndentedPrintf(indent, "License: %s\n", spec.License)
	} else {
		fcli.IndentedPrintf(indent, "License: (custom license)\n")
	}

	fcli.IndentedPrint(indent, "Sources:")
	if len(spec.Sources) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for _, source := range spec.Sources {
		fcli.BulletedPrintf(indent+1, "%s\n", source)
	}
}

func printMaintainer(indent int, maintainer forklift.PkgMaintainer) {
	if maintainer.Email != "" {
		fcli.BulletedPrintf(indent, "%s <%s>\n", maintainer.Name, maintainer.Email)
	} else {
		fcli.BulletedPrintf(indent, "%s\n", maintainer.Name)
	}
}

func printDeplSpec(indent int, spec forklift.PkgDeplSpec) {
	fcli.IndentedPrintf(indent, "Deployment:\n")
	indent++

	// TODO: actually display the definition file?
	fcli.IndentedPrintf(indent, "Definition file: ")
	if len(spec.DefinitionFile) == 0 {
		fmt.Println("(none)")
		return
	}
	fmt.Println(spec.DefinitionFile)
}

func printFeatureSpecs(indent int, features map[string]forklift.PkgFeatureSpec) {
	fcli.IndentedPrint(indent, "Optional features:")
	names := make([]string, 0, len(features))
	for name := range features {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, name := range names {
		if description := features[name].Description; description != "" {
			fcli.IndentedPrintf(indent, "%s: %s\n", name, description)
			continue
		}
		fcli.IndentedPrintf(indent, "%s\n", name)
	}
}

// ls-img

func cacheLsImgAction(c *cli.Context) error {
	client, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}

	imgs, err := client.ListImages(context.Background(), c.Args().First())
	if err != nil {
		return errors.Wrapf(err, "couldn't list local Docker images")
	}
	sort.Slice(imgs, func(i, j int) bool {
		return imgs[i].Repository < imgs[j].Repository
	})
	for _, img := range imgs {
		fmt.Printf("%s: %s", img.ID, img.Repository)
		if img.Tag != "" {
			fmt.Printf(":%s", img.Tag)
		}
		fmt.Println()
	}
	return nil
}

// show-img

func cacheShowImgAction(c *cli.Context) error {
	client, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}

	imageHash := c.Args().First()
	image, err := client.InspectImage(context.Background(), imageHash)
	if err != nil {
		return errors.Wrapf(err, "couldn't inspect image %s", imageHash)
	}
	printImg(0, image)
	return nil
}

func printImg(indent int, img docker.Image) {
	fcli.IndentedPrintf(indent, "Docker container image: %s\n", img.ID)
	indent++

	fcli.IndentedPrint(indent, "Provided by container image repository: ")
	if img.Repository == "" {
		fmt.Print("(none)")
	} else {
		fmt.Print(img.Repository)
	}
	fmt.Println()

	printImgRepoTags(indent+1, img.Inspect.RepoTags)
	printImgRepoDigests(indent+1, img.Inspect.RepoDigests)

	fcli.IndentedPrintf(indent, "Created: %s\n", img.Inspect.Created)
	fcli.IndentedPrintf(indent, "Size: %s\n", units.HumanSize(float64(img.Inspect.Size)))
}

func printImgRepoTags(indent int, tags []string) {
	fcli.IndentedPrint(indent, "Repo tags:")
	if len(tags) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, tag := range tags {
		fcli.BulletedPrintf(indent, "%s\n", tag)
	}
}

func printImgRepoDigests(indent int, digests []string) {
	fcli.IndentedPrint(indent, "Repo digests:")
	if len(digests) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, digest := range digests {
		fcli.BulletedPrintf(indent, "%s\n", digest)
	}
}

// rm

func cacheRmAction(c *cli.Context) error {
	wpath := c.String("workspace")
	fmt.Printf("Removing cache from workspace %s...\n", wpath)
	if err := workspace.RemoveCache(wpath); err != nil {
		return errors.Wrap(err, "couldn't remove forklift cache of repositories")
	}

	fmt.Println("Removing unused Docker container images...")
	client, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}
	report, err := client.PruneUnusedImages(context.Background())
	if err != nil {
		return errors.Wrap(err, "couldn't prune unused Docker container images")
	}
	sort.Slice(report.ImagesDeleted, func(i, j int) bool {
		return docker.CompareDeletedImages(report.ImagesDeleted[i], report.ImagesDeleted[j]) < 0
	})
	for _, deleted := range report.ImagesDeleted {
		if deleted.Untagged != "" {
			fmt.Printf("Untagged %s\n", deleted.Untagged)
		}
		if deleted.Deleted != "" {
			fmt.Printf("Deleted %s\n", deleted.Deleted)
		}
	}
	fmt.Printf("Total reclaimed space: %s\n", units.HumanSize(float64(report.SpaceReclaimed)))
	return nil
}
