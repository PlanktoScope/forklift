package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift/dev"
)

// info

func devEnvInfoAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	fmt.Printf("Development environment: %s\n", envPath)
	return printEnvInfo(envPath)
}

// ls-repo

func devEnvLsRepoAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}

	return printEnvRepos(envPath)
}
