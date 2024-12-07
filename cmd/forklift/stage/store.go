package stage

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

var errMissingStore = errors.Errorf(
	"no pallets have been staged yet: you first must stage a pallet, e.g. with `forklift plt stage`",
)

func loadNextBundle(
	wpath, sspath string, versions Versions,
) (*forklift.FSBundle, *forklift.FSStageStore, error) {
	store, err := getStageStore(wpath, sspath, versions)
	if err != nil {
		return nil, nil, err
	}
	if !store.Exists() {
		return nil, store, errMissingStore
	}

	next, ok := store.GetNext()
	if !ok {
		return nil, store, errors.Errorf(
			"no next staged pallet bundle to apply: you first must set a pallet to stage next, " +
				"e.g. with `forklift stage set-next`",
		)
	}
	if store.NextFailed() {
		fmt.Printf("Next stage failed in the past: %d\n", next)
		current, ok := store.GetCurrent()
		switch {
		case !ok:
			return nil, store, errors.Errorf(
				"the next staged pallet bundle already failed, and no staged pallet bundle was " +
					"applied successfully in the past, so we have no fallback!",
			)
		case current != next:
			fmt.Printf("Current stage will be used instead, as a fallback: %d\n", current)
		default:
			fmt.Println("Trying again, since it had succeeded in the past!")
		}
		next = current
	} else {
		if pending, ok := store.GetPending(); ok && next == pending {
			fmt.Printf("Next stage is pending: %d\n", next)
		} else if current, ok := store.GetCurrent(); ok && next == current {
			fmt.Printf("Next stage previously had a successful apply: %d\n", next)
		} else {
			fmt.Printf("Next stage: %d\n", next)
		}
	}

	bundle, err := store.LoadFSBundle(next)
	if err != nil {
		return nil, store, errors.Wrapf(err, "couldn't load staged pallet bundle %d", next)
	}
	return bundle, store, nil
}

func getStageStore(
	wpath, sspath string, versions Versions,
) (store *forklift.FSStageStore, err error) {
	var workspace *forklift.FSWorkspace
	if sspath == "" {
		workspace, err = forklift.LoadWorkspace(wpath)
		if err != nil {
			return nil, errors.Wrap(
				err, "couldn't load workspace to load the stage store, since no explicit path was "+
					"provided for the stage store",
			)
		}
	}
	if store, err = fcli.GetStageStore(workspace, sspath, versions.NewStageStore); err != nil {
		return nil, err
	}
	return store, nil
}

// show

func showAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), c.String("stage-store"), versions)
		if err != nil {
			return err
		}
		if !store.Exists() {
			return errMissingStore
		}

		indent := 0
		fcli.IndentedPrintf(indent, "Stage store %s:\n", store.Path())
		indent++
		names := getBundleNames(store)

		// TODO: display info about whether the swapfile exists
		fcli.IndentedPrint(indent, "Next staged pallet bundle to be applied:")
		next, hasNext := store.GetNext()
		if !hasNext {
			fmt.Println(" (none)")
		} else {
			fmt.Printf(" %d\n", next)
			printNextSummary(indent+1, store, next, names[next])
		}

		fcli.IndentedPrint(indent, "Last successfully-applied staged pallet bundle:")
		current, hasCurrent := store.GetCurrent()
		switch {
		case !hasCurrent:
			fmt.Println(" (none)")
		case next == current:
			fmt.Printf(" %d (see above)\n", current)
		default:
			fmt.Printf(" %d\n", current)
			printCurrentSummary(indent+1, store, current, names[current])
		}

		fcli.IndentedPrint(indent, "Previous successfully-applied staged pallet bundle:")
		rollback, hasRollback := store.GetRollback()
		if !hasRollback {
			fmt.Println(" (none)")
		} else {
			fmt.Printf(" %d\n", rollback)
			printRollbackSummary(indent+1, store, rollback, names[rollback])
		}

		return nil
	}
}

func printNextSummary(indent int, store *forklift.FSStageStore, index int, names []string) {
	bundle, err := store.LoadFSBundle(index)
	if err != nil {
		fmt.Printf("Error: couldn't load staged bundle d (was it deleted?): %s\n", err.Error())
		return
	}

	printBasicSummary(indent, bundle, names)
	failed := store.NextFailed()
	pending, hasPending := store.GetPending()
	isPending := (hasPending && index == pending)
	current, hasCurrent := store.GetCurrent()
	isCurrent := (hasCurrent && index == current)
	if failed || isPending || isCurrent {
		fcli.IndentedPrint(indent, "Status: ")
		switch {
		case failed && hasCurrent:
			fmt.Printf(
				"failed to be applied; the last successfully-applied staged pallet bundle (%d) will be "+
					"used instead\n",
				current,
			)
		case failed:
			fmt.Println(
				"failed to be applied; no pallet will be applied until another pallet is staged",
			)
		case isPending:
			fmt.Println("not yet applied; will be used for the next apply")
		case hasCurrent && index == current:
			fmt.Println("already applied; will still be used for the next apply")
		}
	}
}

func printBasicSummary(indent int, bundle *forklift.FSBundle, names []string) {
	fcli.IndentedPrint(indent, "Staged names:")
	if len(names) == 0 {
		fmt.Println(" (none)")
	} else {
		fmt.Println()
		for _, name := range names {
			fcli.BulletedPrintln(indent+1, name)
		}
	}

	fcli.IndentedPrintf(
		indent, "Pallet: %s@%s\n", bundle.Manifest.Pallet.Path, bundle.Manifest.Pallet.Version,
	)

	indent++
	if !bundle.Manifest.Pallet.Clean {
		fcli.BulletedPrintln(indent, "Staged with uncommitted pallet changes")
	}
	if bundle.Manifest.Includes.HasOverrides() {
		fcli.BulletedPrint(indent, "Staged with overridden pallet requirements")
	}
}

func printCurrentSummary(indent int, store *forklift.FSStageStore, index int, names []string) {
	bundle, err := store.LoadFSBundle(index)
	if err != nil {
		fmt.Printf("Error: couldn't load staged bundle %d (was it deleted?): %s\n", index, err.Error())
		return
	}

	printBasicSummary(indent, bundle, names)
}

func printRollbackSummary(indent int, store *forklift.FSStageStore, index int, names []string) {
	bundle, err := store.LoadFSBundle(index)
	if err != nil {
		fmt.Printf("Error: couldn't load staged bundle %d (was it deleted?): %s\n", index, err.Error())
		return
	}

	printBasicSummary(indent, bundle, names)
}

// show-hist

func showHistAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), c.String("stage-store"), versions)
		if err != nil {
			return err
		}
		if !store.Exists() {
			return errMissingStore
		}

		names := getBundleNames(store)
		for _, index := range store.Manifest.Stages.History {
			printBundleSummary(store, index, names)
		}
		if index, ok := store.GetPending(); ok {
			printBundleSummary(store, index, names)
		}
		return nil
	}
}

// show-next-index

func showNextIndexAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), c.String("stage-store"), versions)
		if err != nil {
			return err
		}
		if !store.Exists() {
			return errMissingStore
		}

		next, ok := store.GetNext()
		if !ok {
			return errors.New("there is currently no staged pallet bundle to be applied next!")
		}
		fmt.Println(next)
		return nil
	}
}

// set-next

func setNextAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), c.String("stage-store"), versions)
		if err != nil {
			return err
		}
		if !store.Exists() {
			return errMissingStore
		}

		if c.Args().First() == "0" {
			store.SetNext(0)
			fmt.Println(
				"Committing update to the stage store so that no staged pallet bundle will be applied " +
					"next...",
			)
			if err := store.CommitState(); err != nil {
				return errors.Wrap(err, "couldn't commit updated stage store state")
			}
			return nil
		}

		newNext, err := resolveBundleIdentifier(c.Args().First(), store)
		if err != nil {
			return err
		}
		if _, err = store.LoadFSBundle(newNext); err != nil {
			return errors.Wrapf(err, "couldn't load staged bundle %d", newNext)
		}

		if next, hasNext := store.GetNext(); hasNext {
			fmt.Printf("Changing the next staged pallet bundle from %d to %d...\n", next, newNext)
		} else {
			fmt.Printf("Setting the next staged pallet bundle to %d...\n", newNext)
		}

		if err = fcli.SetNextStagedBundle(
			0, store, newNext, c.String("exports"), versions.Tool, versions.MinSupportedBundle,
			c.Bool("no-cache-img"), c.String("platform"), c.Bool("parallel"),
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}
		fmt.Println(
			"Done! To apply the staged pallet, you may need to reboot or run " +
				"`forklift stage apply` (or `sudo -E forklift stage apply` if you need sudo for Docker).",
		)
		return nil
	}
}

// unset-next

func unsetNextAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), c.String("stage-store"), versions)
		if err != nil {
			return err
		}
		if !store.Exists() {
			return errMissingStore
		}

		store.SetNext(0)
		fmt.Println("Committing update to the stage store so that no stage will be applied next...")
		if err := store.CommitState(); err != nil {
			return errors.Wrap(err, "couldn't commit updated stage store state")
		}
		return nil
	}
}

// resolveBundleIdentifier parses/resolves a staged bundle index or name (provided as a string)
// into an index of a staged bundle in the store.
func resolveBundleIdentifier(
	identifier string, store *forklift.FSStageStore,
) (index int, err error) {
	index, indexParseErr := strconv.Atoi(identifier)
	if indexParseErr == nil {
		return index, nil
	}

	// TODO: add special handling for rollback, current, next, and pending names
	switch identifier {
	case rollbackStageName:
		index, ok := store.GetRollback()
		if !ok {
			return 0, errors.New(
				"there have not yet been enough successfully-applied staged bundles for a rollback " +
					"stage to exist yet!",
			)
		}
		return index, nil
	case currentStageName:
		index, ok := store.GetCurrent()
		if !ok {
			return 0, errors.New("there has yet been a successfully-applied staged bundle!")
		}
		return index, nil
	case nextStageName:
		index, ok := store.GetNext()
		if !ok {
			return 0, errors.New("no staged bundle has been set as the next one to apply!")
		}
		return index, nil
	case pendingStageName:
		index, ok := store.GetPending()
		if !ok {
			if currentIndex, ok := store.GetCurrent(); ok && index == currentIndex {
				return 0, errors.New(
					"the next staged bundle has already been applied successfully, so it's no longer " +
						"pending!",
				)
			}
			if _, ok := store.GetNext(); !ok {
				return 0, errors.New("no staged bundle has been set as the next one to apply!")
			}
			return 0, errors.New(
				"there is currently no staged bundle waiting to be applied for the first time!",
			)
		}
		return index, nil
	}

	index, ok := store.Manifest.Stages.Names[identifier]
	if !ok {
		return 0, errors.Errorf(
			"identifier %s is neither a staged bundle index nor a name assigned to a staged bundle!",
			identifier,
		)
	}
	return index, nil
}

// check

func checkAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		bundle, _, err := loadNextBundle(c.String("workspace"), c.String("stage-store"), versions)
		if err != nil {
			return err
		}
		if err = fcli.CheckBundleShallowCompat(
			bundle, versions.Tool, versions.MinSupportedBundle, c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}
		if _, _, err = fcli.Check(0, bundle, bundle); err != nil {
			return err
		}
		return nil
	}
}

// plan

func planAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		bundle, _, err := loadNextBundle(c.String("workspace"), c.String("stage-store"), versions)
		if err != nil {
			return err
		}
		if err = fcli.CheckBundleShallowCompat(
			bundle, versions.Tool, versions.MinSupportedBundle, c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}
		if _, _, err = fcli.Plan(0, bundle, bundle, c.Bool("parallel")); err != nil {
			return err
		}
		return nil
	}
}

// apply

func applyAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		bundle, store, err := loadNextBundle(c.String("workspace"), c.String("stage-store"), versions)
		if err != nil {
			return err
		}
		if err = fcli.CheckBundleShallowCompat(
			bundle, versions.Tool, versions.MinSupportedBundle, c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}
		fmt.Println()

		if err = fcli.ApplyNextOrCurrentBundle(0, store, bundle, c.Bool("parallel")); err != nil {
			return err
		}
		fmt.Println("Done!")
		return nil
	}
}

// set-next-result

func setNextResultAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), c.String("stage-store"), versions)
		if err != nil {
			return err
		}
		if !store.Exists() {
			return errMissingStore
		}

		switch result := c.Args().First(); result {
		case "success":
			store.RecordNextSuccess(true)
		case "pending":
			store.Manifest.Stages.NextFailed = false
		case "failure":
			store.RecordNextSuccess(false)
		default:
			return errors.Errorf(
				"unknown result (must be 'pending', 'success', or 'failure'): %s", result,
			)
		}
		fmt.Println("Committing result to the stage store...")
		if err := store.CommitState(); err != nil {
			return errors.Wrap(err, "couldn't commit updated stage store state")
		}
		fmt.Println("Done!")
		return nil
	}
}
