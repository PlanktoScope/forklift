package plt

import (
	"os"

	"github.com/urfave/cli/v2"

	fcli "github.com/forklift-run/forklift/internal/app/forklift/cli"
)

// ls-pkg

func lsPkgAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		requireRepoCache: true,
		enableOverrides:  true,
		merge:            true,
	})
	if err != nil {
		return err
	}

	return fcli.FprintPalletPkgs(0, os.Stdout, plt, caches.p)
}

// locate-pkg

func locatePkgAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		requireRepoCache: true,
		enableOverrides:  true,
		merge:            true,
	})
	if err != nil {
		return err
	}

	return fcli.FprintPkgLocation(os.Stdout, plt, caches.p, c.Args().First())
}

// show-pkg

func showPkgAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		requireRepoCache: true,
		enableOverrides:  true,
		merge:            true,
	})
	if err != nil {
		return err
	}

	return fcli.FprintPkgInfo(0, os.Stdout, plt, caches.p, c.Args().First())
}
