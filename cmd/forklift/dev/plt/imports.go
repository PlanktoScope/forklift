package plt

import (
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-imp

func lsImpAction(c *cli.Context) error {
	plt, err := getShallowPallet(c.String("cwd"))
	if err != nil {
		return err
	}

	return fcli.PrintPalletImports(0, plt)
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
	return fcli.PrintImportInfo(0, plt, caches.p, importName)
}
