package plt

import (
	"fmt"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-img

func lsImgAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		requireRepoCache: true,
		enableOverrides:  true,
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
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requirePalletCache: true,
			requireRepoCache:   true,
			enableOverrides:    true,
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

		fmt.Println("Downloading Docker container images specified by the development pallet...")
		if err := fcli.DownloadImages(
			0, plt, caches.r, c.String("platform"), c.Bool("include-disabled"), c.Bool("parallel"),
		); err != nil {
			return err
		}
		fmt.Println()
		fmt.Println("Done!")
		return nil
	}
}
