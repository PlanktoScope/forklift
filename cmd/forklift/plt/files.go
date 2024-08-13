package plt

import (
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-file

func lsFileAction(c *cli.Context) error {
	plt, _, err := processFullBaseArgs(c.String("workspace"), processingOptions{
		merge: true,
	})
	if err != nil {
		return err
	}

	return fcli.PrintPalletFiles(0, plt, c.Args().First())
}

// locate-file

func locateFileAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{
		requireRepoCache: true,
		merge:            true,
	})
	if err != nil {
		return err
	}

	return fcli.PrintFileLocation(plt, caches.r, c.Args().First())
}

// show-file

func showFileAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{
		requireRepoCache: true,
		merge:            true,
	})
	if err != nil {
		return err
	}

	return fcli.PrintFile(plt, caches.r, c.Args().First())
}