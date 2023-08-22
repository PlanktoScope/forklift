package env

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/pkg/core"
)

func processFullBaseArgs(c *cli.Context, ensureCache bool) (
	env *forklift.FSEnv, cache *forklift.LayeredRepoCache, override *forklift.RepoOverrideCache,
	err error,
) {
	if env, err = getEnv(c.String("cwd")); err != nil {
		return nil, nil, nil, err
	}
	if cache, override, err = getCache(
		c.String("workspace"), c.StringSlice("repos"), ensureCache,
	); err != nil {
		return nil, nil, nil, err
	}
	return env, cache, override, nil
}

func getEnv(cwdPath string) (env *forklift.FSEnv, err error) {
	if env, err = forklift.LoadFSEnvContaining(cwdPath); err != nil {
		return nil, errors.Wrapf(
			err, "The current working directory %s is not part of a Forklift environment", cwdPath,
		)
	}
	return env, nil
}

func getCache(
	wpath string, repos []string, ensureCache bool,
) (*forklift.LayeredRepoCache, *forklift.RepoOverrideCache, error) {
	cache := &forklift.LayeredRepoCache{}
	replacementRepos, err := loadReplacementRepos(repos)
	if err != nil {
		return nil, nil, err
	}
	override, err := forklift.NewRepoOverrideCache(replacementRepos, nil)
	if err != nil {
		return nil, nil, err
	}
	cache.Overlay = override

	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, nil, err
	}
	fsCache, err := workspace.GetRepoCache()
	if err != nil && len(repos) == 0 {
		return nil, nil, err
	}
	cache.Underlay = fsCache

	if ensureCache && !fsCache.Exists() {
		return nil, nil, errors.New(
			"you first need to cache the repos specified by your environment with " +
				"`forklift dev env cache-repos`",
		)
	}
	return cache, override, nil
}

func loadReplacementRepos(fsPaths []string) (replacements []*core.FSRepo, err error) {
	for _, path := range fsPaths {
		replacementPath, err := filepath.Abs(path)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't convert '%s' into an absolute path", path)
		}
		if !forklift.Exists(replacementPath) {
			return nil, errors.Errorf("couldn't find repo replacement path %s", replacementPath)
		}
		externalRepos, err := core.LoadFSRepos(
			core.AttachPath(os.DirFS(replacementPath), replacementPath), "**",
		)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't list replacement repos in path %s", replacementPath)
		}
		if len(externalRepos) == 0 {
			return nil, errors.Errorf("no replacement repos found in path %s", replacementPath)
		}
		replacements = append(replacements, externalRepos...)
	}
	return replacements, nil
}

func setOverrideCacheVersions(
	env *forklift.FSEnv, overrideCache *forklift.RepoOverrideCache,
) error {
	reqs, err := env.LoadFSRepoReqs("**")
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify repo requirements specified by environment %s", env.FS.Path(),
		)
	}
	repoVersions := make(map[string]map[string]struct{})
	for _, req := range reqs {
		repoPath := req.Path()
		version := req.VersionLock.Version
		if _, ok := repoVersions[repoPath]; !ok {
			repoVersions[repoPath] = make(map[string]struct{})
		}
		repoVersions[repoPath][version] = struct{}{}
	}

	for repoPath, versions := range repoVersions {
		overrideCache.SetVersions(repoPath, versions)
	}
	return nil
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
	env, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(env, overrideCache); err != nil {
		return err
	}

	if err := fcli.CheckEnv(0, env, cache); err != nil {
		return err
	}
	return nil
}

// plan

func planAction(c *cli.Context) error {
	env, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(env, overrideCache); err != nil {
		return err
	}

	if err := fcli.PlanEnv(0, env, cache); err != nil {
		return err
	}
	return nil
}

// apply

func applyAction(c *cli.Context) error {
	env, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(env, overrideCache); err != nil {
		return err
	}

	if err := fcli.ApplyEnv(0, env, cache); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Done!")
	return nil
}
