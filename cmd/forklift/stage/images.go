package stage

import (
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// cache-img

func cacheImgAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), versions)
		if err != nil {
			return err
		}
		if !store.Exists() {
			return errMissingStore
		}

		return fcli.DownloadImagesForStoreApply(
			store, versions.Tool, versions.MinSupportedBundle,
			c.Bool("parallel"), c.Bool("ignore-tool-version"),
		)
	}
}
