package plt

import (
	"fmt"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-img

func lsImgAction(c *cli.Context) error {
	pallet, cache, _, err := processFullBaseArgs(c, true, true)
	if err != nil {
		return err
	}

	images, err := fcli.ListRequiredImages(pallet, cache, c.Bool("include-disabled"))
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
		pallet, cache, _, err := processFullBaseArgs(c, true, true)
		if err != nil {
			return err
		}
		if err = fcli.CheckCompatibility(
			pallet, cache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		fmt.Println("Downloading Docker container images specified by the development pallet...")
		if err := fcli.DownloadImages(
			0, pallet, cache, c.Bool("include-disabled"), c.Bool("parallel"),
		); err != nil {
			return err
		}
		fmt.Println()
		fmt.Println("Done!")
		return nil
	}
}
