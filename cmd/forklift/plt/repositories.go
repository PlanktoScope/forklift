package plt

import (
	"fmt"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// cache-repo

func cacheRepoAction(c *cli.Context) error {
	pallet, cache, err := processFullBaseArgs(c, false)
	if err != nil {
		return err
	}

	fmt.Println("Downloading repos specified by the local pallet...")
	changed, err := fcli.DownloadRepos(0, pallet, cache)
	if err != nil {
		return err
	}
	if !changed {
		fmt.Println("Done! No further actions are needed at this time.")
		return nil
	}
	fmt.Println("Done! Next, you'll probably want to run `sudo -E forklift plt apply`.")
	return nil
}

// ls-repo

func lsRepoAction(c *cli.Context) error {
	pallet, err := getPallet(c.String("workspace"))
	if err != nil {
		return err
	}

	return fcli.PrintPalletRepos(0, pallet)
}

// show-plt

func showRepoAction(c *cli.Context) error {
	pallet, cache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}

	repoPath := c.Args().First()
	return fcli.PrintRepoInfo(0, pallet, cache, repoPath)
}
