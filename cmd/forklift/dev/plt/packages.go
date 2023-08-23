package plt

import (
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-pkg

func lsPkgAction(c *cli.Context) error {
	pallet, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(pallet, overrideCache); err != nil {
		return err
	}

	return fcli.PrintPalletPkgs(0, pallet, cache)
}

// show-pkg

func showPkgAction(c *cli.Context) error {
	pallet, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(pallet, overrideCache); err != nil {
		return err
	}

	pkgPath := c.Args().First()
	return fcli.PrintPkgInfo(0, pallet, cache, pkgPath)
}
