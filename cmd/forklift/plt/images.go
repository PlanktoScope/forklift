package plt

import (
	"fmt"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// cache-img

func cacheImgAction(c *cli.Context) error {
	pallet, cache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}

	fmt.Println("Downloading Docker container images specified by the local pallet...")
	if err := fcli.DownloadImages(0, pallet, cache); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Done! Next, you'll probably want to run `sudo -E forklift plt apply`.")
	return nil
}
