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
	c *cli.Context, ensureCache bool,
) (pallet *forklift.FSPallet, cache forklift.PathedRepoCache, err error) {
	if pallet, err = getPallet(c.String("workspace")); err != nil {
		return nil, nil, err
	}
	if cache, _, err = fcli.GetRepoCache(c.String("workspace"), pallet, ensureCache); err != nil {
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
		pallet, cache, err := processFullBaseArgs(c, false)
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

		// clone pallet
		if err = os.RemoveAll(workspace.GetCurrentPalletPath()); err != nil {
			return errors.Wrap(err, "couldn't remove local pallet")
		}
		if err = fcli.CloneQueriedGitRepoUsingLocalMirror(
			0, workspace.GetPalletCachePath(), c.Args().First(), workspace.GetCurrentPalletPath(),
		); err != nil {
			return err
		}
		fmt.Println()
		// TODO: warn if the git repo doesn't appear to be an actual pallet, or if the pallet's forklift
		// version is incompatible

		// cache everything required by pallet
		pallet, repoCache, err := processFullBaseArgs(c, false)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, repoCache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}
		if _, err = fcli.CacheAllRequirements(
			pallet, repoCache.Path(), repoCache, c.Bool("include-disabled"), c.Bool("parallel"),
		); err != nil {
			return err
		}

		if !c.Bool("apply") {
			// stage pallet
			if err = stagePallet(
				workspace, pallet, repoCache, versions, c.Bool("parallel"), c.Bool("ignore-tool-version"),
			); err != nil {
				return err
			}
			fmt.Println("Done! To apply the staged pallet, run `forklift stage apply`.")
			return nil
		}
		// apply pallet
		if err = fcli.ApplyPallet(
			pallet, repoCache, workspace, versions.NewStageStore, versions.NewBundle, c.Bool("parallel"),
		); err != nil {
			return err
		}
		fmt.Println("Done!")
		return nil
	}
}

func ensureWorkspace(wpath string) (*forklift.FSWorkspace, error) {
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

func stagePallet(
	workspace *forklift.FSWorkspace, pallet *forklift.FSPallet, repoCache forklift.PathedRepoCache,
	versions Versions, parallel, ignoreToolVersion bool,
) error {
	stageStore, err := workspace.GetStageStore(versions.NewStageStore)
	if err != nil {
		return err
	}
	if _, err = fcli.StagePallet(pallet, stageStore, repoCache, versions.NewBundle); err != nil {
		return errors.Wrap(err, "couldn't stage pallet to be applied immediately")
	}
	if err = fcli.DownloadImagesForStoreApply(
		stageStore, versions.Tool, versions.MinSupportedBundle, parallel, ignoreToolVersion,
	); err != nil {
		return errors.Wrap(err, "couldn't cache Docker container images required by staged pallet")
	}
	return nil
}

// clone

func cloneAction(c *cli.Context) error {
	wpath := c.String("workspace")
	if !forklift.Exists(wpath) {
		fmt.Printf("Making a new workspace at %s...", wpath)
	}
	if err := forklift.EnsureExists(wpath); err != nil {
		return errors.Wrapf(err, "couldn't make new workspace at %s", wpath)
	}
	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return err
	}
	if err = forklift.EnsureExists(workspace.GetDataPath()); err != nil {
		return errors.Wrapf(err, "couldn't ensure the existence of %s", workspace.GetDataPath())
	}

	if c.Bool("force") {
		if err = os.RemoveAll(workspace.GetCurrentPalletPath()); err != nil {
			return errors.Wrap(err, "couldn't remove local pallet")
		}
	}
	if err := fcli.CloneQueriedGitRepoUsingLocalMirror(
		0, workspace.GetPalletCachePath(), c.Args().First(), workspace.GetCurrentPalletPath(),
	); err != nil {
		return err
	}

	// TODO: warn if the git repo doesn't appear to be an actual pallet, or if the pallet's forklift
	// version is incompatible
	fmt.Println("Done! Next, you'll probably want to run `forklift plt cache-all`.")
	return nil
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
		pallet, cache, err := processFullBaseArgs(c, true)
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
		pallet, cache, err := processFullBaseArgs(c, true)
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
		pallet, cache, err := processFullBaseArgs(c, true)
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
		stageStore, err := workspace.GetStageStore(versions.NewStageStore)
		if err != nil {
			return err
		}
		if _, err = fcli.StagePallet(pallet, stageStore, cache, versions.NewBundle); err != nil {
			return err
		}
		fmt.Println("Done! To apply the staged pallet, you can run `sudo -E forklift stage apply`.")
		return nil
	}
}

// apply

func applyAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, repoCache, err := processFullBaseArgs(c, true)
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

		if err = fcli.ApplyPallet(
			pallet, repoCache, workspace, versions.NewStageStore, versions.NewBundle, c.Bool("parallel"),
		); err != nil {
			return err
		}
		fmt.Println("Done!")
		return nil
	}
}
