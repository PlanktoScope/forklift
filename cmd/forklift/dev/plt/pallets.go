package plt

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
	pallet *forklift.FSPallet, cache *forklift.LayeredRepoCache, override *forklift.RepoOverrideCache,
	err error,
) {
	if pallet, err = getPallet(c.String("cwd")); err != nil {
		return nil, nil, nil, err
	}
	if cache, override, err = getCache(
		c.String("workspace"), c.StringSlice("repos"), ensureCache,
	); err != nil {
		return nil, nil, nil, err
	}
	return pallet, cache, override, nil
}

func getPallet(cwdPath string) (pallet *forklift.FSPallet, err error) {
	if pallet, err = forklift.LoadFSPalletContaining(cwdPath); err != nil {
		return nil, errors.Wrapf(
			err, "The current working directory %s is not part of a Forklift pallet", cwdPath,
		)
	}
	return pallet, nil
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
			"you first need to cache the repos specified by your pallet with " +
				"`forklift dev plt cache-repos`",
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
	pallet *forklift.FSPallet, overrideCache *forklift.RepoOverrideCache,
) error {
	reqs, err := pallet.LoadFSRepoReqs("**")
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify repo requirements specified by pallet %s", pallet.FS.Path(),
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
	pallet, err := getPallet(c.String("cwd"))
	if err != nil {
		return err
	}
	return fcli.PrintPalletInfo(0, pallet)
}

// check

func checkAction(c *cli.Context) error {
	pallet, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(pallet, overrideCache); err != nil {
		return err
	}

	if err := fcli.CheckPallet(0, pallet, cache); err != nil {
		return err
	}
	return nil
}

// plan

func planAction(c *cli.Context) error {
	pallet, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(pallet, overrideCache); err != nil {
		return err
	}

	if err := fcli.PlanPallet(0, pallet, cache); err != nil {
		return err
	}
	return nil
}

// apply

func applyAction(c *cli.Context) error {
	pallet, cache, overrideCache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}
	if err = setOverrideCacheVersions(pallet, overrideCache); err != nil {
		return err
	}

	if err := fcli.ApplyPallet(0, pallet, cache); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Done!")
	return nil
}
