package plt

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	fcli "github.com/forklift-run/forklift/internal/app/forklift/cli"
)

// ls-dl

func lsDlAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		enableOverrides: true,
		merge:           true,
	})
	if err != nil {
		return err
	}

	http, oci, err := fcli.ListRequiredDownloads(plt, caches.pp, c.Bool("include-disabled"))
	if err != nil {
		return err
	}
	for _, download := range http {
		fmt.Println(download)
	}
	for _, download := range oci {
		fmt.Println(download)
	}
	return nil
}

// cache-dl

func cacheDlAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requirePalletCache: true,
			enableOverrides:    true,
			merge:              true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if err := fcli.DownloadExportFiles(
			0, plt, caches.pp, caches.d, c.String("platform"), false, c.Bool("parallel"),
		); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Done!")
		return nil
	}
}
