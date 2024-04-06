package stage

import (
	"fmt"

	"github.com/pkg/errors"
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

		next, hasNext := store.GetNext()
		current, hasCurrent := store.GetCurrent()

		if hasCurrent && current != next {
			bundle, err := store.LoadFSBundle(current)
			if err != nil {
				return errors.Wrapf(err, "couldn't load staged pallet bundle %d", current)
			}
			if err = checkShallowCompatibility(
				bundle, versions, c.Bool("ignore-tool-version"),
			); err != nil {
				return err
			}
			fmt.Println(
				"Downloading Docker container images specified by the last successfully-applied staged " +
					"pallet bundle, in case the next to be applied fails to be applied",
			)
			if err := fcli.DownloadImages(0, bundle, bundle, false, c.Bool("parallel")); err != nil {
				return err
			}
			fmt.Println()
		}
		if hasNext {
			bundle, err := store.LoadFSBundle(next)
			if err != nil {
				return errors.Wrapf(err, "couldn't load staged pallet bundle %d", next)
			}
			if err = checkShallowCompatibility(
				bundle, versions, c.Bool("ignore-tool-version"),
			); err != nil {
				return err
			}
			fmt.Println(
				"Downloading Docker container images specified by the next staged pallet bundle to be " +
					"applied...",
			)
			if err := fcli.DownloadImages(0, bundle, bundle, false, c.Bool("parallel")); err != nil {
				return err
			}
			fmt.Println()
		}

		fmt.Println("Done! Cached images will be used when you run `sudo -E forklift stage apply`.")
		return nil
	}
}
