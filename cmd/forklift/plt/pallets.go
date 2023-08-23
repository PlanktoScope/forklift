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
) (pallet *forklift.FSPallet, cache *forklift.FSRepoCache, err error) {
	workspace, err := forklift.LoadWorkspace(c.String("workspace"))
	if err != nil {
		return nil, nil, err
	}
	if pallet, err = getPallet(c.String("workspace")); err != nil {
		return nil, nil, err
	}
	if cache, err = workspace.GetRepoCache(); err != nil {
		return nil, nil, err
	}
	if ensureCache && !cache.Exists() {
		return nil, nil, errors.New(
			"you first need to cache the repos specified by your pallet with `forklift plt cache-repo`",
		)
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
		if !c.Bool("force") {
			return errors.Wrap(
				err,
				"you need to first delete your local pallet with `forklift plt rm` before "+
					"cloning another remote release to it",
			)
		}

		pallet, perr := workspace.GetCurrentPallet()
		if perr != nil {
			return err
		}
		fmt.Println(
			"Removing local pallet from workspace, because it already exists and the " +
				"command's --force flag was enabled...",
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
	fmt.Printf("Checking out release %s...\n", release)
	if err = gitRepo.Checkout(release); err != nil {
		return errors.Wrapf(err, "couldn't check out release %s at %s", release, local)
	}
	fmt.Println("Done! Next, you'll probably want to run `forklift plt cache-repo`.")
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

func checkAction(c *cli.Context) error {
	pallet, cache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}

	if err := fcli.CheckPallet(0, pallet, cache); err != nil {
		return err
	}
	return nil
}

// plan

func planAction(c *cli.Context) error {
	pallet, cache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}

	if err := fcli.PlanPallet(0, pallet, cache); err != nil {
		return errors.Wrap(
			err, "couldn't deploy local pallet (have you run `forklift plt cache` recently?)",
		)
	}
	return nil
}

// apply

func applyAction(c *cli.Context) error {
	pallet, cache, err := processFullBaseArgs(c, true)
	if err != nil {
		return err
	}

	if err := fcli.ApplyPallet(0, pallet, cache); err != nil {
		return errors.Wrap(
			err, "couldn't deploy local pallet (have you run `forklift plt cache` recently?)",
		)
	}
	fmt.Println()
	fmt.Println("Done!")
	return nil
}
