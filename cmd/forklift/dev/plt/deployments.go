package plt

import (
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-depl

func lsDeplAction(c *cli.Context) error {
	pallet, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(pallet, overrideCache); err != nil {
		return err
	}

	return fcli.PrintPalletDepls(0, pallet, cache)
}

// show-depl

func showDeplAction(c *cli.Context) error {
	pallet, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(pallet, overrideCache); err != nil {
		return err
	}

	deplName := c.Args().First()
	return fcli.PrintDeplInfo(0, pallet, cache, deplName)
}

// locate-depl-pkg

func locateDeplPkgAction(c *cli.Context) error {
	pallet, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(pallet, overrideCache); err != nil {
		return err
	}

	deplName := c.Args().First()
	return fcli.PrintDeplPkgPath(0, pallet, cache, deplName, c.Bool("allow-disabled"))
}
