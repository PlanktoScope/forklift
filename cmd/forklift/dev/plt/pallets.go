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
	pallet *forklift.FSPallet,
	repoCache *forklift.LayeredRepoCache, dlCache *forklift.FSDownloadCache, err error,
) {
	if pallet, err = getPallet(c.String("cwd")); err != nil {
		return nil, nil, nil, err
	}
	if dlCache, err = fcli.GetDlCache(c.String("workspace"), ensureCache); err != nil {
		return nil, nil, nil, err
	}
	if repoCache, _, err = fcli.GetRepoCache(c.String("workspace"), pallet, ensureCache); err != nil {
		return nil, nil, nil, err
	}
	if !enableOverrides {
		return pallet, repoCache, dlCache, nil
	}
	if repoCache, err = overlayCacheOverrides(repoCache, c.StringSlice("repos"), pallet); err != nil {
		return nil, nil, nil, err
	}
	return pallet, repoCache, dlCache, nil
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
) (repoCache *forklift.LayeredRepoCache, err error) {
	repoCache = &forklift.LayeredRepoCache{
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
	repoCache.Overlay = override
	return repoCache, nil
}

func loadReplacementRepos(fsPaths []string) (replacements []*core.FSRepo, err error) {
	for _, path := range fsPaths {
		replacementPath, err := filepath.Abs(path)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't convert '%s' into an absolute path", path)
		}
		if !forklift.DirExists(replacementPath) {
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

func cacheAllAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, repoCache, dlCache, err := processFullBaseArgs(c, true, true)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if err = fcli.CacheAllRequirements(
			0, pallet, repoCache.Underlay.Path(), repoCache, dlCache,
			c.Bool("include-disabled"), c.Bool("parallel"),
		); err != nil {
			return err
		}
		fmt.Println("Done!")
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

func checkAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, repoCache, _, err := processFullBaseArgs(c, true, true)
		if err != nil {
			return err
		}
		if err = fcli.CheckCompatibility(
			pallet, repoCache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if _, _, err := fcli.Check(0, pallet, repoCache); err != nil {
			return err
		}
		return nil
	}
}

// plan

func planAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, repoCache, _, err := processFullBaseArgs(c, true, true)
		if err != nil {
			return err
		}
		if err = fcli.CheckCompatibility(
			pallet, repoCache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if _, _, err = fcli.Plan(0, pallet, repoCache, c.Bool("parallel")); err != nil {
			return err
		}
		return nil
	}
}

// stage

func stageAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, repoCache, dlCache, err := processFullBaseArgs(c, true, true)
		if err != nil {
			return err
		}
		// Note: we cannot guarantee that all requirements are cached, so we don't check their versions
		// here; fcli.StagePallet will do those checks for us.
		if err = fcli.CheckShallowCompatibility(
			pallet, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		workspace, err := forklift.LoadWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}
		stageStore, err := fcli.GetStageStore(
			workspace, c.String("stage-store"), versions.NewStageStore,
		)
		if err != nil {
			return err
		}
		if _, err = fcli.StagePallet(
			pallet, stageStore, repoCache, dlCache, c.String("exports"),
			versions.Versions, c.Bool("no-cache-img"), c.Bool("parallel"), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}
		fmt.Println(
			"Done! To apply the staged pallet, you may need to reboot or run " +
				"`forklift stage apply` (or `sudo -E forklift stage apply` if you need sudo for Docker).",
		)
		return nil
	}
}

// apply

func applyAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, repoCache, dlCache, err := processFullBaseArgs(c, true, true)
		if err != nil {
			return err
		}
		// Note: we cannot guarantee that all requirements are cached, so we don't check their versions
		// here; fcli.StagePallet will do those checks for us.
		if err = fcli.CheckShallowCompatibility(
			pallet, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}
		workspace, err := forklift.LoadWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}

		stageStore, err := fcli.GetStageStore(
			workspace, c.String("stage-store"), versions.NewStageStore,
		)
		if err != nil {
			return err
		}
		index, err := fcli.StagePallet(
			pallet, stageStore, repoCache, dlCache, c.String("exports"),
			versions.Versions, false, c.Bool("parallel"), c.Bool("ignore-tool-version"),
		)
		if err != nil {
			return errors.Wrap(err, "couldn't stage pallet to be applied immediately")
		}

		bundle, err := stageStore.LoadFSBundle(index)
		if err != nil {
			return errors.Wrapf(err, "couldn't load staged pallet bundle %d", index)
		}
		if err = fcli.ApplyNextOrCurrentBundle(0, stageStore, bundle, c.Bool("parallel")); err != nil {
			return errors.Wrapf(err, "couldn't apply staged pallet bundle %d", index)
		}
		fmt.Println("Done! You may need to reboot for some changes to take effect.")
		return nil
	}
}
