package plt

import (
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-feat

func lsFeatAction(c *cli.Context) error {
	plt, _, err := processFullBaseArgs(c, processingOptions{
		requirePalletCache: true,
		enableOverrides:    true,
		merge:              true,
	})
	if err != nil {
		return err
	}

	return fcli.PrintPalletFeatures(0, plt)
}

// show-feat

func showFeatAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		requirePalletCache: true,
		enableOverrides:    true,
		merge:              true,
	})
	if err != nil {
		return err
	}

	return fcli.PrintFeatureInfo(0, plt, caches.p, c.Args().First())
}
