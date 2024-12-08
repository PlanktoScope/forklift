package plt

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-img

func lsImgAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{
		requireRepoCache: true,
		merge:            true,
	})
	if err != nil {
		return err
	}

	images, err := fcli.ListRequiredImages(plt, caches.r, c.Bool("include-disabled"))
	if err != nil {
		return err
	}
	for _, image := range images {
		fmt.Println(image)
	}
	return nil
}

// cache-img

func cacheImgAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{
			requirePalletCache: true,
			requireRepoCache:   true,
			merge:              true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, caches.r, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Downloading Docker container images specified by the local pallet...")
		if err := fcli.DownloadImages(
			0, plt, caches.r, c.String("platform"), c.Bool("include-disabled"), c.Bool("parallel"),
		); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Done!")
		return nil
	}
}
