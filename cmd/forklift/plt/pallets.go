package plt

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/forklift-run/forklift/exp/caching"
	ffs "github.com/forklift-run/forklift/exp/fs"
	fplt "github.com/forklift-run/forklift/exp/pallets"
	fws "github.com/forklift-run/forklift/exp/workspaces"
	"github.com/forklift-run/forklift/internal/app/forklift"
	fcli "github.com/forklift-run/forklift/internal/app/forklift/cli"
	"github.com/forklift-run/forklift/internal/clients/git"
)

type workspaceCaches struct {
	m  *caching.FSMirrorCache
	p  *caching.FSPalletCache
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
	merge                bool
}

func processFullBaseArgs(
	wpath string, opts processingOptions,
) (plt *fplt.FSPallet, caches workspaceCaches, err error) {
	if plt, err = getShallowPallet(wpath); err != nil {
		return nil, workspaceCaches{}, err
	}
	workspace, err := fws.LoadWorkspace(wpath)
	if err != nil {
		return nil, workspaceCaches{}, err
	}
	if caches.m, err = workspace.GetMirrorCache(); err != nil {
		return nil, workspaceCaches{}, err
	}
	if caches.p, err = forklift.GetPalletCache(
		wpath, plt, opts.requirePalletCache || opts.merge,
	); err != nil {
		return nil, workspaceCaches{}, err
	}
	if opts.merge {
		if plt, err = fplt.MergeFSPallet(plt, caches.p, nil); err != nil {
			return nil, workspaceCaches{}, errors.Wrap(
				err, "couldn't merge local pallet with file imports from any pallets required by it",
			)
		}
	}
	if caches.pp, err = forklift.MakeOverlayCache(plt, caches.p); err != nil {
		return nil, workspaceCaches{}, errors.Wrap(
			err, "couldn't make overlay of local pallet with pallet cache",
		)
	}
	if caches.d, err = forklift.GetDownloadCache(wpath, opts.requireDownloadCache); err != nil {
		return nil, workspaceCaches{}, err
	}
	return plt, caches, nil
}

func getShallowPallet(wpath string) (plt *fplt.FSPallet, err error) {
	workspace, err := fws.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	if plt, err = workspace.GetCurrentPallet(); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load local pallet from workspace (you may need to first set up a local "+
				"pallet with `forklift plt clone`)",
		)
	}
	return plt, nil
}

// cache-all

func cacheAllAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{})
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

// switch

func switchAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		workspace, err := ensureWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}

		query, err := handlePalletQuery(workspace, c.Args().First(), c.Bool("set-upgrade-query"))
		if err != nil {
			return errors.Wrapf(err, "couldn't handle provided version query %s", c.Args().First())
		}

		if err = checkPalletDirtiness(workspace, c.Bool("force")); err != nil {
			return err
		}
		if ffs.DirExists(workspace.GetCurrentPalletPath()) {
			fmt.Fprintf(os.Stderr, "Deleting the local pallet to replace it with %s...", query)
			fmt.Fprintln(os.Stderr)
		}
		if err := os.RemoveAll(workspace.GetCurrentPalletPath()); err != nil {
			return errors.Wrap(err, "couldn't remove local pallet")
		}

		if err = preparePallet(
			// Note: we don't cache staging requirements because that will be handled by the apply/stage
			// step anyways:
			workspace, query, true, false, c.String("platform"), c.Bool("parallel"),
			c.Bool("ignore-tool-version"), versions,
		); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr)

		if c.Bool("apply") {
			return applyAction(versions)(c)
		}
		return stageAction(versions)(c)
	}
}

func ensureWorkspace(wpath string) (*fws.FSWorkspace, error) {
	if !ffs.DirExists(wpath) {
		fmt.Fprintf(os.Stderr, "Making a new workspace at %s...", wpath)
	}
	if err := ffs.EnsureExists(wpath); err != nil {
		return nil, errors.Wrapf(err, "couldn't make new workspace at %s", wpath)
	}
	workspace, err := fws.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	if err = ffs.EnsureExists(workspace.GetDataPath()); err != nil {
		return nil, errors.Wrapf(err, "couldn't ensure the existence of %s", workspace.GetDataPath())
	}
	return workspace, nil
}

func handlePalletQuery(
	workspace *fws.FSWorkspace, providedQuery string, commitPalletQuery bool,
) (fws.GitRepoQuery, error) {
	query, loaded, provided, err := completePalletQuery(workspace, providedQuery)
	if err != nil {
		return fws.GitRepoQuery{}, errors.Wrapf(
			err, "couldn't complete provided version query %s", providedQuery,
		)
	}

	printed := false
	if !provided.Complete() {
		fmt.Fprintf(
			os.Stderr,
			"Provided query %s was completed (based on stored query %s) as %s!\n",
			provided, loaded, query,
		)
		printed = true
	}
	if query == loaded {
		if printed {
			fmt.Fprintln(os.Stderr)
		}
		return query, nil
	}

	if !commitPalletQuery {
		fmt.Fprintf(
			os.Stderr,
			"Using (but not saving) the path & version query: %s\n", query,
		)
		return query, nil
	}

	if loaded == (fws.GitRepoQuery{}) {
		fmt.Fprintf(
			os.Stderr,
			"Initializing the tracked path & version query for the current pallet as %s...\n", query,
		)
	} else {
		fmt.Fprintf(
			os.Stderr,
			"Updating the tracked path & version query for the current pallet from %s to %s...\n",
			loaded, query,
		)
	}
	if err := workspace.CommitCurrentPalletUpgrades(query); err != nil {
		return query, errors.Wrapf(err, "couldn't commit pallet query %s", query)
	}
	fmt.Fprintln(os.Stderr)
	return query, nil
}

func completePalletQuery(
	workspace *fws.FSWorkspace, providedQuery string,
) (query, loaded, provided fws.GitRepoQuery, err error) {
	if providedQuery == "" {
		providedQuery = "@"
	}
	pltPath, versionQuery, ok := strings.Cut(providedQuery, "@")
	if !ok {
		return fws.GitRepoQuery{}, fws.GitRepoQuery{}, fws.GitRepoQuery{}, errors.Errorf(
			"couldn't parse '%s' as [pallet_path]@[version_query]", providedQuery,
		)
	}
	provided = fws.GitRepoQuery{
		Path:         pltPath,
		VersionQuery: versionQuery,
	}
	if loaded, err = workspace.GetCurrentPalletUpgrades(); err != nil {
		if !provided.Complete() {
			return fws.GitRepoQuery{}, fws.GitRepoQuery{}, provided, errors.Wrap(
				err, "couldn't load stored query for the current pallet",
			)
		}
		loaded = fws.GitRepoQuery{}
	}
	query = loaded.Overlay(provided)

	if !query.Complete() {
		return query, loaded, provided, errors.Errorf(
			"provided query %s could not be fully completed with stored query %s", provided, loaded,
		)
	}

	return query, loaded, provided, nil
}

const (
	notGitSnippet       = "is not a valid Git repo"
	noBackupSnippet     = "(i.e. not yet backed up)"
	uncommittedSnippet  = "has changes which are not yet saved in a Git commit " + noBackupSnippet
	unpushedSnippet     = "is on a commit which might not be in a remote Git repo " + noBackupSnippet
	existsSnippet       = "the local pallet already exists"
	forceEnabledSnippet = "we can only delete and replace such pallets if the --force flag is enabled"
	evenThoughSnippet   = "we will delete and replace the local pallet even though"
)

func checkPalletDirtiness(workspace *fws.FSWorkspace, force bool) error {
	pltPath := workspace.GetCurrentPalletPath()
	if !ffs.DirExists(pltPath) {
		return nil
	}

	gitRepo, err := git.Open(pltPath)
	if err != nil {
		if !force {
			return errors.Errorf(
				existsSnippet + " and " + notGitSnippet + ", but " + forceEnabledSnippet,
			)
		}
		fmt.Fprintln(os.Stderr, "Warning: "+evenThoughSnippet+" "+notGitSnippet+"!")
	}

	status, err := gitRepo.Status()
	if err != nil {
		return errors.Wrapf(err, "couldn't check status of %s as a Git repo", pltPath)
	}
	if len(status) > 0 {
		if !force {
			return errors.Errorf(
				existsSnippet + " and " + uncommittedSnippet + "; to ignore this, enable the --force flag",
			)
		}
		fmt.Fprintln(os.Stderr, "Warning: "+evenThoughSnippet+" it "+uncommittedSnippet+"!")
	}

	fmt.Fprintln(os.Stderr, "Fetching changes from the remote...")
	if err = gitRepo.FetchAll(1, os.Stdout); err != nil {
		fcli.IndentedFprintf(
			1, os.Stderr,
			"Warning: couldn't fetch changes (maybe you don't have internet, or maybe the repo doesn't "+
				"exist?): %s\n", err,
		)
		fcli.IndentedFprintln(1, os.Stderr, "We may be able to continue anyways, so we'll keep going!")
	}

	fmt.Fprintf(
		os.Stderr, "Checking whether current commit of %s exists on a remote Git repo...\n", pltPath,
	)
	remotesHaveHead, err := isHeadInRemotes(1, gitRepo)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't check whether current commit of %s exists on a remote Git repo", pltPath,
		)
	}
	if !remotesHaveHead {
		if !force {
			return errors.Errorf(existsSnippet + " and " + unpushedSnippet + ", " + forceEnabledSnippet)
		}
		fmt.Fprintln(os.Stderr, "Warning: "+evenThoughSnippet+" it "+unpushedSnippet+"!")
	}

	return nil
}

func isHeadInRemotes(indent int, gitRepo *git.Repo) (bool, error) {
	refs, err := getRemoteRefs(indent, gitRepo)
	if err != nil {
		return false, errors.Wrapf(err, "couldn't retrieve references of remotes")
	}

	head, err := gitRepo.GetHead()
	if err != nil {
		return false, errors.Wrapf(err, "couldn't determine the current Git commit")
	}
	const shortHashLength = 7
	fcli.IndentedFprintf(
		indent, os.Stderr,
		"Searching ancestors of retrieved remote references for current commit %s...\n",
		head[:shortHashLength],
	)
	// Warning: the following function call assumes that head commits are available to the git repo,
	// which requires the git repo to have fetched all updated heads from the remotes!
	remotesHaveHead, err := gitRepo.RefsHaveAncestor(refs, head)
	if err != nil {
		fcli.IndentedFprintln(indent, os.Stderr, errors.Wrapf(
			err, "Warning: couldn't check whether remotes have commit %s", head,
		))
	}
	if remotesHaveHead {
		fcli.IndentedFprintf(
			indent, os.Stderr, "Found current commit %s in one of the remotes!\n", head[:shortHashLength],
		)
	}
	return remotesHaveHead, nil
}

func getRemoteRefs(indent int, gitRepo *git.Repo) ([]*plumbing.Reference, error) {
	remotes, err := gitRepo.Remotes()
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't check Git remotes")
	}
	fcli.SortRemotes(remotes)

	refs := make([]*plumbing.Reference, 0)
	queryCacheMirrorRemote := false
	for _, remote := range remotes {
		if remote.Config().Name == forklift.ForkliftCacheMirrorRemoteName && !queryCacheMirrorRemote {
			fcli.IndentedFprintf(
				indent, os.Stderr,
				"Skipped remote %s, because remote origin's references were successfully retrieved!\n",
				remote.Config().Name,
			)
			continue
		}

		remoteRefs, err := remote.List(git.EmptyListOptions())
		if err != nil {
			fcli.IndentedFprintf(indent, os.Stderr, "Warning: %s\n", errors.Wrapf(
				err, "couldn't retrieve references for remote %s", remote.Config().Name,
			))
			if remote.Config().Name == forklift.OriginRemoteName {
				queryCacheMirrorRemote = true
			}
			continue
		}
		fcli.IndentedFprintf(
			indent, os.Stderr, "Retrieved references for remote %s!\n", remote.Config().Name,
		)
		for _, ref := range remoteRefs {
			if strings.HasPrefix(string(ref.Name()), "refs/pull/") {
				continue
			}
			refs = append(refs, ref)
		}
	}
	return refs, nil
}

func preparePallet(
	workspace *fws.FSWorkspace, gitRepoQuery fws.GitRepoQuery,
	updateLocalMirror, cacheStagingReqs bool, platform string, parallel, ignoreToolVersion bool,
	versions Versions,
) error {
	// clone pallet
	if err := fcli.CloneQueriedGitRepoUsingLocalMirror(
		0, workspace.GetMirrorCachePath(), gitRepoQuery.Path, gitRepoQuery.VersionQuery,
		workspace.GetCurrentPalletPath(), updateLocalMirror,
	); err != nil {
		return err
	}
	// TODO: warn if the git repo doesn't appear to be an actual pallet

	plt, caches, err := processFullBaseArgs(workspace.FS.Path(), processingOptions{})
	if err != nil {
		return err
	}

	if err = fcli.CheckPltShallowCompat(plt, versions.Core(), ignoreToolVersion); err != nil {
		return err
	}

	// cache everything required by pallet
	if cacheStagingReqs {
		fmt.Fprintln(os.Stderr)
		if _, _, err = fcli.CacheStagingReqs(
			0, plt, caches.m, caches.p, caches.d, platform, false, parallel,
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

		query, err := handlePalletQuery(workspace, c.Args().First(), c.Bool("set-upgrade-query"))
		if err != nil {
			return errors.Wrapf(err, "couldn't handle provided version query %s", c.Args().First())
		}
		if err = checkUpgrade(0, workspace, query, c.Bool("allow-downgrade")); err != nil {
			return err
		}

		if err = checkPalletDirtiness(workspace, c.Bool("force")); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Deleting the local pallet to replace it with %s...", query)
		fmt.Fprintln(os.Stderr)
		if err := os.RemoveAll(workspace.GetCurrentPalletPath()); err != nil {
			return errors.Wrap(err, "couldn't remove local pallet")
		}

		if err = preparePallet(
			// Note: we don't cache staging requirements because that will be handled by the apply/stage
			// step anyways:
			workspace, query, false, false, c.String("platform"), c.Bool("parallel"),
			c.Bool("ignore-tool-version"), versions,
		); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr)

		if c.Bool("apply") {
			return applyAction(versions)(c)
		}
		return stageAction(versions)(c)
	}
}

func checkUpgrade(
	indent int, workspace *fws.FSWorkspace, upgradeQuery fws.GitRepoQuery,
	allowDowngrade bool,
) error {
	fcli.IndentedFprintln(indent, os.Stderr, "Resolving upgrade version query...")
	upgradeResolved, err := fcli.ResolveQueriesUsingLocalMirrors(
		indent+1, workspace.GetMirrorCachePath(), []string{upgradeQuery.String()}, true,
	)
	if err != nil {
		return errors.Wrap(err, "couldn't resolve upgrade version query")
	}

	currentResolved, err := resolveCurrentPalletVersion(indent, workspace)
	if err != nil {
		currentResolved = fplt.GitRepoReq{}
		fcli.IndentedFprintf(indent, os.Stderr, "Warning: %s\n", errors.Wrap(
			err,
			"we couldn't determine & resolve the current version of the local pallet, so any change "+
				"could be either an upgrade or a downgrade",
		))
	}

	fmt.Fprintln(os.Stderr)
	return printUpgrade(
		indent, currentResolved, upgradeResolved[upgradeQuery.String()], allowDowngrade,
	)
	// TODO: also report whether the update is cached
}

func resolveCurrentPalletVersion(
	indent int, workspace *fws.FSWorkspace,
) (resolved fplt.GitRepoReq, err error) {
	// Inspect the current plt
	plt, err := workspace.GetCurrentPallet()
	if err != nil {
		return fplt.GitRepoReq{}, errors.Wrap(err, "couldn't load local pallet from workspace")
	}
	ref, err := git.Head(plt.FS.Path())
	if err != nil {
		return fplt.GitRepoReq{}, errors.Wrap(
			err, "couldn't determine current commit of local pallet",
		)
	}
	currentQuery := fws.GitRepoQuery{
		Path:         plt.Decl.Pallet.Path,
		VersionQuery: ref.Hash().String(),
	}
	fcli.IndentedFprintf(
		indent, os.Stderr,
		"Local pallet currently is %s at %s\n", plt.Decl.Pallet.Path, git.StringifyRef(ref),
	)
	indent++
	fcli.IndentedFprintln(indent, os.Stderr, "Resolving current version query...")
	currentResolved, err := fcli.ResolveQueriesUsingLocalMirrors(
		// Note: we don't update the local mirror because we already updated it to resolve the current
		// version query
		indent, workspace.GetMirrorCachePath(), []string{currentQuery.String()}, false,
	)
	if err != nil {
		fcli.IndentedFprintf(indent, os.Stderr, "Warning: %s\n", errors.Wrap(
			err,
			"couldn't resolve current version query from the Forklift pallet cache's local mirror of "+
				"the remote repo (is the local pallet currently on a commit not in the remote origin?)",
		))
		fcli.IndentedFprintln(
			indent, os.Stderr, "Resolving current version query using local pallet instead...",
		)
		resolvedVersionLock, err := forklift.ResolveVersionQueryUsingRepo(
			plt.FS.Path(), currentQuery.VersionQuery,
		)
		if err != nil {
			return fplt.GitRepoReq{}, errors.Wrap(
				err, "couldn't resolve current version query from the local pallet",
			)
		}

		fcli.IndentedFprintf(
			indent, os.Stderr, "Resolved %s as %s@%s",
			currentQuery.String(), plt.Decl.Pallet.Path, resolvedVersionLock.Version,
		)
		return fplt.GitRepoReq{
			RequiredPath: plt.Decl.Pallet.Path,
			VersionLock:  resolvedVersionLock,
		}, nil
	}

	return currentResolved[currentQuery.String()], nil
}

func printUpgrade(indent int, current, upgrade fplt.GitRepoReq, allowDowngrade bool) error {
	if current == upgrade {
		return errors.New("no upgrade found")
	}
	if current == (fplt.GitRepoReq{}) {
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
		fcli.IndentedFprintf(
			indent, os.Stderr,
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
		fmt.Fprintf(os.Stderr, "Loaded upgrade query: %s\n", query)
	} else if !provided.Complete() {
		fmt.Fprintf(
			os.Stderr,
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

	_, err = handlePalletQuery(workspace, c.Args().First(), true)
	if err != nil {
		return errors.Wrapf(err, "couldn't handle provided version query %s", c.Args().First())
	}
	fmt.Fprintln(os.Stderr, "Done!")
	return nil
}

// clone

func cloneAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		workspace, err := ensureWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}

		query, err := handlePalletQuery(workspace, c.Args().First(), c.Bool("set-upgrade-query"))
		if err != nil {
			return errors.Wrapf(err, "couldn't handle provided version query %s", c.Args().First())
		}

		if c.Bool("force") {
			fmt.Fprintf(
				os.Stderr,
				"Warning: if a local pallet already exists, it will be deleted now to be replaced with "+
					"%s...\n",
				query,
			)
			fmt.Fprintln(os.Stderr)
			if err := os.RemoveAll(workspace.GetCurrentPalletPath()); err != nil {
				return errors.Wrap(err, "couldn't remove local pallet")
			}
		}

		if err = preparePallet(
			workspace, query, true, c.Bool("cache-req"), c.String("platform"), c.Bool("parallel"),
			c.Bool("ignore-tool-version"), versions,
		); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr)

		switch {
		case c.Bool("apply"):
			return applyAction(versions)(c)
		case c.Bool("stage"):
			return stageAction(versions)(c)
		default:
			fmt.Fprintln(os.Stderr, "Done!")
			return nil
		}
	}
}

// fetch

func fetchAction(c *cli.Context) error {
	workspace, err := fws.LoadWorkspace(c.String("workspace"))
	if err != nil {
		return err
	}
	pltPath := workspace.GetCurrentPalletPath()

	fmt.Fprintln(os.Stderr, "Fetching updates...")
	updated, err := git.Fetch(0, pltPath, os.Stdout)
	if err != nil {
		return errors.Wrap(err, "couldn't fetch changes from the remote release")
	}
	if !updated {
		fmt.Fprintln(os.Stderr, "No updates from the remote release.")
	}

	// TODO: display changes
	return nil
}

// pull

func pullAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		workspace, err := fws.LoadWorkspace(c.String("workspace"))
		if err != nil {
			return err
		}
		pltPath := workspace.GetCurrentPalletPath()

		// FIXME: update the local mirror

		fmt.Fprintln(os.Stderr, "Attempting to fast-forward the local pallet...")
		updated, err := git.Pull(1, pltPath, os.Stderr)
		if err != nil {
			return errors.Wrap(err, "couldn't fast-forward the local pallet")
		}
		if !updated {
			fmt.Fprintln(os.Stderr, "No changes from the remote release.")
		}
		// TODO: display changes

		fmt.Fprintln(os.Stderr)

		plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{})
		if err != nil {
			return err
		}
		if err = fcli.CheckPltShallowCompat(
			plt, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if c.Bool("cache-req") {
			if _, _, err = fcli.CacheStagingReqs(
				0, plt, caches.m, caches.p, caches.d,
				c.String("platform"), false, c.Bool("parallel"),
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
			fmt.Fprintln(os.Stderr, "Done!")
			return nil
		}
	}
}

// del

func delAction(c *cli.Context) error {
	workspace, err := fws.LoadWorkspace(c.String("workspace"))
	if err != nil {
		return err
	}
	pltPath := workspace.GetCurrentPalletPath()

	fmt.Fprintf(os.Stderr, "Removing local pallet from workspace...\n")
	// TODO: return an error if there are uncommitted or unpushed changes to be removed - in which
	// case require a --force flag
	return errors.Wrap(os.RemoveAll(pltPath), "couldn't remove local pallet")
}

// show

func showAction(c *cli.Context) error {
	plt, err := getShallowPallet(c.String("workspace"))
	if err != nil {
		return err
	}
	return fcli.FprintPalletInfo(0, os.Stdout, plt)
}

// check

func checkAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{
			requirePalletCache: true,
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
		plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{
			requirePalletCache: true,
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
		plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{})
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
			os.Stderr,
			"Done! To apply the staged pallet immediately, run `forklift stage apply` (or "+
				"`sudo -E forklift stage apply` if you need sudo for Docker).",
		)
		return nil
	}
}

// apply

func applyAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{})
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
		plt, err := getShallowPallet(c.String("workspace"))
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
	plt, err := getShallowPallet(c.String("workspace"))
	if err != nil {
		return err
	}

	return fcli.FprintRequiredPallets(0, os.Stdout, plt)
}

// show-plt

func showPltAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{
		requirePalletCache: true,
	})
	if err != nil {
		return err
	}

	return fcli.FprintRequiredPalletInfo(0, os.Stdout, plt, caches.p, c.Args().First())
}

// show-plt-version

func showPltVersionAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{})
	if err != nil {
		return err
	}

	return fcli.FprintRequiredPalletVersion(0, os.Stdout, plt, caches.p, c.Args().First())
}

// add-plt

func addPltAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, err := getShallowPallet(c.String("workspace"))
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
			plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{})
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
		plt, err := getShallowPallet(c.String("workspace"))
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
	plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{})
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
	plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{})
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
	plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{})
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
	plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{})
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
	plt, caches, err := processFullBaseArgs(c.String("workspace"), processingOptions{})
	if err != nil {
		return err
	}

	plt, err = forklift.GetRequiredPallet(plt, caches.p, c.Args().First())
	if err != nil {
		return nil
	}
	return fcli.FprintFeatureInfo(0, os.Stdout, plt, caches.p, c.Args().Get(1))
}
