package env

import (
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
)

// ls-depl

func lsDeplAction(c *cli.Context) error {
	wpath, err := processFullBaseArgs(c)
	if err != nil {
		return nil
	}

	return fcli.PrintEnvDepls(0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), nil)
}

// show-depl

func showDeplAction(c *cli.Context) error {
	wpath, err := processFullBaseArgs(c)
	if err != nil {
		return nil
	}

	deplName := c.Args().First()
	return fcli.PrintDeplInfo(
		0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), nil, deplName,
	)
}
