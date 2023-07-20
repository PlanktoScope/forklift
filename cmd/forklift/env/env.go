package env

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
	"github.com/PlanktoScope/forklift/internal/clients/git"
)

var errMissingEnv = errors.Errorf(
	"you first need to set up a local environment with `forklift env clone`",
)

func processFullBaseArgs(c *cli.Context) (workspacePath string, err error) {
	workspacePath = c.String("workspace")
	if !workspace.Exists(workspace.LocalEnvPath(workspacePath)) {
		return "", errMissingEnv
	}
	if !workspace.Exists(workspace.CachePath(workspacePath)) {
		return "", errors.Errorf(
			"you first need to cache the repos specified by your environment with " +
				"`forklift env cache-repo`",
		)
	}
	return workspacePath, nil
}

// clone

func cloneAction(c *cli.Context) error {
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
	gitRepo, err := git.Clone(remote, local)
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
		if gitRepo, err = git.Clone(remote, local); err != nil {
			return errors.Wrapf(
				err, "couldn't clone environment %s at release %s to %s", remote, release, local,
			)
		}
	}
	fmt.Printf("Checking out release %s...\n", release)
	if err = gitRepo.Checkout(release); err != nil {
		return errors.Wrapf(
			err, "couldn't check out release %s at %s", release, local,
		)
	}
	fmt.Println("Done! Next, you'll probably want to run `forklift env cache-repo`.")
	return nil
}

// fetch

func fetchAction(c *cli.Context) error {
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
		fmt.Println("No updates from the remote release.")
	}
	// TODO: display changes
	return nil
}

// pull

func pullAction(c *cli.Context) error {
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

func rmAction(c *cli.Context) error {
	wpath := c.String("workspace")
	fmt.Printf("Removing local environment from workspace %s...\n", wpath)
	// TODO: return an error if there are uncommitted or unpushed changes to be removed - in which
	// case require a --force flag
	return errors.Wrap(workspace.RemoveLocalEnv(wpath), "couldn't remove local environment")
}

// show

func showAction(c *cli.Context) error {
	wpath := c.String("workspace")
	envPath := workspace.LocalEnvPath(wpath)
	if !workspace.Exists(envPath) {
		return errMissingEnv
	}
	return fcli.PrintEnvInfo(0, envPath)
}

// check

func checkAction(c *cli.Context) error {
	wpath, err := processFullBaseArgs(c)
	if err != nil {
		return nil
	}

	if err := fcli.CheckEnv(
		0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), nil,
	); err != nil {
		return err
	}
	return nil
}

// plan

func planAction(c *cli.Context) error {
	wpath, err := processFullBaseArgs(c)
	if err != nil {
		return nil
	}

	if err := fcli.PlanEnv(
		0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), nil,
	); err != nil {
		return errors.Wrap(
			err, "couldn't deploy local environment (have you run `forklift env cache` recently?)",
		)
	}
	return nil
}

// apply

func applyAction(c *cli.Context) error {
	wpath, err := processFullBaseArgs(c)
	if err != nil {
		return nil
	}

	if err := fcli.ApplyEnv(
		0, workspace.LocalEnvPath(wpath), workspace.CachePath(wpath), nil,
	); err != nil {
		return errors.Wrap(
			err, "couldn't deploy local environment (have you run `forklift env cache` recently?)",
		)
	}
	fmt.Println()
	fmt.Println("Done!")
	return nil
}
