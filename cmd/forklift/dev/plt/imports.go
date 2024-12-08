package plt

import (
	"os"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-imp

func lsImpAction(c *cli.Context) error {
	plt, err := getShallowPallet(c.String("cwd"))
	if err != nil {
		return err
	}

	return fcli.FprintPalletImports(0, os.Stdout, plt)
}

// show-imp

func showImpAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		requirePalletCache: true,
		enableOverrides:    true,
	})
	if err != nil {
		return err
	}

	importName := c.Args().First()
	return fcli.FprintImportInfo(0, os.Stdout, plt, caches.p, importName)
}
