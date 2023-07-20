package env

import (
	"fmt"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
)

// cache-img

func cacheImgAction(c *cli.Context) error {
	wpath, err := processFullBaseArgs(c)
	if err != nil {
		return nil
	}

	fmt.Println("Downloading Docker container images specified by the local environment...")
	if err := fcli.DownloadImages(
		0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), nil,
	); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Done! Next, you'll probably want to run `sudo -E forklift env apply`.")
	return nil
}
