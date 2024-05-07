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

func processFullBaseArgs(wpath string, ensureCache bool) (
	pallet *forklift.FSPallet, repoCache forklift.PathedRepoCache, dlCache *forklift.FSDownloadCache,
	err error,
) {
	if pallet, err = getPallet(wpath); err != nil {
		return nil, nil, nil, err
	}
	if dlCache, err = fcli.GetDlCache(wpath, ensureCache); err != nil {
		return nil, nil, nil, err
	}
	if repoCache, _, err = fcli.GetRepoCache(wpath, pallet, ensureCache); err != nil {
		return nil, nil, nil, err
	}
	return pallet, repoCache, dlCache, nil
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
		pallet, repoCache, dlCache, err := processFullBaseArgs(c.String("workspace"), false)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, repoCache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if err = fcli.CacheAllRequirements(
			pallet, repoCache.Path(), repoCache, dlCache,
			c.Bool("include-disabled"), c.Bool("parallel"),
		); err != nil {
			return err
		}
		fmt.Println("Done!")
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
		pallet, repoCache, dlCache, err := preparePallet(
			workspace, c.Args().First(), true, true, c.Bool("parallel"),
			c.Bool("ignore-tool-version"), versions,
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
			pallet, stageStore, repoCache, dlCache, c.String("exports"),
			versions.Tool, versions.MinSupportedBundle, versions.NewBundle,
			c.Bool("no-cache-img") || c.Bool("apply"), c.Bool("parallel"), c.Bool("ignore-tool-version"),
		)
		if err != nil {
			return err
		}
		if !c.Bool("apply") {
			fmt.Println(
				"Done! To apply the staged pallet, you may need to reboot or run " +
					"`forklift stage apply` (or `sudo -E forklift stage apply` if you need sudo for Docker).",
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
	removeExistingLocalPallet, cacheStagingReqs, parallel,
	ignoreToolVersion bool, versions Versions,
) (
	pallet *forklift.FSPallet, repoCache forklift.PathedRepoCache, dlCache *forklift.FSDownloadCache,
	err error,
) {
	// clone pallet
	if removeExistingLocalPallet {
		fmt.Println("Warning: if a local pallet already exists, it will be deleted now...")
		if err = os.RemoveAll(workspace.GetCurrentPalletPath()); err != nil {
			return nil, nil, nil, errors.Wrap(err, "couldn't remove local pallet")
		}
	}
	if err = fcli.CloneQueriedGitRepoUsingLocalMirror(
		0, workspace.GetPalletCachePath(), gitRepoQuery, workspace.GetCurrentPalletPath(),
	); err != nil {
		return nil, nil, nil, err
	}
	fmt.Println()
	// TODO: warn if the git repo doesn't appear to be an actual pallet

	if pallet, repoCache, dlCache, err = processFullBaseArgs(workspace.FS.Path(), false); err != nil {
		return nil, nil, nil, err
	}
	if err = fcli.CheckShallowCompatibility(
		pallet, repoCache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
		ignoreToolVersion,
	); err != nil {
		return pallet, repoCache, dlCache, err
	}

	// cache everything required by pallet
	if cacheStagingReqs {
		if err = fcli.CacheStagingRequirements(
			pallet, repoCache.Path(), repoCache, dlCache, false, parallel,
		); err != nil {
			return pallet, repoCache, dlCache, err
		}
	}
	return pallet, repoCache, dlCache, nil
}

// clone

func cloneAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		workspace, err := ensureWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}

		if _, _, _, err = preparePallet(
			workspace, c.Args().First(),
			c.Bool("force"), !c.Bool("no-cache-req"), c.Bool("parallel"),
			c.Bool("ignore-tool-version"), versions,
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

func pullAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
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
		// TODO: display changes

		fmt.Println()

		pallet, repoCache, dlCache, err := processFullBaseArgs(c.String("workspace"), false)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, repoCache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if !c.Bool("no-cache-req") {
			if err = fcli.CacheStagingRequirements(
				pallet, repoCache.Path(), repoCache, dlCache, false, c.Bool("parallel"),
			); err != nil {
				return err
			}
		}
		fmt.Println("Done!")
		return nil
	}
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
		pallet, repoCache, _, err := processFullBaseArgs(c.String("workspace"), true)
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
		pallet, repoCache, _, err := processFullBaseArgs(c.String("workspace"), true)
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
		pallet, repoCache, dlCache, err := processFullBaseArgs(c.String("workspace"), true)
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
		if _, err = fcli.StagePallet(
			pallet, stageStore, repoCache, dlCache, c.String("exports"),
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
		pallet, repoCache, dlCache, err := processFullBaseArgs(c.String("workspace"), true)
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
			pallet, stageStore, repoCache, dlCache, c.String("exports"),
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
