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

// switch

func switchAction(toolVersion, repoMinVersion, palletMinVersion string) cli.ActionFunc {
	return func(c *cli.Context) error {
		wpath := c.String("workspace")
		if !forklift.Exists(wpath) {
			fmt.Printf("Making a new workspace at %s...", wpath)
		}
		if err := forklift.EnsureExists(wpath); err != nil {
			return errors.Wrapf(err, "couldn't make new workspace at %s", wpath)
		}

		// clone pallet
		remoteRelease := c.Args().First()
		remote, release, err := git.ParseRemoteRelease(remoteRelease)
		if err != nil {
			return errors.Wrapf(err, "couldn't parse remote release %s", remoteRelease)
		}
		if err = clonePallet(remote, release, wpath, true); err != nil {
			return errors.Wrapf(err, "couldn't clone %s@%s into %s", remote, release, wpath)
		}
		fmt.Println()
		// TODO: warn if the git repo doesn't appear to be an actual pallet, or if the pallet's forklift
		// version is incompatible

		// cache repos required by pallet
		pallet, cache, err := processFullBaseArgs(c, false)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, cache, toolVersion, repoMinVersion, palletMinVersion, c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}
		fmt.Println("Downloading repos specified by the local pallet...")
		if _, err = fcli.DownloadRequiredRepos(0, pallet, cache); err != nil {
			return err
		}
		fmt.Println()

		// apply pallet
		if err = fcli.ApplyPallet(0, pallet, cache, c.Bool("parallel")); err != nil {
			return errors.Wrap(
				err, "couldn't deploy local pallet (have you run `forklift plt cache` recently?)",
			)
		}
		fmt.Println()
		fmt.Println("Done!")
		return nil
	}
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

	remoteRelease := c.Args().First()
	remote, release, err := git.ParseRemoteRelease(remoteRelease)
	if err != nil {
		return errors.Wrapf(err, "couldn't parse remote release %s", remoteRelease)
	}
	if err = clonePallet(remote, release, wpath, c.Bool("force")); err != nil {
		return errors.Wrapf(err, "couldn't clone %s@%s into %s", remote, release, wpath)
	}

	// TODO: warn if the git repo doesn't appear to be an actual pallet, or if the pallet's forklift
	// version is incompatible
	fmt.Println("Done! Next, you'll probably want to run `forklift plt cache-repo`.")
	return nil
}

func clonePallet(remote, release, wpath string, force bool) error {
	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return errors.Wrap(err, "couldn't load workspace")
	}

	local := workspace.GetCurrentPalletPath()
	fmt.Printf("Cloning pallet %s to %s...\n", remote, local)
	gitRepo, err := git.Clone(remote, local)
	if err != nil {
		if !errors.Is(err, git.ErrRepositoryAlreadyExists) {
			return errors.Wrapf(
				err, "couldn't clone pallet %s at release %s to %s", remote, release, local,
			)
		}
		if !force {
			return errors.Wrap(
				err,
				"you need to first delete your local pallet with `forklift plt rm` before "+
					"cloning another remote release to it, or re-run this command with the `--force` flag",
			)
		}

		pallet, perr := workspace.GetCurrentPallet()
		if perr != nil {
			return err
		}
		// TODO: we should instead clone each pallet into a pallet cache to avoid the need to overwrite
		// the local pallet
		fmt.Println(
			"Removing local pallet from workspace, because it already exists and we're " +
				"overwriting the local pallet with the pallet to be cloned...",
		)
		if err = pallet.Remove(); err != nil {
			return errors.Wrap(err, "couldn't remove local pallet")
		}
		fmt.Printf("Cloning pallet %s to %s...\n", remote, pallet.FS.Path())
		if gitRepo, err = git.Clone(remote, pallet.FS.Path()); err != nil {
			return errors.Wrapf(
				err, "couldn't clone pallet %s at release %s to %s", remote, release, pallet.FS.Path(),
			)
		}
	}
	fmt.Printf("Checking out version query %s...\n", release)
	if err = gitRepo.Checkout(release, "origin"); err != nil {
		return errors.Wrapf(err, "couldn't check out version query %s at %s", release, local)
	}
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

func checkAction(toolVersion, repoMinVersion, palletMinVersion string) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, cache, err := processFullBaseArgs(c, true)
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
		pallet, cache, err := processFullBaseArgs(c, true)
		if err != nil {
			return err
		}
		if err = fcli.CheckCompatibility(
			pallet, cache, toolVersion, repoMinVersion, palletMinVersion, c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if _, _, err = fcli.PlanPallet(0, pallet, cache, c.Bool("parallel")); err != nil {
			return errors.Wrap(
				err, "couldn't deploy local pallet (have you run `forklift plt cache` recently?)",
			)
		}
		return nil
	}
}

// apply

func applyAction(toolVersion, repoMinVersion, palletMinVersion string) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, cache, err := processFullBaseArgs(c, true)
		if err != nil {
			return err
		}
		if err = fcli.CheckCompatibility(
			pallet, cache, toolVersion, repoMinVersion, palletMinVersion, c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if err := fcli.ApplyPallet(0, pallet, cache, c.Bool("parallel")); err != nil {
			return errors.Wrap(
				err, "couldn't deploy local pallet (have you run `forklift plt cache` recently?)",
			)
		}
		fmt.Println()
		fmt.Println("Done!")
		return nil
	}
}
