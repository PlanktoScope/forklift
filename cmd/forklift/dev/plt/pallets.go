package plt

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/forklift-run/forklift/internal/app/forklift"
	fcli "github.com/forklift-run/forklift/internal/app/forklift/cli"
	"github.com/forklift-run/forklift/pkg/caching"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
	"github.com/forklift-run/forklift/pkg/structures"
	fws "github.com/forklift-run/forklift/pkg/workspaces"
)

type workspaceCaches struct {
	m  *caching.FSMirrorCache
	p  *caching.LayeredPalletCache
	pp *caching.LayeredPalletCache
	d  *caching.FSDownloadCache
}

func (c workspaceCaches) staging() fcli.StagingCaches {
	return fcli.StagingCaches{
		Mirrors:   c.m,
		Pallets:   c.p,
		Downloads: c.d,
	}
}

type processingOptions struct {
	requirePalletCache   bool
	requireDownloadCache bool
	enableOverrides      bool
	merge                bool
}

func processFullBaseArgs(
	c *cli.Context, opts processingOptions,
) (plt *fplt.FSPallet, caches workspaceCaches, err error) {
	if plt, err = getShallowPallet(c.String("cwd")); err != nil {
		return nil, workspaceCaches{}, err
	}
	wpath := c.String("workspace")
	workspace, err := fws.LoadWorkspace(wpath)
	if err != nil {
		return nil, workspaceCaches{}, err
	}
	if caches.m, err = workspace.GetMirrorCache(); err != nil {
		return nil, workspaceCaches{}, err
	}
	caches.p = &caching.LayeredPalletCache{}
	if caches.p.Underlay, err = forklift.GetPalletCache(
		wpath, plt, opts.requirePalletCache || opts.merge,
	); err != nil {
		return nil, workspaceCaches{}, err
	}
	if opts.enableOverrides {
		if caches.p, err = overlayPalletCacheOverrides(
			caches.p.Underlay, c.StringSlice("plt"), plt,
		); err != nil {
			return nil, workspaceCaches{}, err
		}
	}
	if opts.merge {
		if plt, err = fplt.MergeFSPallet(plt, caches.p, nil); err != nil {
			return nil, workspaceCaches{}, errors.Wrap(
				err, "couldn't merge development pallet with file imports from any pallets required by it",
			)
		}
	}
	if caches.pp, err = fcli.MakeOverlayCache(plt, caches.p); err != nil {
		return nil, workspaceCaches{}, errors.Wrap(
			err, "couldn't make overlay of development pallet with pallet cache",
		)
	}
	if caches.d, err = forklift.GetDownloadCache(wpath, opts.requireDownloadCache); err != nil {
		return nil, workspaceCaches{}, err
	}
	return plt, caches, nil
}

func getShallowPallet(cwdPath string) (plt *fplt.FSPallet, err error) {
	if plt, err = fplt.LoadFSPalletContaining(cwdPath); err != nil {
		return nil, errors.Wrapf(
			err, "The current working directory %s is not part of a Forklift pallet", cwdPath,
		)
	}
	return plt, nil
}

func overlayPalletCacheOverrides(
	underlay caching.PathedPalletCache, pallets []string, plt *fplt.FSPallet,
) (palletCache *caching.LayeredPalletCache, err error) {
	palletCache = &caching.LayeredPalletCache{
		Underlay: underlay,
	}
	replacementPallets, err := loadReplacementPallets(pallets)
	if err != nil {
		return nil, err
	}
	override, err := caching.NewPalletOverrideCache(replacementPallets, nil)
	if err != nil {
		return nil, err
	}
	if err = setPalletOverrideCacheVersions(plt, override); err != nil {
		return nil, err
	}
	palletCache.Overlay = override
	return palletCache, nil
}

func loadReplacementPallets(fsPaths []string) (replacements []*fplt.FSPallet, err error) {
	for _, path := range fsPaths {
		replacementPath, err := filepath.Abs(path)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't convert '%s' into an absolute path", path)
		}
		if !ffs.DirExists(replacementPath) {
			return nil, errors.Errorf("couldn't find pallet replacement path %s", replacementPath)
		}
		externalPallets, err := fplt.LoadFSPallets(ffs.DirFS(replacementPath), "**")
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't list replacement pallets in path %s", replacementPath)
		}
		if len(externalPallets) == 0 {
			return nil, errors.Errorf("no replacement pallets found in path %s", replacementPath)
		}
		for _, pallet := range externalPallets {
			version, clean := forklift.CheckGitRepoVersion(pallet.FS.Path())
			if clean {
				pallet.Version = version
			}
		}
		replacements = append(replacements, externalPallets...)
	}
	return replacements, nil
}

func setPalletOverrideCacheVersions(
	plt *fplt.FSPallet, overrideCache *caching.PalletOverrideCache,
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

// cache-all

func cacheAllAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			enableOverrides: true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckPltShallowCompat(
			plt, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if err = fcli.CacheAllReqs(
			0, plt, caches.m, caches.p, caches.d,
			c.String("platform"), c.Bool("include-disabled"), c.Bool("parallel"),
		); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Done!")
		return nil
	}
}

// show

func showAction(c *cli.Context) error {
	plt, err := getShallowPallet(c.String("cwd"))
	if err != nil {
		return err
	}
	return fcli.FprintPalletInfo(0, os.Stdout, plt)
}

// check

func checkAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requirePalletCache: true,
			enableOverrides:    true,
			merge:              true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if _, _, err := fcli.Check(0, plt, caches.pp); err != nil {
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
			enableOverrides:    true,
			merge:              true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if _, _, err = fcli.Plan(0, plt, caches.pp, c.Bool("parallel")); err != nil {
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
		if err = fcli.CheckPltShallowCompat(
			plt, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		workspace, err := fws.LoadWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}
		stageStore, err := forklift.GetStageStore(
			workspace, c.String("stage-store"), versions.NewStageStore,
		)
		if err != nil {
			return err
		}
		if _, err = fcli.StagePallet(
			0, plt, stageStore, caches.staging(), c.String("exports"),
			versions.Staging, !c.Bool("cache-img"), c.String("platform"), c.Bool("parallel"),
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}
		fmt.Fprintln(
			os.Stderr, "Done! To apply the staged pallet, you may need to reboot or run "+
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
		if err = fcli.CheckPltShallowCompat(
			plt, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}
		workspace, err := fws.LoadWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}

		stageStore, err := forklift.GetStageStore(
			workspace, c.String("stage-store"), versions.NewStageStore,
		)
		if err != nil {
			return err
		}
		index, err := fcli.StagePallet(
			0, plt, stageStore, caches.staging(), c.String("exports"),
			versions.Staging, false, c.String("platform"), c.Bool("parallel"),
			c.Bool("ignore-tool-version"),
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
		fmt.Fprintln(os.Stderr, "Done! You may need to reboot for some changes to take effect.")
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
		workspace, err := fws.LoadWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}

		if err = fcli.CheckPltShallowCompat(
			plt, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		mirrorCache, err := workspace.GetMirrorCache()
		if err != nil {
			return err
		}
		palletCache, err := workspace.GetPalletCache()
		if err != nil {
			return err
		}
		downloaded, err := fcli.DownloadAllRequiredPallets(0, plt, mirrorCache, palletCache, nil)
		if err != nil {
			return err
		}
		if len(downloaded) == 0 {
			fmt.Fprintln(os.Stderr, "Done! No further actions are needed at this time.")
			return nil
		}

		// TODO: warn if any downloaded pallet doesn't appear to be an actual pallet, or if any pallet's
		// forklift version is incompatible or ahead of the pallet version
		fmt.Fprintln(os.Stderr, "Done!")
		return nil
	}
}

// ls-plt

func lsPltAction(c *cli.Context) error {
	plt, err := getShallowPallet(c.String("cwd"))
	if err != nil {
		return err
	}
	return fcli.FprintRequiredPallets(0, os.Stdout, plt)
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

	return fcli.FprintRequiredPalletInfo(0, os.Stdout, plt, caches.p, c.Args().First())
}

// show-plt-version

func showPltVersionAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		enableOverrides: true,
	})
	if err != nil {
		return err
	}

	return fcli.FprintRequiredPalletVersion(0, os.Stdout, plt, caches.p, c.Args().First())
}

// add-plt

func addPltAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, err := getShallowPallet(c.String("cwd"))
		if err != nil {
			return err
		}
		workspace, err := fws.LoadWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}

		if err = fcli.CheckPltShallowCompat(
			plt, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if err = fcli.AddPalletReqs(
			0, plt, workspace.GetMirrorCachePath(), c.Args().Slice(),
		); err != nil {
			return err
		}
		if c.Bool("cache-req") {
			plt, caches, err := processFullBaseArgs(c, processingOptions{
				enableOverrides: true,
			})
			if err != nil {
				return err
			}
			if _, _, err = fcli.CacheStagingReqs(
				0, plt, caches.m, caches.p, caches.d, c.String("platform"), false, c.Bool("parallel"),
			); err != nil {
				return err
			}
			// TODO: check version compatibility between the pallet and the added pallet!
		}
		fmt.Fprintln(os.Stderr, "Done!")
		return nil
	}
}

// del-plt

func delPltAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, err := getShallowPallet(c.String("cwd"))
		if err != nil {
			return err
		}

		if err = fcli.CheckPltShallowCompat(
			plt, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if err = fcli.RemovePalletReqs(0, plt, c.Args().Slice(), c.Bool("force")); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Done!")
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

	plt, err = forklift.GetRequiredPallet(plt, caches.p, c.Args().First())
	if err != nil {
		return nil
	}
	filter := c.Args().Get(1)
	if filter == "" {
		// Exclude hidden directories such as `.git`
		filter = "{*,[^.]*/**}"
	}
	paths, err := forklift.ListPalletFiles(plt, filter)
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

	plt, err = forklift.GetRequiredPallet(plt, caches.p, c.Args().First())
	if err != nil {
		return nil
	}
	location, err := forklift.GetFileLocation(plt, c.Args().Get(1))
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

	plt, err = forklift.GetRequiredPallet(plt, caches.p, c.Args().First())
	if err != nil {
		return nil
	}
	return fcli.FprintFile(os.Stdout, plt, c.Args().Get(1))
}

// ls-plt-feat

func lsPltFeatAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		enableOverrides: true,
	})
	if err != nil {
		return err
	}

	plt, err = forklift.GetRequiredPallet(plt, caches.p, c.Args().First())
	if err != nil {
		return nil
	}
	return fcli.FprintPalletFeatures(0, os.Stdout, plt)
}

// show-plt-feat

func showPltFeatAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		enableOverrides: true,
	})
	if err != nil {
		return err
	}

	plt, err = forklift.GetRequiredPallet(plt, caches.p, c.Args().First())
	if err != nil {
		return nil
	}
	return fcli.FprintFeatureInfo(0, os.Stdout, plt, caches.p, c.Args().Get(1))
}
