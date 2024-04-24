package plt

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/internal/clients/git"
)

func processFullBaseArgs(
	wpath string, ensureCache bool,
) (pallet *forklift.FSPallet, cache forklift.PathedRepoCache, err error) {
	if pallet, err = getPallet(wpath); err != nil {
		return nil, nil, err
	}
	if cache, _, err = fcli.GetRepoCache(wpath, pallet, ensureCache); err != nil {
		return nil, nil, err
	}
	return pallet, cache, nil
}

func getPallet(wpath string) (pallet *forklift.FSPallet, err error) {
	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	if pallet, err = workspace.GetCurrentPallet(); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load local pallet from workspace (you may need to first set up a local "+
				"pallet with `forklift plt clone`)",
		)
	}
	return pallet, nil
}

// cache-all

func cacheAllAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, cache, err := processFullBaseArgs(c.String("workspace"), false)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, cache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		changed, err := fcli.CacheAllRequirements(
			pallet, cache.Path(), cache, c.Bool("include-disabled"), c.Bool("parallel"),
		)
		if err != nil {
			return err
		}
		if !changed {
			fmt.Println("Done! No further actions are needed at this time.")
			return nil
		}
		fmt.Println("Done! Next, you'll probably want to run `forklift plt stage`.")
		return nil
	}
}

// switch

func switchAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		workspace, err := ensureWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}
		pallet, repoCache, err := preparePallet(
			workspace, c.Args().First(), true, true, c.Bool("ignore-tool-version"), versions,
		)
		if err != nil {
			return err
		}
		fmt.Println()

		stageStore, err := fcli.GetStageStore(
			workspace, c.String("stage-store"), versions.NewStageStore,
		)
		if err != nil {
			return err
		}
		index, err := fcli.StagePallet(
			pallet, stageStore, repoCache, c.String("exports"),
			versions.Tool, versions.MinSupportedBundle, versions.NewBundle,
			c.Bool("no-cache-img") || c.Bool("apply"), c.Bool("parallel"), c.Bool("ignore-tool-version"),
		)
		if err != nil {
			return err
		}
		if !c.Bool("apply") {
			fmt.Println(
				"Done! To apply the staged pallet immediately, run `sudo -E forklift stage apply`.",
			)
			return nil
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

func ensureWorkspace(wpath string) (*forklift.FSWorkspace, error) {
	if !forklift.Exists(wpath) {
		fmt.Printf("Making a new workspace at %s...", wpath)
	}
	if err := forklift.EnsureExists(wpath); err != nil {
		return nil, errors.Wrapf(err, "couldn't make new workspace at %s", wpath)
	}
	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	if err = forklift.EnsureExists(workspace.GetDataPath()); err != nil {
		return nil, errors.Wrapf(err, "couldn't ensure the existence of %s", workspace.GetDataPath())
	}
	return workspace, nil
}

func preparePallet(
	workspace *forklift.FSWorkspace, gitRepoQuery string,
	removeExistingLocalPallet, cacheStagingReqs, ignoreToolVersion bool, versions Versions,
) (pallet *forklift.FSPallet, repoCache forklift.PathedRepoCache, err error) {
	// clone pallet
	if removeExistingLocalPallet {
		fmt.Println("Warning: if a local pallet already exists, it will be deleted now...")
		if err = os.RemoveAll(workspace.GetCurrentPalletPath()); err != nil {
			return nil, nil, errors.Wrap(err, "couldn't remove local pallet")
		}
	}
	if err = fcli.CloneQueriedGitRepoUsingLocalMirror(
		0, workspace.GetPalletCachePath(), gitRepoQuery, workspace.GetCurrentPalletPath(),
	); err != nil {
		return nil, nil, err
	}
	fmt.Println()
	// TODO: warn if the git repo doesn't appear to be an actual pallet

	if pallet, repoCache, err = processFullBaseArgs(workspace.FS.Path(), false); err != nil {
		return nil, nil, err
	}
	if err = fcli.CheckShallowCompatibility(
		pallet, repoCache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
		ignoreToolVersion,
	); err != nil {
		return pallet, repoCache, err
	}

	// cache everything required by pallet
	if cacheStagingReqs {
		if _, err = fcli.CacheStagingRequirements(pallet, repoCache.Path()); err != nil {
			return pallet, repoCache, err
		}
	}
	return pallet, repoCache, nil
}

// clone

func cloneAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		workspace, err := ensureWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}

		if _, _, err = preparePallet(
			workspace, c.Args().First(),
			c.Bool("force"), !c.Bool("no-cache-req"), c.Bool("ignore-tool-version"), versions,
		); err != nil {
			return err
		}

		fmt.Println("Done!")
		return nil
	}
}

// fetch

func fetchAction(c *cli.Context) error {
	workspace, err := forklift.LoadWorkspace(c.String("workspace"))
	if err != nil {
		return err
	}
	palletPath := workspace.GetCurrentPalletPath()

	fmt.Println("Fetching updates...")
	updated, err := git.Fetch(palletPath)
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
	workspace, err := forklift.LoadWorkspace(c.String("workspace"))
	if err != nil {
		return err
	}
	palletPath := workspace.GetCurrentPalletPath()

	fmt.Println("Attempting to fast-forward the local pallet...")
	updated, err := git.Pull(palletPath)
	if err != nil {
		return errors.Wrap(err, "couldn't fast-forward the local pallet")
	}
	if !updated {
		fmt.Println("No changes from the remote release.")
	}

	// TODO: warn if the git repo doesn't appear to be an actual pallet, or if the pallet's forklift
	// version is incompatible
	// TODO: display changes
	return nil
}

// rm

func rmAction(c *cli.Context) error {
	workspace, err := forklift.LoadWorkspace(c.String("workspace"))
	if err != nil {
		return err
	}
	palletPath := workspace.GetCurrentPalletPath()

	fmt.Printf("Removing local pallet from workspace...\n")
	// TODO: return an error if there are uncommitted or unpushed changes to be removed - in which
	// case require a --force flag
	return errors.Wrap(os.RemoveAll(palletPath), "couldn't remove local pallet")
}

// show

func showAction(c *cli.Context) error {
	pallet, err := getPallet(c.String("workspace"))
	if err != nil {
		return err
	}
	return fcli.PrintPalletInfo(0, pallet)
}

// check

func checkAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, cache, err := processFullBaseArgs(c.String("workspace"), true)
		if err != nil {
			return err
		}
		if err = fcli.CheckCompatibility(
			pallet, cache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if _, _, err := fcli.Check(0, pallet, cache); err != nil {
			return err
		}
		return nil
	}
}

// plan

func planAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, cache, err := processFullBaseArgs(c.String("workspace"), true)
		if err != nil {
			return err
		}
		if err = fcli.CheckCompatibility(
			pallet, cache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if _, _, err = fcli.Plan(0, pallet, cache, c.Bool("parallel")); err != nil {
			return errors.Wrap(
				err, "couldn't deploy local pallet (have you run `forklift plt cache` recently?)",
			)
		}
		return nil
	}
}

// stage

func stageAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, cache, err := processFullBaseArgs(c.String("workspace"), true)
		if err != nil {
			return err
		}
		if err = fcli.CheckCompatibility(
			pallet, cache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
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
			pallet, stageStore, cache, c.String("exports"),
			versions.Tool, versions.MinSupportedBundle, versions.NewBundle,
			c.Bool("no-cache-img"), c.Bool("parallel"), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}
		fmt.Println("Done! To apply the staged pallet immediately, run `sudo -E forklift stage apply`.")
		return nil
	}
}

// apply

func applyAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, repoCache, err := processFullBaseArgs(c.String("workspace"), true)
		if err != nil {
			return err
		}
		if err = fcli.CheckCompatibility(
			pallet, repoCache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
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
			pallet, stageStore, repoCache, c.String("exports"),
			versions.Tool, versions.MinSupportedBundle, versions.NewBundle,
			false, c.Bool("parallel"), c.Bool("ignore-tool-version"),
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
