package plt

import (
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-pkg

func lsPkgAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c.String("workspace"), true)
	if err != nil {
		return err
	}

	return fcli.PrintPalletPkgs(0, plt, caches.r)
}

// locate-pkg

func locatePkgAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c.String("workspace"), true)
	if err != nil {
		return err
	}

	pkgPath := c.Args().First()
	return fcli.PrintPkgLocation(plt, caches.r, pkgPath)
}

// show-pkg

func showPkgAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c.String("workspace"), true)
	if err != nil {
		return err
	}

	pkgPath := c.Args().First()
	return fcli.PrintPkgInfo(0, plt, caches.r, pkgPath)
}
