package env

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/internal/app/forklift/dev"
	"github.com/PlanktoScope/forklift/internal/app/forklift/workspace"
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

func processFullBaseArgs(c *cli.Context) (
	envPath string, workspacePath string, replacementRepos map[string]pallets.FSRepo,
	err error,
) {
	if envPath, err = dev.FindParentEnv(c.String("cwd")); err != nil {
		return "", "", nil, errors.Wrap(
			err, "The current working directory is not part of a Forklift environment.",
		)
	}
	if replacementRepos, err = loadReplacementRepos(c.StringSlice("repo")); err != nil {
		return "", "", nil, err
	}
	workspacePath = c.String("workspace")
	if !workspace.Exists(workspace.CachePath(workspacePath)) && len(replacementRepos) == 0 {
		return "", "", nil, errors.Errorf(
			"you first need to cache the repos specified by your environment with " +
				"`forklift dev env cache-repo`",
		)
	}
	return envPath, workspacePath, replacementRepos, nil
}

func loadReplacementRepos(fsPaths []string) (repos map[string]pallets.FSRepo, err error) {
	repos = make(map[string]pallets.FSRepo)
	for _, path := range fsPaths {
		replacementPath, err := filepath.Abs(path)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't convert '%s' into an absolute path", path)
		}
		if !workspace.Exists(replacementPath) {
			return nil, errors.Errorf("couldn't find repository replacement path %s", replacementPath)
		}
		externalRepos, err := forklift.ListExternalRepos(
			pallets.AttachPath(os.DirFS(replacementPath), replacementPath),
		)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't list replacement repos in path %s", replacementPath)
		}
		if len(externalRepos) == 0 {
			return nil, errors.Errorf("no replacement repos found in path %s", replacementPath)
		}
		for _, repo := range externalRepos {
			repos[repo.Path()] = repo
		}
	}
	return repos, nil
}

// show

func showAction(c *cli.Context) error {
	envPath, err := dev.FindParentEnv(c.String("cwd"))
	if err != nil {
		return errors.Wrap(err, "The current working directory is not part of a Forklift environment.")
	}
	return fcli.PrintEnvInfo(0, envPath)
}

// check

func checkAction(c *cli.Context) error {
	envPath, wpath, replacementRepos, err := processFullBaseArgs(c)
	if err != nil {
		return nil
	}

	if err := fcli.CheckEnv(0, envPath, workspace.CachePath(wpath), replacementRepos); err != nil {
		return err
	}
	return nil
}

// plan

func planAction(c *cli.Context) error {
	envPath, wpath, replacementRepos, err := processFullBaseArgs(c)
	if err != nil {
		return nil
	}

	if err := fcli.PlanEnv(0, envPath, workspace.CachePath(wpath), replacementRepos); err != nil {
		return err
	}
	return nil
}

// apply

func applyAction(c *cli.Context) error {
	envPath, wpath, replacementRepos, err := processFullBaseArgs(c)
	if err != nil {
		return nil
	}

	if err := fcli.ApplyEnv(0, envPath, workspace.CachePath(wpath), replacementRepos); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Done!")
	return nil
}
