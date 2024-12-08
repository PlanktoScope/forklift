package plt

import (
	"os"

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

	return fcli.FprintPalletFeatures(0, os.Stdout, plt)
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

	return fcli.FprintFeatureInfo(0, os.Stdout, plt, caches.p, c.Args().First())
}
