package plt

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/pkg/core"
	"github.com/PlanktoScope/forklift/pkg/structures"
)

type workspaceCaches struct {
	p *forklift.LayeredPalletCache
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

type processingOptions struct {
	requirePalletCache   bool
	requireRepoCache     bool
	requireDownloadCache bool
	enableOverrides      bool
	merge                bool
}

func processFullBaseArgs(
	c *cli.Context, opts processingOptions,
) (plt *forklift.FSPallet, caches workspaceCaches, err error) {
	if plt, err = getShallowPallet(c.String("cwd")); err != nil {
		return nil, workspaceCaches{}, err
	}
	wpath := c.String("workspace")
	caches.p = &forklift.LayeredPalletCache{}
	if caches.p.Underlay, err = fcli.GetPalletCache(
		wpath, plt, opts.requirePalletCache || opts.merge,
	); err != nil {
		return nil, workspaceCaches{}, err
	}
	if opts.enableOverrides {
		if caches.p, err = overlayPalletCacheOverrides(
			caches.p.Underlay, c.StringSlice("plts"), plt,
		); err != nil {
			return nil, workspaceCaches{}, err
		}
	}
	if opts.merge {
		if plt, err = forklift.MergeFSPallet(plt, caches.p, nil); err != nil {
			return nil, workspaceCaches{}, errors.Wrap(
				err, "couldn't merge development pallet with file imports from any pallets required by it",
			)
		}
	}
	if caches.r, _, err = fcli.GetRepoCache(wpath, plt, opts.requireRepoCache); err != nil {
		return nil, workspaceCaches{}, err
	}
	if opts.enableOverrides {
		if caches.r, err = overlayRepoCacheOverrides(
			caches.r, c.StringSlice("repos"), plt,
		); err != nil {
			return nil, workspaceCaches{}, err
		}
	}
	if caches.d, err = fcli.GetDownloadCache(wpath, opts.requireDownloadCache); err != nil {
		return nil, workspaceCaches{}, err
	}
	return plt, caches, nil
}

func getShallowPallet(cwdPath string) (plt *forklift.FSPallet, err error) {
	if plt, err = forklift.LoadFSPalletContaining(cwdPath); err != nil {
		return nil, errors.Wrapf(
			err, "The current working directory %s is not part of a Forklift pallet", cwdPath,
		)
	}
	return plt, nil
}

func overlayPalletCacheOverrides(
	underlay forklift.PathedPalletCache, pallets []string, plt *forklift.FSPallet,
) (palletCache *forklift.LayeredPalletCache, err error) {
	palletCache = &forklift.LayeredPalletCache{
		Underlay: underlay,
	}
	replacementPallets, err := loadReplacementPallets(pallets)
	if err != nil {
		return nil, err
	}
	override, err := forklift.NewPalletOverrideCache(replacementPallets, nil)
	if err != nil {
		return nil, err
	}
	if err = setPalletOverrideCacheVersions(plt, override); err != nil {
		return nil, err
	}
	palletCache.Overlay = override
	return palletCache, nil
}

func loadReplacementPallets(fsPaths []string) (replacements []*forklift.FSPallet, err error) {
	for _, path := range fsPaths {
		replacementPath, err := filepath.Abs(path)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't convert '%s' into an absolute path", path)
		}
		if !forklift.DirExists(replacementPath) {
			return nil, errors.Errorf("couldn't find pallet replacement path %s", replacementPath)
		}
		externalPallets, err := forklift.LoadFSPallets(forklift.DirFS(replacementPath), "**")
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't list replacement pallets in path %s", replacementPath)
		}
		if len(externalPallets) == 0 {
			return nil, errors.Errorf("no replacement pallets found in path %s", replacementPath)
		}
		replacements = append(replacements, externalPallets...)
	}
	return replacements, nil
}

func setPalletOverrideCacheVersions(
	plt *forklift.FSPallet, overrideCache *forklift.PalletOverrideCache,
) error {
	reqs, err := plt.LoadFSPalletReqs("**")
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify pallet requirements specified by pallet %s", plt.FS.Path(),
		)
	}
	palletVersions := make(map[string]structures.Set[string])
	for _, req := range reqs {
		palletPath := req.Path()
		version := req.VersionLock.Version
		if _, ok := palletVersions[palletPath]; !ok {
			palletVersions[palletPath] = make(structures.Set[string])
		}
		palletVersions[palletPath].Add(version)
	}

	for palletPath, versions := range palletVersions {
		overrideCache.SetVersions(palletPath, versions)
	}
	return nil
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
	if err = setRepoOverrideCacheVersions(plt, override); err != nil {
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
		externalRepos, err := core.LoadFSRepos(forklift.DirFS(replacementPath), "**")
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

func setRepoOverrideCacheVersions(
	plt *forklift.FSPallet, overrideCache *forklift.RepoOverrideCache,
) error {
	reqs, err := plt.LoadFSRepoReqs("**")
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify repo requirements specified by pallet %s", plt.FS.Path(),
		)
	}
	repoVersions := make(map[string]structures.Set[string])
	for _, req := range reqs {
		repoPath := req.Path()
		version := req.VersionLock.Version
		if _, ok := repoVersions[repoPath]; !ok {
			repoVersions[repoPath] = make(structures.Set[string])
		}
		repoVersions[repoPath].Add(version)
	}

	for repoPath, versions := range repoVersions {
		overrideCache.SetVersions(repoPath, versions)
	}
	return nil
}

// cache-all

func cacheAllAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			enableOverrides: true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckPltCompat(plt, versions.Core(), c.Bool("ignore-tool-version")); err != nil {
			return err
		}

		if err = fcli.CacheAllReqs(
			0, plt, caches.p, caches.r, caches.d, c.Bool("include-disabled"), c.Bool("parallel"),
		); err != nil {
			return err
		}
		fmt.Println("Done!")
		return nil
	}
}

// show

func showAction(c *cli.Context) error {
	plt, err := getShallowPallet(c.String("cwd"))
	if err != nil {
		return err
	}
	return fcli.PrintPalletInfo(0, plt)
}

// check

func checkAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requirePalletCache: true,
			requireRepoCache:   true,
			enableOverrides:    true,
			merge:              true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, caches.r, versions.Core(), c.Bool("ignore-tool-version"),
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
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requirePalletCache: true,
			requireRepoCache:   true,
			enableOverrides:    true,
			merge:              true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, caches.r, versions.Core(), c.Bool("ignore-tool-version"),
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
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			enableOverrides: true,
		})
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
			0, plt, stageStore, caches.staging(), c.String("exports"),
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
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			enableOverrides: true,
		})
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
			0, plt, stageStore, caches.staging(), c.String("exports"),
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
		plt, err := getShallowPallet(c.String("cwd"))
		if err != nil {
			return err
		}
		workspace, err := forklift.LoadWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}

		if err = fcli.CheckPltCompat(plt, versions.Core(), c.Bool("ignore-tool-version")); err != nil {
			return err
		}

		fmt.Printf("Downloading pallets specified by the development pallet...\n")
		changed, err := fcli.DownloadRequiredPallets(0, plt, workspace.GetPalletCachePath())
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
	plt, err := getShallowPallet(c.String("cwd"))
	if err != nil {
		return err
	}
	return fcli.PrintRequiredPallets(0, plt)
}

// show-plt

func showPltAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		requirePalletCache: true,
		enableOverrides:    true,
	})
	if err != nil {
		return err
	}

	return fcli.PrintRequiredPalletInfo(0, plt, caches.p, c.Args().First())
}

// add-plt

func addPltAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, err := getShallowPallet(c.String("cwd"))
		if err != nil {
			return err
		}
		workspace, err := forklift.LoadWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}

		if err = fcli.CheckPltCompat(plt, versions.Core(), c.Bool("ignore-tool-version")); err != nil {
			return err
		}

		if err = fcli.AddPalletReqs(
			0, plt, workspace.GetPalletCachePath(), c.Args().Slice(),
		); err != nil {
			return err
		}
		if !c.Bool("no-cache-req") {
			plt, caches, err := processFullBaseArgs(c, processingOptions{
				enableOverrides: true,
			})
			if err != nil {
				return err
			}
			if _, _, err = fcli.CacheStagingReqs(
				0, plt, caches.p, caches.r, caches.d, false, c.Bool("parallel"),
			); err != nil {
				return err
			}
			// TODO: check version compatibility between the pallet and the added pallet!
		}
		fmt.Println("Done!")
		return nil
	}
}

// rm-plt

func rmPltAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, err := getShallowPallet(c.String("cwd"))
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

// ls-plt-file

func lsPltFileAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		enableOverrides: true,
	})
	if err != nil {
		return err
	}

	plt, err = fcli.GetRequiredPallet(plt, caches.p, c.Args().First())
	if err != nil {
		return nil
	}
	paths, err := fcli.ListPalletFiles(plt, c.Args().Get(1))
	if err != nil {
		return err
	}
	for _, p := range paths {
		fmt.Println(p)
	}
	return nil
}

// locate-plt-file

func locatePltFileAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		enableOverrides: true,
	})
	if err != nil {
		return err
	}

	plt, err = fcli.GetRequiredPallet(plt, caches.p, c.Args().First())
	if err != nil {
		return nil
	}
	location, err := fcli.GetFileLocation(plt, c.Args().Get(1))
	if err != nil {
		return err
	}
	fmt.Println(location)
	return nil
}

// show-plt-file

func showPltFileAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		enableOverrides: true,
	})
	if err != nil {
		return err
	}

	plt, err = fcli.GetRequiredPallet(plt, caches.p, c.Args().First())
	if err != nil {
		return nil
	}
	return fcli.PrintFile(plt, c.Args().Get(1))
}
