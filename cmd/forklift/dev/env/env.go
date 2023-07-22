package env

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

func processFullBaseArgs(c *cli.Context, ensureCache bool) (
	env *forklift.FSEnv, cache *forklift.FSCache, replacementRepos map[string]*pallets.FSRepo,
	err error,
) {
	if env, err = getEnv(c.String("cwd")); err != nil {
		return nil, nil, nil, err
	}
	if replacementRepos, err = loadReplacementRepos(c.StringSlice("repo")); err != nil {
		return nil, nil, nil, err
	}
	if cache, err = getCache(c.String("workspace")); err != nil && len(replacementRepos) == 0 {
		return nil, nil, nil, err
	}
	if (ensureCache || len(replacementRepos) == 0) && !cache.Exists() {
		return nil, nil, nil, errors.New(
			"you first need to cache the repos specified by your environment with " +
				"`forklift dev env cache-repo`",
		)
	}
	return env, cache, replacementRepos, nil
}

func getEnv(cwdPath string) (env *forklift.FSEnv, err error) {
	envPath, err := forklift.FindParentEnv(cwdPath)
	if err != nil {
		return nil, errors.Wrapf(
			err, "The current working directory %s is not part of a Forklift environment.", cwdPath,
		)
	}
	if env, err = forklift.LoadFSEnv(
		pallets.AttachPath(os.DirFS(envPath), envPath), ".",
	); err != nil {
		return nil, errors.Wrap(err, "couldn't load environment")
	}
	return env, nil
}

func loadReplacementRepos(fsPaths []string) (repos map[string]*pallets.FSRepo, err error) {
	repos = make(map[string]*pallets.FSRepo)
	for _, path := range fsPaths {
		replacementPath, err := filepath.Abs(path)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't convert '%s' into an absolute path", path)
		}
		if !forklift.Exists(replacementPath) {
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

func getCache(wpath string) (*forklift.FSCache, error) {
	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	cache, err := workspace.GetCache()
	if err != nil {
		return nil, err
	}
	return cache, nil
}

// show

func showAction(c *cli.Context) error {
	env, err := getEnv(c.String("cwd"))
	if err != nil {
		return err
	}
	return fcli.PrintEnvInfo(0, env)
}

// check

func checkAction(c *cli.Context) error {
	env, cache, replacementRepos, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}

	if err := fcli.CheckEnv(0, env, cache, replacementRepos); err != nil {
		return err
	}
	return nil
}

// plan

func planAction(c *cli.Context) error {
	env, cache, replacementRepos, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}

	if err := fcli.PlanEnv(0, env, cache, replacementRepos); err != nil {
		return err
	}
	return nil
}

// apply

func applyAction(c *cli.Context) error {
	env, cache, replacementRepos, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}

	if err := fcli.ApplyEnv(0, env, cache, replacementRepos); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Done!")
	return nil
}
