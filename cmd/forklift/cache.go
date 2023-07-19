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
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
	"github.com/PlanktoScope/forklift/internal/clients/docker"
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
	indentedPrintf(indent, "Cached Pallet repository: %s\n", repo.Config.Repository.Path)
	indent++

	indentedPrintf(indent, "Version: %s\n", repo.Version)
	indentedPrintf(indent, "Provided by Git repository: %s\n", repo.VCSRepoPath)
	indentedPrintf(indent, "Path in cache: %s\n", repo.ConfigPath)
	indentedPrintf(indent, "Description: %s\n", repo.Config.Repository.Description)
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
	indentedPrintf(indent, "Pallet package: %s\n", pkg.Path)
	indent++

	printCachedPkgRepo(indent, pkg)
	indentedPrintf(indent, "Path in cache: %s\n", pkg.ConfigPath)
	fmt.Println()
	printPkgSpec(indent, pkg.Config.Package)
	fmt.Println()
	printDeplSpec(indent, pkg.Config.Deployment)
	fmt.Println()
	printFeatureSpecs(indent, pkg.Config.Features)
}

func printCachedPkgRepo(indent int, pkg forklift.CachedPkg) {
	indentedPrintf(indent, "Provided by Pallet repository: %s\n", pkg.Repo.Config.Repository.Path)
	indent++

	indentedPrintf(indent, "Version: %s\n", pkg.Repo.Version)
	indentedPrintf(indent, "Description: %s\n", pkg.Repo.Config.Repository.Description)
	indentedPrintf(indent, "Provided by Git repository: %s\n", pkg.Repo.VCSRepoPath)
}

func printPkgSpec(indent int, spec forklift.PkgSpec) {
	indentedPrintf(indent, "Description: %s\n", spec.Description)

	indentedPrint(indent, "Maintainers:")
	if len(spec.Maintainers) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for _, maintainer := range spec.Maintainers {
		printMaintainer(indent+1, maintainer)
	}

	if spec.License != "" {
		indentedPrintf(indent, "License: %s\n", spec.License)
	} else {
		indentedPrintf(indent, "License: (custom license)\n")
	}

	indentedPrint(indent, "Sources:")
	if len(spec.Sources) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for _, source := range spec.Sources {
		bulletedPrintf(indent+1, "%s\n", source)
	}
}

func printMaintainer(indent int, maintainer forklift.PkgMaintainer) {
	if maintainer.Email != "" {
		bulletedPrintf(indent, "%s <%s>\n", maintainer.Name, maintainer.Email)
	} else {
		bulletedPrintf(indent, "%s\n", maintainer.Name)
	}
}

func printDeplSpec(indent int, spec forklift.PkgDeplSpec) {
	indentedPrintf(indent, "Deployment:\n")
	indent++

	// TODO: actually display the definition file?
	indentedPrintf(indent, "Definition file: ")
	if len(spec.DefinitionFile) == 0 {
		fmt.Println("(none)")
		return
	}
	fmt.Println(spec.DefinitionFile)
}

func printFeatureSpecs(indent int, features map[string]forklift.PkgFeatureSpec) {
	indentedPrint(indent, "Optional features:")
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
			indentedPrintf(indent, "%s: %s\n", name, description)
			continue
		}
		indentedPrintf(indent, "%s\n", name)
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
	indentedPrintf(indent, "Docker container image: %s\n", img.ID)
	indent++

	indentedPrint(indent, "Provided by container image repository: ")
	if img.Repository == "" {
		fmt.Print("(none)")
	} else {
		fmt.Print(img.Repository)
	}
	fmt.Println()

	printImgRepoTags(indent+1, img.Inspect.RepoTags)
	printImgRepoDigests(indent+1, img.Inspect.RepoDigests)

	indentedPrintf(indent, "Created: %s\n", img.Inspect.Created)
	indentedPrintf(indent, "Size: %s\n", units.HumanSize(float64(img.Inspect.Size)))
}

func printImgRepoTags(indent int, tags []string) {
	indentedPrint(indent, "Repo tags:")
	if len(tags) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, tag := range tags {
		bulletedPrintf(indent, "%s\n", tag)
	}
}

func printImgRepoDigests(indent int, digests []string) {
	indentedPrint(indent, "Repo digests:")
	if len(digests) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, digest := range digests {
		bulletedPrintf(indent, "%s\n", digest)
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
