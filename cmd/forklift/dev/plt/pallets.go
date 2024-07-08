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

type workspaceCaches struct {
	p *forklift.FSPalletCache
	r *forklift.LayeredRepoCache
	d *forklift.FSDownloadCache
}

func (c workspaceCaches) staging() fcli.StagingCaches {
	return fcli.StagingCaches{
		Pallets:   c.p,
		Repos:     c.r,
		Downloads: c.d,
	}
}

func processFullBaseArgs(
	c *cli.Context, ensureCache, enableOverrides bool,
) (plt *forklift.FSPallet, caches workspaceCaches, err error) {
	if plt, err = getPallet(c.String("cwd")); err != nil {
		return nil, workspaceCaches{}, err
	}
	if caches.d, err = fcli.GetDownloadCache(c.String("workspace"), ensureCache); err != nil {
		return nil, workspaceCaches{}, err
	}
	if caches.p, err = fcli.GetPalletCache(c.String("workspace"), plt, ensureCache); err != nil {
		return nil, workspaceCaches{}, err
	}
	if caches.r, _, err = fcli.GetRepoCache(c.String("workspace"), plt, ensureCache); err != nil {
		return nil, workspaceCaches{}, err
	}
	if !enableOverrides {
		return plt, caches, nil
	}
	// FIXME: add overlayPalletCacheOverrides
	if caches.r, err = overlayRepoCacheOverrides(caches.r, c.StringSlice("repos"), plt); err != nil {
		return nil, workspaceCaches{}, err
	}
	return plt, caches, nil
}

func getPallet(cwdPath string) (plt *forklift.FSPallet, err error) {
	if plt, err = forklift.LoadFSPalletContaining(cwdPath); err != nil {
		return nil, errors.Wrapf(
			err, "The current working directory %s is not part of a Forklift pallet", cwdPath,
		)
	}
	return plt, nil
}

func overlayRepoCacheOverrides(
	underlay forklift.PathedRepoCache, repos []string, plt *forklift.FSPallet,
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
	if err = setOverrideCacheVersions(plt, override); err != nil {
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
	plt *forklift.FSPallet, overrideCache *forklift.RepoOverrideCache,
) error {
	reqs, err := plt.LoadFSRepoReqs("**")
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify repo requirements specified by pallet %s", plt.FS.Path(),
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
		plt, caches, err := processFullBaseArgs(c, true, true)
		if err != nil {
			return err
		}
		if err = fcli.CheckPltCompat(plt, versions.Core(), c.Bool("ignore-tool-version")); err != nil {
			return err
		}

		if err = fcli.CacheAllReqs(
			0, plt, caches.r.Underlay.Path(), caches.p.Path(), caches.r, caches.d,
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
	plt, err := getPallet(c.String("cwd"))
	if err != nil {
		return err
	}
	return fcli.PrintPalletInfo(0, plt)
}

// check

func checkAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, true, true)
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.r, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if _, _, err := fcli.Check(0, plt, caches.r); err != nil {
			return err
		}
		return nil
	}
}

// plan

func planAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, true, true)
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.r, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if _, _, err = fcli.Plan(0, plt, caches.r, c.Bool("parallel")); err != nil {
			return err
		}
		return nil
	}
}

// stage

func stageAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, true, true)
		if err != nil {
			return err
		}
		// Note: we cannot guarantee that all requirements are cached, so we don't check their versions
		// here; fcli.StagePallet will do those checks for us.
		if err = fcli.CheckPltCompat(plt, versions.Core(), c.Bool("ignore-tool-version")); err != nil {
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
			plt, stageStore, caches.staging(), c.String("exports"),
			versions.Staging, c.Bool("no-cache-img"), c.Bool("parallel"), c.Bool("ignore-tool-version"),
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
		plt, caches, err := processFullBaseArgs(c, true, true)
		if err != nil {
			return err
		}
		// Note: we cannot guarantee that all requirements are cached, so we don't check their versions
		// here; fcli.StagePallet will do those checks for us.
		if err = fcli.CheckPltCompat(plt, versions.Core(), c.Bool("ignore-tool-version")); err != nil {
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
			plt, stageStore, caches.staging(), c.String("exports"),
			versions.Staging, false, c.Bool("parallel"), c.Bool("ignore-tool-version"),
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

// cache-plt

func cachePltAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, false, false)
		if err != nil {
			return err
		}
		if err = fcli.CheckPltCompat(plt, versions.Core(), c.Bool("ignore-tool-version")); err != nil {
			return err
		}

		fmt.Printf("Downloading pallets specified by the development pallet...\n")
		changed, err := fcli.DownloadRequiredPallets(0, plt, caches.p.Path())
		if err != nil {
			return err
		}
		if !changed {
			fmt.Println("Done! No further actions are needed at this time.")
			return nil
		}

		// TODO: warn if any downloaded pallet doesn't appear to be an actual pallet, or if any pallet's
		// forklift version is incompatible or ahead of the pallet version
		fmt.Println("Done!")
		return nil
	}
}

// ls-plt

func lsPltAction(c *cli.Context) error {
	plt, err := getPallet(c.String("cwd"))
	if err != nil {
		return err
	}
	return fcli.PrintRequiredPallets(0, plt)
}

// show-plt

func showPltAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, true, true)
	if err != nil {
		return err
	}

	return fcli.PrintRequiredPalletInfo(0, plt, caches.p, c.Args().First())
}

// add-plt

func addPltAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, false, false)
		if err != nil {
			return err
		}
		if err = fcli.CheckPltCompat(plt, versions.Core(), c.Bool("ignore-tool-version")); err != nil {
			return err
		}

		if err = fcli.AddPalletReqs(0, plt, caches.p.Path(), c.Args().Slice()); err != nil {
			return err
		}
		if !c.Bool("no-cache-req") {
			if err = fcli.CacheStagingReqs(
				0, plt, caches.r.Path(), caches.p.Path(), caches.r, caches.d, false, c.Bool("parallel"),
			); err != nil {
				return err
			}
		}
		fmt.Println("Done!")
		return nil
	}
}

// rm-plt

func rmPltAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, _, err := processFullBaseArgs(c, false, false)
		if err != nil {
			return err
		}
		if err = fcli.CheckPltCompat(plt, versions.Core(), c.Bool("ignore-tool-version")); err != nil {
			return err
		}

		if err = fcli.RemovePalletReqs(0, plt, c.Args().Slice(), c.Bool("force")); err != nil {
			return err
		}
		fmt.Println("Done!")
		return nil
	}
}
