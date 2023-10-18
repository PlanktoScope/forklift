package plt

import (
	"fmt"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// cache-img

func cacheImgAction(toolVersion, minVersion string) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, cache, overrideCache, err := processFullBaseArgs(c, true)
		if err != nil {
			return err
		}
		if err = setOverrideCacheVersions(pallet, overrideCache); err != nil {
			return err
		}
		if err = fcli.CheckCompatibility(
			pallet, cache, toolVersion, minVersion, c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		fmt.Println("Downloading Docker container images specified by the development pallet...")
		if err := fcli.DownloadImages(0, pallet, cache); err != nil {
			return err
		}
		fmt.Println()
		fmt.Println("Done! Next, you'll probably want to run `sudo -E forklift dev plt apply`.")
		return nil
	}
}
