package env

import (
	"fmt"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
)

// cache-img

func cacheImgAction(c *cli.Context) error {
	envPath, wpath, replacementRepos, err := processFullBaseArgs(c)
	if err != nil {
		return nil
	}

	fmt.Println("Downloading Docker container images specified by the development environment...")
	if err := fcli.DownloadImages(
		0, envPath, workspace.CachePath(wpath), replacementRepos,
	); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Done! Next, you'll probably want to run `sudo -E forklift dev env apply`.")
	return nil
}
