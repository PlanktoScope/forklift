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

		// TODO: detect if there are any un-committed and un-pushed changes in the pallet as a git repo,
		// and require special confirmation if so.
		fmt.Printf(
			"Warning: if a local pallet already exists, it will be deleted now to be replaced with "+
				"%s...\n",
			query,
		)
		fmt.Println()
		if err := os.RemoveAll(workspace.GetCurrentPalletPath()); err != nil {
			return errors.Wrap(err, "couldn't remove local pallet")
		}

		if err = preparePallet(
			// Note: we don't cache staging requirements because that will be handled by the apply/stage
			// step anyways:
			workspace, query, true, false, c.Bool("parallel"), c.Bool("ignore-tool-version"), versions,
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
	updateLocalMirror, cacheStagingReqs, parallel, ignoreToolVersion bool, versions Versions,
) error {
	// clone pallet
	if err := fcli.CloneQueriedGitRepoUsingLocalMirror(
		0, workspace.GetPalletCachePath(), gitRepoQuery.Path, gitRepoQuery.VersionQuery,
		workspace.GetCurrentPalletPath(), updateLocalMirror,
	); err != nil {
		return err
	}
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
		fmt.Println()
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
	if providedQuery == "" {
		providedQuery = "@"
	}
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
	query = loaded.Overlay(provided)

	if !query.Complete() {
		return query, loaded, provided, errors.Errorf(
			"provided query %s could not be fully completed with stored query %s", provided, loaded,
		)
	}

	return query, loaded, provided, nil
}

// upgrade

func upgradeAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		workspace, err := ensureWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}

		query, err := handlePalletQuery(workspace, c.Args().First())
		if err != nil {
			return errors.Wrapf(err, "couldn't handle provided version query %s", c.Args().First())
		}
		if err = checkUpgrade(0, workspace, query, c.Bool("allow-downgrade")); err != nil {
			return err
		}

		// TODO: detect if there are any un-committed and un-pushed changes in the pallet as a git repo,
		// and require special confirmation if so.
		fmt.Printf("Deleting the local pallet to replace it with %s...", query)
		fmt.Println()
		if err := os.RemoveAll(workspace.GetCurrentPalletPath()); err != nil {
			return errors.Wrap(err, "couldn't remove local pallet")
		}

		if err = preparePallet(
			// Note: we don't cache staging requirements because that will be handled by the apply/stage
			// step anyways:
			workspace, query, false, false, c.Bool("parallel"), c.Bool("ignore-tool-version"), versions,
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

func checkUpgrade(
	indent int, workspace *forklift.FSWorkspace, upgradeQuery forklift.GitRepoQuery,
	allowDowngrade bool,
) error {
	queries := []string{upgradeQuery.String()}

	// Inspect the current pallet
	pallet, err := workspace.GetCurrentPallet()
	if err != nil {
		return errors.Wrap(err, "couldn't load local pallet from workspace")
	}
	currentQuery := forklift.GitRepoQuery{}
	ref, err := git.Head(pallet.FS.Path())
	if err != nil {
		// Note: the !allowDowngrade case is handled by printUpgrade. Here we print the error for the
		// underlying reason we can't determine the current version:
		fcli.IndentedPrintf(indent, "Warning: %s\n", errors.Wrap(
			err,
			"we couldn't determine the current version of the local pallet, so any change could be "+
				"either an upgrade or a downgrade",
		))
	} else {
		currentQuery = forklift.GitRepoQuery{
			Path:         pallet.Def.Pallet.Path,
			VersionQuery: ref.Hash().String(),
		}
		fmt.Printf("Current pallet: %s at %s\n", pallet.Def.Pallet.Path, git.StringifyRef(ref))
		queries = append(queries, currentQuery.String())
	}
	fmt.Println()

	fmt.Println("Resolving version queries...")
	resolved, err := fcli.ResolveQueriesUsingLocalMirrors(
		// Note: we don't increase indentation level because go-git prints to stdout without indentation
		indent, workspace.GetPalletCachePath(), queries, true,
	)
	if err != nil {
		return errors.Wrap(err, "couldn't resolve version queries")
	}

	fmt.Println()
	return printUpgrade(
		resolved[currentQuery.String()], resolved[upgradeQuery.String()], allowDowngrade,
	)
	// TODO: also report whether the update is cached
}

func printUpgrade(current, upgrade forklift.GitRepoReq, allowDowngrade bool) error {
	if current == upgrade {
		return errors.New("no upgrade found")
	}
	if current == (forklift.GitRepoReq{}) {
		if !allowDowngrade {
			return errors.Errorf(
				"upgrade/downgrade available to %s, but we couldn't determine whether the change is a "+
					"downgrade because we couldn't determine the current version, and we aren't considering "+
					"downgrades because the --allow-downgrade flag isn't set",
				upgrade.VersionLock.Version,
			)
		}
		fmt.Printf(
			"Upgrade/downgrade available: unknown version -> %s\n", upgrade.VersionLock.Version,
		)
		return nil
	}
	operation := "Upgrade"
	if current.RequiredPath != upgrade.RequiredPath {
		operation = "Upgrade/downgrade"
		if !allowDowngrade {
			// Note: the !allowDowngrade case is handled by printUpgrade
			return errors.Errorf(
				"the upgrade query would change the local pallet from %s to %s, but we can't determine "+
					"whether that might result in a downgrade, and we aren't considering downgrades because "+
					"the --allow-downgrade flag isn't set",
				current.RequiredPath, upgrade.RequiredPath,
			)
		}
		fmt.Printf(
			"Warning: the upgrade query would change the local pallet from %s to %s!\n",
			current.RequiredPath, upgrade.RequiredPath,
		)
	} else if current.VersionLock.Version > upgrade.VersionLock.Version {
		operation = "Downgrade"
		if !allowDowngrade {
			return errors.Errorf(
				"downgrade available from %s to %s, but we aren't considering downgrades because the "+
					"--allow-downgrade flag isn't set",
				current.VersionLock.Version, upgrade.VersionLock.Version,
			)
		}
	}
	fmt.Printf(
		"%s available: %s@%s -> %s@%s\n",
		operation,
		current.RequiredPath, current.VersionLock.Version,
		upgrade.RequiredPath, upgrade.VersionLock.Version,
	)
	return nil
}

// check-upgrade

func checkUpgradeAction(c *cli.Context) error {
	workspace, err := ensureWorkspace(c.String("workspace"))
	if err != nil {
		return err
	}

	providedQuery := c.Args().First()
	query, loaded, provided, err := completePalletQuery(workspace, providedQuery)
	if err != nil {
		return errors.Wrapf(err, "couldn't complete provided version query %s", providedQuery)
	}
	if providedQuery == "" {
		fmt.Printf("Loaded upgrade query: %s\n", query)
	} else if !provided.Complete() {
		fmt.Printf(
			"Provided query %s was completed with stored query %s as %s!\n", provided, loaded, query,
		)
	}

	if err := checkUpgrade(0, workspace, query, c.Bool("allow-downgrade")); err != nil {
		return err
	}
	return nil
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

		if c.Bool("force") {
			fmt.Printf(
				"Warning: if a local pallet already exists, it will be deleted now to be replaced with "+
					"%s...\n",
				query,
			)
			fmt.Println()
			if err := os.RemoveAll(workspace.GetCurrentPalletPath()); err != nil {
				return errors.Wrap(err, "couldn't remove local pallet")
			}
		}

		if err = preparePallet(
			workspace, query, true, !c.Bool("no-cache-req"), c.Bool("parallel"),
			c.Bool("ignore-tool-version"), versions,
		); err != nil {
			return err
		}
		fmt.Println()

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

		// FIXME: update the local mirror

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
