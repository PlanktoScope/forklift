package env

import (
	"fmt"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// cache-img

func cacheImgAction(c *cli.Context) error {
	env, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(env, overrideCache); err != nil {
		return err
	}

	fmt.Println("Downloading Docker container images specified by the development environment...")
	if err := fcli.DownloadImages(0, env, cache); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Done! Next, you'll probably want to run `sudo -E forklift dev env apply`.")
	return nil
}
