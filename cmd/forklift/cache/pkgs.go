package cache

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// ls-pkg

func lsPkgAction(c *cli.Context) error {
	cache, err := getCache(c.String("workspace"))
	if err != nil {
		return err
	}
	if !cache.Exists() {
		return errMissingCache
	}

	pkgs, err := cache.ListPkgs("")
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet packages")
	}
	sort.Slice(pkgs, func(i, j int) bool {
		return pallets.CompareFSPkgs(pkgs[i], pkgs[j]) < 0
	})
	for _, pkg := range pkgs {
		fmt.Printf("%s@%s\n", pkg.Path(), pkg.Repo.Version)
	}
	return nil
}

// show-pkg

func showPkgAction(c *cli.Context) error {
	cache, err := getCache(c.String("workspace"))
	if err != nil {
		return err
	}
	if !cache.Exists() {
		return errMissingCache
	}

	versionedPkgPath := c.Args().First()
	pkgPath, version, ok := strings.Cut(versionedPkgPath, "@")
	if !ok {
		return errors.Errorf(
			"Couldn't parse Pallet package path %s as repo_path@version", versionedPkgPath,
		)
	}
	pkg, err := cache.FindPkg(pkgPath, version)
	if err != nil {
		return errors.Wrapf(err, "couldn't find Pallet package %s@%s", pkgPath, version)
	}
	printCachedPkg(0, pkg)
	return nil
}

func printCachedPkg(indent int, pkg *pallets.FSPkg) {
	fcli.IndentedPrintf(indent, "Pallet package: %s\n", pkg.Path())
	indent++

	printCachedPkgRepo(indent, pkg)
	fcli.IndentedPrintf(indent, "Path in cache: %s\n", pkg.FS.Path())
	fmt.Println()
	fcli.PrintPkgSpec(indent, pkg.Config.Package)
	fmt.Println()
	fcli.PrintDeplSpec(indent, pkg.Config.Deployment)
	fmt.Println()
	fcli.PrintFeatureSpecs(indent, pkg.Config.Features)
}

func printCachedPkgRepo(indent int, pkg *pallets.FSPkg) {
	fcli.IndentedPrintf(
		indent, "Provided by Pallet repository: %s\n", pkg.Repo.Config.Repository.Path,
	)
	indent++

	fcli.IndentedPrintf(indent, "Version: %s\n", pkg.Repo.Version)
	fcli.IndentedPrintf(indent, "Description: %s\n", pkg.Repo.Config.Repository.Description)
	fcli.IndentedPrintf(indent, "Provided by Git repository: %s\n", pkg.Repo.VCSRepoPath)
}
