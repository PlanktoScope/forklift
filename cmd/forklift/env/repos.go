package env

import (
	"fmt"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// cache-repo

func cacheRepoAction(c *cli.Context) error {
	env, cache, err := processFullBaseArgs(c, false)
	if err != nil {
		return err
	}

	fmt.Println("Downloading Pallet repositories specified by the local environment...")
	changed, err := fcli.DownloadRepos(0, env, cache)
	if err != nil {
		return err
	}
	if !changed {
		fmt.Println("Done! No further actions are needed at this time.")
		return nil
	}
	fmt.Println("Done! Next, you'll probably want to run `sudo -E forklift env apply`.")
	return nil
}

// ls-repo

func lsRepoAction(c *cli.Context) error {
	env, err := getEnv(c.String("workspace"))
	if err != nil {
		return err
	}

	return fcli.PrintEnvRepos(0, env)
}

// show-repo

func showRepoAction(c *cli.Context) error {
	env, cache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}

	repoPath := c.Args().First()
	return fcli.PrintRepoInfo(0, env, cache, nil, repoPath)
}
