package env

import (
	"fmt"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
)

// cache-repo

func cacheRepoAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		return errMissingEnv
	}

	fmt.Println("Downloading Pallet repositories specified by the local environment...")
	changed, err := fcli.DownloadRepos(0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath))
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
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(wpath)) {
		fmt.Println("The local environment is empty.")
		return nil
	}

	return fcli.PrintEnvRepos(0, workspace.LocalEnvPath(wpath))
}

// show-repo

func showRepoAction(c *cli.Context) error {
	wpath, err := processFullBaseArgs(c)
	if err != nil {
		return nil
	}

	repoPath := c.Args().First()
	return fcli.PrintRepoInfo(
		0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), nil, repoPath,
	)
}
