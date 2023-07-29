package env

import (
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-depl

func lsDeplAction(c *cli.Context) error {
	env, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(env, overrideCache); err != nil {
		return err
	}

	return fcli.PrintEnvDepls(0, env, cache)
}

// show-depl

func showDeplAction(c *cli.Context) error {
	env, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(env, overrideCache); err != nil {
		return err
	}

	deplName := c.Args().First()
	return fcli.PrintDeplInfo(0, env, cache, deplName)
}
