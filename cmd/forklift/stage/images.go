package stage

import (
	"fmt"
	"os"

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
			c.String("platform"), c.Bool("parallel"), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}
		fmt.Fprintln(
			os.Stderr, "Done caching images! They will be used when the staged pallet bundle is applied.",
		)
		return nil
	}
}
