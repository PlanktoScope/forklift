package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift/dev"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
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

// cache

func devEnvCacheAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	wpath := c.String("workspace")

	fmt.Printf("Downloading Pallet repositories specified by the development environment...\n")
	changed, err := downloadRepos(envPath, workspace.CachePath(wpath))
	if err != nil {
		return err
	}
	if !changed {
		fmt.Println("Done! No further actions are needed at this time.")
		return nil
	}

	// TODO: download all Docker images used by packages in the repo - either by inspecting the
	// Docker stack definitions or by allowing packages to list Docker images used.
	fmt.Println(
		// TODO: add a command to check if resource constraints are all satisfied
		// "Done! Next, you might want to run `forklift dev env check` or `forklift dev env deploy`.",
		"Done! Next, you might want to run `forklift dev env deploy`.",
	)
	return nil
}

// deploy

func devEnvDeployAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) {
		fmt.Println("The cache is empty, please run `forklift dev env cache` first")
		return nil
	}

	if err := deployEnv(envPath, workspace.CachePath(wpath)); err != nil {
		return errors.Wrap(
			err, "couldn't deploy development environment (have you run `forklift env cache` recently?)",
		)
	}
	fmt.Println("Done!")
	return nil
}

// ls-repo

func devEnvLsRepoAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}

	return printEnvRepos(envPath)
}

// ls-depl

func devEnvLsDeplAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	wpath := c.String("workspace")
	if !workspace.Exists(workspace.CachePath(wpath)) {
		fmt.Println("The cache is empty, please run `forklift dev env cache` first")
		return nil
	}

	if err = printEnvDepls(envPath, workspace.CachePath(wpath)); err != nil {
		return errors.Wrap(
			err, "couldn't list deployments (have you run `forklift dev env cache` recently?)",
		)
	}
	return nil
}
