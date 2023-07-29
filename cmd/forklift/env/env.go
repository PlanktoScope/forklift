package env

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/internal/clients/git"
)

var errMissingEnv = errors.Errorf(
	"you first need to set up a local environment with `forklift env clone`",
)

func processFullBaseArgs(
	c *cli.Context, ensureCache bool,
) (env *forklift.FSEnv, cache *forklift.FSCache, err error) {
	workspace, err := forklift.LoadWorkspace(c.String("workspace"))
	if err != nil {
		return nil, nil, err
	}
	if env, err = workspace.GetCurrentEnv(); err != nil {
		return nil, nil, errMissingEnv
	}
	if cache, err = workspace.GetCache(); err != nil {
		return nil, nil, err
	}
	if ensureCache && !cache.Exists() {
		return nil, nil, errors.New(
			"you first need to cache the repos specified by your environment with " +
				"`forklift env cache-repo`",
		)
	}
	return env, cache, nil
}

func getEnv(wpath string) (env *forklift.FSEnv, err error) {
	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	if env, err = workspace.GetCurrentEnv(); err != nil {
		return nil, errMissingEnv
	}
	return env, nil
}

// clone

func cloneAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !forklift.Exists(wpath) {
		fmt.Printf("Making a new workspace at %s...", wpath)
	}
	if err := forklift.EnsureExists(wpath); err != nil {
		return errors.Wrapf(err, "couldn't make new workspace at %s", wpath)
	}

	remoteRelease := c.Args().First()
	remote, release, err := git.ParseRemoteRelease(remoteRelease)
	if err != nil {
		return errors.Wrapf(err, "couldn't parse remote release %s", remoteRelease)
	}
	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return errors.Wrap(err, "couldn't load workspace")
	}
	local := workspace.GetCurrentEnvPath()
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

		env, eerr := workspace.GetCurrentEnv()
		if eerr != nil {
			return err
		}
		fmt.Println(
			"Removing local environment from workspace, because it already exists and the " +
				"command's --force flag was enabled...",
		)
		if err = env.Remove(); err != nil {
			return errors.Wrap(err, "couldn't remove local environment")
		}
		fmt.Printf("Cloning environment %s to %s...\n", remote, env.FS.Path())
		if gitRepo, err = git.Clone(remote, env.FS.Path()); err != nil {
			return errors.Wrapf(
				err, "couldn't clone environment %s at release %s to %s", remote, release, env.FS.Path(),
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
	env, err := getEnv(c.String("workspace"))
	if err != nil {
		return err
	}

	fmt.Println("Fetching updates...")
	updated, err := git.Fetch(env.FS.Path())
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
	env, err := getEnv(c.String("workspace"))
	if err != nil {
		return err
	}

	fmt.Println("Attempting to fast-forward the local environment...")
	updated, err := git.Pull(env.FS.Path())
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
	env, err := getEnv(c.String("workspace"))
	if err != nil {
		return err
	}

	fmt.Printf("Removing local environment from workspace...\n")
	// TODO: return an error if there are uncommitted or unpushed changes to be removed - in which
	// case require a --force flag
	return errors.Wrap(env.Remove(), "couldn't remove local environment")
}

// show

func showAction(c *cli.Context) error {
	env, err := getEnv(c.String("workspace"))
	if err != nil {
		return err
	}
	return fcli.PrintEnvInfo(0, env)
}

// check

func checkAction(c *cli.Context) error {
	env, cache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}

	if err := fcli.CheckEnv(0, env, cache); err != nil {
		return err
	}
	return nil
}

// plan

func planAction(c *cli.Context) error {
	env, cache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}

	if err := fcli.PlanEnv(0, env, cache); err != nil {
		return errors.Wrap(
			err, "couldn't deploy local environment (have you run `forklift env cache` recently?)",
		)
	}
	return nil
}

// apply

func applyAction(c *cli.Context) error {
	env, cache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}

	if err := fcli.ApplyEnv(0, env, cache); err != nil {
		return errors.Wrap(
			err, "couldn't deploy local environment (have you run `forklift env cache` recently?)",
		)
	}
	fmt.Println()
	fmt.Println("Done!")
	return nil
}
