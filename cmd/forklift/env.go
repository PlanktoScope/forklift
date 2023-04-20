package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
	"github.com/PlanktoScope/forklift/internal/clients/git"
)

var errMissingEnv = errors.Errorf(
	"you first need to set up a local environment with `forklift env clone`",
)

// clone

func envCloneAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !workspace.Exists(wpath) {
		fmt.Printf("Making a new workspace at %s...", wpath)
	}
	if err := workspace.EnsureExists(wpath); err != nil {
		return errors.Wrapf(err, "couldn't make new workspace at %s", wpath)
	}
	remoteRelease := c.Args().First()
	remote, release, err := git.ParseRemoteRelease(remoteRelease)
	if err != nil {
		return errors.Wrapf(err, "couldn't parse remote release %s", remoteRelease)
	}
	local := workspace.LocalEnvPath(wpath)
	fmt.Printf("Cloning environment %s to %s...\n", remote, local)
	repo, err := git.Clone(remote, local)
	if err != nil {
		if !errors.Is(err, git.ErrRepositoryAlreadyExists) {
			return errors.Wrapf(
				err, "couldn't clone environment %s at release %s to %s", remote, release, local,
			)
		}
		if !c.Bool("force") {
			return errors.Wrap(
				err,
				"you need to first delete your local environment with `forklift env rm` before "+
					"cloning another remote release to it",
			)
		}
		fmt.Printf(
			"Removing local environment from workspace %s, because it already exists and the "+
				"command's --force flag was enabled...\n",
			wpath,
		)
		if err = workspace.RemoveLocalEnv(wpath); err != nil {
			return errors.Wrap(err, "couldn't remove local environment")
		}
		fmt.Printf("Cloning environment %s to %s...\n", remote, local)
		if repo, err = git.Clone(remote, local); err != nil {
			return errors.Wrapf(
				err, "couldn't clone environment %s at release %s to %s", remote, release, local,
			)
		}
	}
	fmt.Printf("Checking out release %s...\n", release)
	if _, err = git.Checkout(repo, release); err != nil {
		return errors.Wrapf(
			err, "couldn't check out release %s at %s", release, local,
		)
	}
	fmt.Println("Done! Next, you'll probably want to run `forklift cache update`.")
	return nil
}

// fetch

func envFetchAction(c *cli.Context) error {
	wpath := c.String("workspace")
	envPath := workspace.LocalEnvPath(wpath)
	if !workspace.Exists(envPath) {
		return errMissingEnv
	}
	fmt.Println("Fetching updates...")
	updated, err := git.Fetch(envPath)
	if err != nil {
		return errors.Wrap(err, "couldn't fetch changes from the remote release")
	}
	if !updated {
		fmt.Print("No updates from the remote release.")
	}
	// TODO: display changes
	return nil
}

// pull

func envPullAction(c *cli.Context) error {
	wpath := c.String("workspace")
	envPath := workspace.LocalEnvPath(wpath)
	if !workspace.Exists(envPath) {
		return errMissingEnv
	}
	fmt.Println("Attempting to fast-forward the local environment...")
	updated, err := git.Pull(envPath)
	if err != nil {
		return errors.Wrap(err, "couldn't fast-forward the local environment")
	}
	if !updated {
		fmt.Println("No changes from the remote release.")
	}
	// TODO: display changes
	return nil
}

// rm

func envRmAction(c *cli.Context) error {
	wpath := c.String("workspace")
	fmt.Printf("Removing local environment from workspace %s...\n", wpath)
	return errors.Wrap(workspace.RemoveLocalEnv(wpath), "couldn't remove local environment")
}
