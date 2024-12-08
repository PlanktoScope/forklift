package cache

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/pkg/core"
)

// ls-pkg

func lsPkgAction(c *cli.Context) error {
	cache, err := getRepoCache(c.String("workspace"), false)
	if err != nil {
		return err
	}
	if !cache.Exists() {
		return errMissingCache
	}

	// TODO: add a --pattern cli flag for the pattern
	pkgs, err := cache.LoadFSPkgs("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't identify packages")
	}
	sort.Slice(pkgs, func(i, j int) bool {
		return core.ComparePkgs(pkgs[i].Pkg, pkgs[j].Pkg) < 0
	})
	for _, pkg := range pkgs {
		fmt.Printf("%s@%s\n", pkg.Path(), pkg.Repo.Version)
	}
	return nil
}

// show-pkg

func showPkgAction(c *cli.Context) error {
	cache, err := getRepoCache(c.String("workspace"), false)
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
			"Couldn't parse package query %s as package_path@version", versionedPkgPath,
		)
	}
	pkg, err := cache.LoadFSPkg(pkgPath, version)
	if err != nil {
		return errors.Wrapf(err, "couldn't resolve package query %s@%s", pkgPath, version)
	}
	fcli.FprintPkg(0, os.Stdout, cache, pkg)
	return nil
}
