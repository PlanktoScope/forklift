package stage

import (
	"fmt"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// cache-img

func cacheImgAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), c.String("stage-store"), versions)
		if err != nil {
			return err
		}
		if !store.Exists() {
			return errMissingStore
		}

		if err = fcli.DownloadImagesForStoreApply(
			0, store, versions.Tool, versions.MinSupportedBundle,
			c.Bool("parallel"), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}
		fmt.Println("Done caching images! They will be used when the staged pallet bundle is applied.")
		return nil
	}
}
