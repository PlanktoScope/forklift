package plt

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
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
			pallet, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if err = fcli.CacheAllRequirements(
			0, pallet, repoCache.Path(), repoCache, dlCache,
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

		if err = checkPalletDirtiness(workspace, c.Bool("force")); err != nil {
			return err
		}
		if forklift.DirExists(workspace.GetCurrentPalletPath()) {
			fmt.Printf("Deleting the local pallet to replace it with %s...", query)
			fmt.Println()
		}
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

func checkPalletDirtiness(workspace *forklift.FSWorkspace, force bool) error {
	palletPath := workspace.GetCurrentPalletPath()
	if !forklift.DirExists(palletPath) {
		return nil
	}

	gitRepo, err := git.Open(palletPath)
	if err != nil {
		if !force {
			return errors.Errorf(
				"the local pallet already exists and is not a valid Git repo, but we can only delete and " +
					"replace such pallets if the --force flag is enabled",
			)
		}
		fmt.Println(
			"Warning: we will delete and replace the local pallet even though it's not a Git repo!",
		)
	}

	status, err := gitRepo.Status()
	if err != nil {
		return errors.Wrapf(err, "couldn't check status of %s as a Git repo", palletPath)
	}
	if len(status) > 0 {
		if !force {
			return errors.Errorf(
				"the local pallet already exists and has changes which have not yet been saved in a Git " +
					"commit (i.e. which have not yet been backed up), but we can only delete and replace " +
					"such pallets if the --force flag is enabled",
			)
		}
		fmt.Println(
			"Warning: we will delete and replace the local pallet even though it has changes which " +
				"have not yet been saved in a Git commit (i.e. which have not yet been backed up)!",
		)
	}

	fmt.Printf(
		"Checking whether the current commit of %s exists on a remote Git repo...\n", palletPath,
	)
	remotesHaveHead, err := isHeadInRemotes(1, gitRepo)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't check whether current commit of %s exists on a remote Git repo", palletPath,
		)
	}
	if !remotesHaveHead {
		if !force {
			return errors.Errorf(
				"the local pallet already exists and is on a commit which might not exist on a remote " +
					"Git repo (i.e. which has not yet been backed up), but we can only delete and replace " +
					"such pallets if the --force flag is enabled",
			)
		}
		fmt.Println(
			"Warning: we will delete and replace the local pallet even though it is on a commit which " +
				"might not exist on a remote Git repo (i.e. which has not yet been backed up)!",
		)
	}

	return nil
}

func isHeadInRemotes(indent int, gitRepo *git.Repo) (bool, error) {
	remotes, err := gitRepo.Remotes()
	if err != nil {
		return false, errors.Wrapf(err, "couldn't check Git remotes")
	}
	fcli.SortRemotes(remotes)

	refs := make([]*plumbing.Reference, 0)
	queryCacheMirrorRemote := false
	for _, remote := range remotes {
		if remote.Config().Name == fcli.ForkliftCacheMirrorRemoteName && !queryCacheMirrorRemote {
			fcli.IndentedPrintf(
				indent, "Skipped remote %s, because remote origin's references were successfully "+
					"retrieved!\n",
				remote.Config().Name,
			)
			continue
		}

		remoteRefs, err := remote.List(git.EmptyListOptions())
		if err != nil {
			fcli.IndentedPrintf(indent, "Warning: %s\n", errors.Wrapf(
				err, "couldn't retrieve references for remote %s", remote.Config().Name,
			))
			if remote.Config().Name == fcli.OriginRemoteName {
				queryCacheMirrorRemote = true
			}
			continue
		}
		fcli.IndentedPrintf(indent, "Retrieved references for remote %s!\n", remote.Config().Name)
		for _, ref := range remoteRefs {
			if strings.HasPrefix(string(ref.Name()), "refs/pull/") {
				continue
			}
			refs = append(refs, ref)
		}
	}

	head, err := gitRepo.GetHead()
	if err != nil {
		return false, errors.Wrapf(err, "couldn't determine the current Git commit")
	}
	const shortHashLength = 7
	fcli.IndentedPrintf(
		indent, "Searching ancestors of retrieved remote references for current commit %s...\n",
		head[:shortHashLength],
	)
	remotesHaveHead, err := gitRepo.RefsHaveAncestor(refs, head)
	if err != nil {
		fcli.IndentedPrintln(
			indent, errors.Wrapf(err, "Warning: couldn't check whether remotes have commit %s", head),
		)
	}
	if remotesHaveHead {
		fcli.IndentedPrintf(
			indent, "Found current commit %s in one of the remotes!\n", head[:shortHashLength],
		)
	}
	return remotesHaveHead, nil
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
		pallet, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
		ignoreToolVersion,
	); err != nil {
		return err
	}

	// cache everything required by pallet
	if cacheStagingReqs {
		fmt.Println()
		if err = fcli.CacheStagingRequirements(
			0, pallet, repoCache.Path(), repoCache, dlCache, false, parallel,
		); err != nil {
			return err
		}
	}
	return nil
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

		if err = checkPalletDirtiness(workspace, c.Bool("force")); err != nil {
			return err
		}
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
	fcli.IndentedPrintln(indent, "Resolving upgrade version query...")
	upgradeResolved, err := fcli.ResolveQueriesUsingLocalMirrors(
		// Note: we don't increase indentation level because go-git prints to stdout without indentation
		indent, workspace.GetPalletCachePath(), []string{upgradeQuery.String()}, true,
	)
	if err != nil {
		return errors.Wrap(err, "couldn't resolve upgrade version query")
	}

	fmt.Println()
	currentResolved, err := resolveCurrentPalletVersion(indent, workspace)
	if err != nil {
		currentResolved = forklift.GitRepoReq{}
		fcli.IndentedPrintf(indent, "Warning: %s\n", errors.Wrap(
			err,
			"we couldn't determine & resolve the current version of the local pallet, so any change "+
				"could be either an upgrade or a downgrade",
		))
	}

	fmt.Println()
	return printUpgrade(
		indent, currentResolved, upgradeResolved[upgradeQuery.String()], allowDowngrade,
	)
	// TODO: also report whether the update is cached
}

func resolveCurrentPalletVersion(
	indent int, workspace *forklift.FSWorkspace,
) (resolved forklift.GitRepoReq, err error) {
	// Inspect the current pallet
	pallet, err := workspace.GetCurrentPallet()
	if err != nil {
		return forklift.GitRepoReq{}, errors.Wrap(err, "couldn't load local pallet from workspace")
	}
	ref, err := git.Head(pallet.FS.Path())
	if err != nil {
		return forklift.GitRepoReq{}, errors.Wrap(
			err, "couldn't determine current commit of local pallet",
		)
	}
	currentQuery := forklift.GitRepoQuery{
		Path:         pallet.Def.Pallet.Path,
		VersionQuery: ref.Hash().String(),
	}
	fcli.IndentedPrintf(
		indent, "Current pallet: %s at %s\n", pallet.Def.Pallet.Path, git.StringifyRef(ref),
	)
	fmt.Println()

	fcli.IndentedPrintln(indent, "Resolving current version query...")
	currentResolved, err := fcli.ResolveQueriesUsingLocalMirrors(
		// Note: we don't increase indentation level because go-git prints to stdout without indentation
		// Note: we don't update the local mirror because we already updated it to resolve the current
		// version query
		indent, workspace.GetPalletCachePath(), []string{currentQuery.String()}, false,
	)
	if err != nil {
		fcli.IndentedPrintf(indent+1, "Warning: %s\n", errors.Wrap(
			err,
			"couldn't resolve current version query from the Forklift pallet cache's local mirror of "+
				"the remote repo (is the local pallet currently on a commit not in the remote origin?)",
		))
		fcli.IndentedPrintln(indent, "Resolving current version query using local pallet instead...")
		resolvedVersionLock, err := fcli.ResolveVersionQueryUsingRepo(
			pallet.FS.Path(), currentQuery.VersionQuery,
		)
		if err != nil {
			return forklift.GitRepoReq{}, errors.Wrap(
				err, "couldn't resolve current version query from the local pallet",
			)
		}

		fcli.IndentedPrintf(
			indent+1, "Resolved %s as %s@%s",
			currentQuery.String(), pallet.Def.Pallet.Path, resolvedVersionLock.Version,
		)
		return forklift.GitRepoReq{
			RequiredPath: pallet.Def.Pallet.Path,
			VersionLock:  resolvedVersionLock,
		}, nil
	}

	return currentResolved[currentQuery.String()], nil
}

func printUpgrade(indent int, current, upgrade forklift.GitRepoReq, allowDowngrade bool) error {
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
		fcli.IndentedPrintf(
			indent, "Upgrade/downgrade available: unknown version -> %s\n", upgrade.VersionLock.Version,
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
		fcli.IndentedPrintf(
			indent, "Warning: the upgrade query would change the local pallet from %s to %s!\n",
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
	fcli.IndentedPrintf(
		indent, "%s available: %s@%s -> %s@%s\n",
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
			pallet, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if !c.Bool("no-cache-req") {
			if err = fcli.CacheStagingRequirements(
				0, pallet, repoCache.Path(), repoCache, dlCache, false, c.Bool("parallel"),
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
		// Note: we cannot guarantee that all requirements are cached, so we don't check their versions
		// here; fcli.StagePallet will do those checks for us.
		if err = fcli.CheckShallowCompatibility(
			pallet, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
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
			versions.Versions, c.Bool("no-cache-img"), c.Bool("parallel"), c.Bool("ignore-tool-version"),
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
		// Note: we cannot guarantee that all requirements are cached, so we don't check their versions
		// here; fcli.StagePallet will do those checks for us.
		if err = fcli.CheckShallowCompatibility(
			pallet, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
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
			versions.Versions, false, c.Bool("parallel"), c.Bool("ignore-tool-version"),
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
