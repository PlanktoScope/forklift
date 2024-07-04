package plt

import (
	"fmt"
	"os"
	"strings"

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

		query, err := handlePalletQuery(workspace, c.Args().First())
		if err != nil {
			return errors.Wrapf(err, "couldn't handle provided version query %s", c.Args().First())
		}

		if err = preparePallet(
			workspace, query, true, true, c.Bool("parallel"),
			c.Bool("ignore-tool-version"), versions,
		); err != nil {
			return err
		}
		fmt.Println()

		if c.Bool("apply") {
			return applyAction(versions)(c)
		}
		return stageAction(versions)(c)
	}
}

func ensureWorkspace(wpath string) (*forklift.FSWorkspace, error) {
	if !forklift.DirExists(wpath) {
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
	workspace *forklift.FSWorkspace, gitRepoQuery forklift.GitRepoQuery,
	removeExistingLocalPallet, cacheStagingReqs, parallel,
	ignoreToolVersion bool, versions Versions,
) error {
	// clone pallet
	if removeExistingLocalPallet {
		fmt.Println("Warning: if a local pallet already exists, it will be deleted now...")
		if err := os.RemoveAll(workspace.GetCurrentPalletPath()); err != nil {
			return errors.Wrap(err, "couldn't remove local pallet")
		}
	}

	if err := fcli.CloneQueriedGitRepoUsingLocalMirror(
		0, workspace.GetPalletCachePath(), gitRepoQuery.Path, gitRepoQuery.VersionQuery,
		workspace.GetCurrentPalletPath(),
	); err != nil {
		return err
	}
	fmt.Println()
	// TODO: warn if the git repo doesn't appear to be an actual pallet

	pallet, repoCache, dlCache, err := processFullBaseArgs(workspace.FS.Path(), false)
	if err != nil {
		return err
	}
	if err = fcli.CheckShallowCompatibility(
		pallet, repoCache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
		ignoreToolVersion,
	); err != nil {
		return err
	}

	// cache everything required by pallet
	if cacheStagingReqs {
		if err = fcli.CacheStagingRequirements(
			pallet, repoCache.Path(), repoCache, dlCache, false, parallel,
		); err != nil {
			return err
		}
	}
	return nil
}

func handlePalletQuery(
	workspace *forklift.FSWorkspace, providedQuery string,
) (forklift.GitRepoQuery, error) {
	query, loaded, provided, err := completePalletQuery(workspace, providedQuery)
	if err != nil {
		return forklift.GitRepoQuery{}, errors.Wrapf(
			err, "couldn't complete provided version query %s", providedQuery,
		)
	}
	if !query.Complete() {
		return query, errors.Errorf(
			"provided query %s could not be fully completed with stored query %s", provided, loaded,
		)
	}

	printed := false
	if !provided.Complete() {
		fmt.Printf(
			"Provided query %s was completed with stored query %s as %s!\n", provided, loaded, query,
		)
		printed = true
	}
	if query == loaded {
		if printed {
			fmt.Println()
		}
		return query, nil
	}

	if loaded == (forklift.GitRepoQuery{}) {
		fmt.Printf(
			"Initializing the tracked path & version query for the current pallet as %s...\n", query,
		)
	} else {
		fmt.Printf(
			"Updating the tracked path & version query for the current pallet from %s to %s...\n",
			loaded, query,
		)
	}
	if err := workspace.CommitCurrentPalletUpgrades(query); err != nil {
		return query, errors.Wrapf(err, "couldn't commit pallet query %s", query)
	}
	fmt.Println()
	return query, nil
}

func completePalletQuery(
	workspace *forklift.FSWorkspace, providedQuery string,
) (query, loaded, provided forklift.GitRepoQuery, err error) {
	palletPath, versionQuery, ok := strings.Cut(providedQuery, "@")
	if !ok {
		return forklift.GitRepoQuery{}, forklift.GitRepoQuery{}, forklift.GitRepoQuery{}, errors.Errorf(
			"couldn't parse '%s' as [pallet_path]@[version_query]", providedQuery,
		)
	}
	provided = forklift.GitRepoQuery{
		Path:         palletPath,
		VersionQuery: versionQuery,
	}
	if loaded, err = workspace.GetCurrentPalletUpgrades(); err != nil {
		if !provided.Complete() {
			return forklift.GitRepoQuery{}, forklift.GitRepoQuery{}, provided, errors.Wrap(
				err, "couldn't load stored query for the current pallet",
			)
		}
		loaded = forklift.GitRepoQuery{}
	}
	return loaded.Overlay(provided), loaded, provided, nil
}

// upgrade

func upgradeAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		workspace, err := ensureWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}

		query, err := workspace.GetCurrentPalletUpgrades()
		if err != nil {
			return errors.Wrap(err, "couldn't load stored query for upgrading the current pallet")
		}
		if !query.Complete() {
			return errors.Errorf("stored query for the current pallet is incomplete: %s", query)
		}

		// TODO: show what we're upgrading from, and what we're upgrading to

		if err = preparePallet(
			workspace, query, true, true, c.Bool("parallel"),
			c.Bool("ignore-tool-version"), versions,
		); err != nil {
			return err
		}
		fmt.Println()

		if c.Bool("apply") {
			return applyAction(versions)(c)
		}
		return stageAction(versions)(c)
	}
}

// show-upgrade-query

func showUpgradeQueryAction(c *cli.Context) error {
	workspace, err := ensureWorkspace(c.String("workspace"))
	if err != nil {
		return err
	}

	query, err := workspace.GetCurrentPalletUpgrades()
	if err != nil {
		return errors.Wrap(err, "couldn't load stored query for upgrading the current pallet")
	}
	fmt.Printf("%s\n", query)
	return nil
}

// set-upgrade-query

func setUpgradeQueryAction(c *cli.Context) error {
	workspace, err := ensureWorkspace(c.String("workspace"))
	if err != nil {
		return err
	}

	_, err = handlePalletQuery(workspace, c.Args().First())
	if err != nil {
		return errors.Wrapf(err, "couldn't handle provided version query %s", c.Args().First())
	}
	fmt.Println("Done!")
	return nil
}

// clone

func cloneAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		workspace, err := ensureWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}

		query, err := handlePalletQuery(workspace, c.Args().First())
		if err != nil {
			return errors.Wrapf(err, "couldn't handle provided version query %s", c.Args().First())
		}

		if err = preparePallet(
			workspace, query,
			c.Bool("force"), !c.Bool("no-cache-req"), c.Bool("parallel"),
			c.Bool("ignore-tool-version"), versions,
		); err != nil {
			return err
		}

		switch {
		case c.Bool("apply"):
			return applyAction(versions)(c)
		case c.Bool("stage"):
			return stageAction(versions)(c)
		default:
			fmt.Println("Done!")
			return nil
		}
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

		switch {
		case c.Bool("apply"):
			return applyAction(versions)(c)
		case c.Bool("stage"):
			return stageAction(versions)(c)
		default:
			fmt.Println("Done!")
			return nil
		}
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
