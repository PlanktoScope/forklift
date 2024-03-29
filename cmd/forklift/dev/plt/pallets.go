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

func processFullBaseArgs(c *cli.Context, ensureCache, enableOverrides bool) (
	pallet *forklift.FSPallet, cache *forklift.LayeredRepoCache, err error,
) {
	if pallet, err = getPallet(c.String("cwd")); err != nil {
		return nil, nil, err
	}
	if cache, _, err = fcli.GetRepoCache(c.String("workspace"), pallet, ensureCache); err != nil {
		return nil, nil, err
	}
	if !enableOverrides {
		return pallet, cache, nil
	}
	if cache, err = overlayCacheOverrides(cache, c.StringSlice("repos"), pallet); err != nil {
		return nil, nil, err
	}
	return pallet, cache, nil
}

func getPallet(cwdPath string) (pallet *forklift.FSPallet, err error) {
	if pallet, err = forklift.LoadFSPalletContaining(cwdPath); err != nil {
		return nil, errors.Wrapf(
			err, "The current working directory %s is not part of a Forklift pallet", cwdPath,
		)
	}
	return pallet, nil
}

func overlayCacheOverrides(
	underlay forklift.PathedRepoCache, repos []string, pallet *forklift.FSPallet,
) (cache *forklift.LayeredRepoCache, err error) {
	cache = &forklift.LayeredRepoCache{
		Underlay: underlay,
	}
	replacementRepos, err := loadReplacementRepos(repos)
	if err != nil {
		return nil, err
	}
	override, err := forklift.NewRepoOverrideCache(replacementRepos, nil)
	if err != nil {
		return nil, err
	}
	if err = setOverrideCacheVersions(pallet, override); err != nil {
		return nil, err
	}
	cache.Overlay = override
	return cache, nil
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

// cache-all

func cacheAllAction(toolVersion, repoMinVersion, palletMinVersion string) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, cache, err := processFullBaseArgs(c, false, false)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, cache, toolVersion, repoMinVersion, palletMinVersion, c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		changed, err := fcli.CacheAllRequirements(
			pallet, cache.Underlay.Path(), cache, c.Bool("include-disabled"), c.Bool("parallel"),
		)
		if err != nil {
			return err
		}
		if !changed {
			fmt.Println("Done! No further actions are needed at this time.")
			return nil
		}
		fmt.Println("Done! Next, you might want to run `sudo -E forklift dev plt apply`.")
		return nil
	}
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

func checkAction(toolVersion, repoMinVersion, palletMinVersion string) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, cache, err := processFullBaseArgs(c, true, true)
		if err != nil {
			return err
		}
		if err = fcli.CheckCompatibility(
			pallet, cache, toolVersion, repoMinVersion, palletMinVersion, c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if _, _, err := fcli.CheckPallet(0, pallet, cache); err != nil {
			return err
		}
		return nil
	}
}

// plan

func planAction(toolVersion, repoMinVersion, palletMinVersion string) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, cache, err := processFullBaseArgs(c, true, true)
		if err != nil {
			return err
		}
		if err = fcli.CheckCompatibility(
			pallet, cache, toolVersion, repoMinVersion, palletMinVersion, c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if _, _, err = fcli.PlanPallet(0, pallet, cache, c.Bool("parallel")); err != nil {
			return err
		}
		return nil
	}
}

// apply

func applyAction(toolVersion, repoMinVersion, palletMinVersion string) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, cache, err := processFullBaseArgs(c, true, true)
		if err != nil {
			return err
		}
		if err = fcli.CheckCompatibility(
			pallet, cache, toolVersion, repoMinVersion, palletMinVersion, c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if err := fcli.ApplyPallet(0, pallet, cache, c.Bool("parallel")); err != nil {
			return err
		}
		fmt.Println()
		fmt.Println("Done!")
		return nil
	}
}
