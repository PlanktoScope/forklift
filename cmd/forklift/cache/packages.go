package cache

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	fcli "github.com/forklift-run/forklift/internal/app/forklift/cli"
	fpkg "github.com/forklift-run/forklift/pkg/packaging"
)

// ls-pkg

func lsPkgAction(c *cli.Context) error {
	cache, err := getPalletCache(c.String("workspace"), false)
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
	slices.SortFunc(pkgs, fpkg.CompareFSPkgs)
	for _, pkg := range pkgs {
		fmt.Printf("%s@%s\n", pkg.Path(), pkg.FSPkgTree.Version)
	}
	return nil
}

// show-pkg

func showPkgAction(c *cli.Context) error {
	cache, err := getPalletCache(c.String("workspace"), false)
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
