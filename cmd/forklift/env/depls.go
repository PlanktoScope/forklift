package env

import (
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-depl

func lsDeplAction(c *cli.Context) error {
	env, cache, err := processFullBaseArgs(c)
	if err != nil {
		return err
	}

	return fcli.PrintEnvDepls(0, env, cache, nil)
}

// show-depl

func showDeplAction(c *cli.Context) error {
	env, cache, err := processFullBaseArgs(c)
	if err != nil {
		return err
	}

	deplName := c.Args().First()
	return fcli.PrintDeplInfo(0, env, cache, nil, deplName)
}
