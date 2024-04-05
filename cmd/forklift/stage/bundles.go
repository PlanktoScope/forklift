package stage

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-bun

func lsBunAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), versions)
		if err != nil {
			return err
		}
		if !store.Exists() {
			return errMissingStore
		}

		names := getBundleNames(store)
		indices, err := store.List()
		if err != nil {
			return err
		}
		for _, index := range indices {
			printBundleSummary(store, index, names)
		}
		return nil
	}
}

func getBundleNames(store *forklift.FSStageStore) map[int][]string {
	names := make(map[int][]string)
	for name, index := range store.Def.Stages.Names {
		names[index] = append(names[index], name)
	}
	for _, indexNames := range names {
		slices.Sort(indexNames)
	}
	if index, ok := store.GetRollback(); ok {
		names[index] = slices.Concat([]string{rollbackStageName}, names[index])
	}
	if index, ok := store.GetNext(); ok {
		names[index] = slices.Concat([]string{nextStageName}, names[index])
	}
	if index, ok := store.GetCurrent(); ok {
		names[index] = slices.Concat([]string{currentStageName}, names[index])
	}
	if index, ok := store.GetPending(); ok {
		names[index] = slices.Concat([]string{pendingStageName}, names[index])
	}
	return names
}

func printBundleSummary(store *forklift.FSStageStore, index int, names map[int][]string) {
	bundle, err := store.LoadFSBundle(index)
	if err != nil {
		fmt.Printf("%d: Error: couldn't load bundle: %s\n", index, err)
		return
	}
	fmt.Print(index)
	if indexNames := names[index]; len(indexNames) > 0 {
		fmt.Printf(" (%s)", strings.Join(indexNames, ", "))
	}
	fmt.Printf(": %s@%s", bundle.Def.Pallet.Path, bundle.Def.Pallet.Version)
	if !bundle.Def.Pallet.Clean {
		fmt.Print(" (staged with uncommitted pallet changes)")
	}
	if bundle.Def.Includes.HasOverrides() {
		fmt.Print(" (staged with overridden pallet requirements)")
	}
	fmt.Println()
}

// show-bun

func showBunAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), versions)
		if err != nil {
			return err
		}
		if !store.Exists() {
			return errMissingStore
		}

		index, err := resolveBundleIdentifier(c.Args().First(), store)
		if err != nil {
			return err
		}
		bundle, err := store.LoadFSBundle(index)
		if err != nil {
			return errors.Wrapf(err, "couldn't load staged bundle %d", index)
		}
		fcli.PrintStagedBundle(0, store, bundle, index, getBundleNames(store)[index])
		return nil
	}
}

// rm-bun

func rmBunAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), versions)
		if err != nil {
			return err
		}
		if !store.Exists() {
			return errMissingStore
		}

		deleteIndex, err := resolveBundleIdentifier(c.Args().First(), store)
		if err != nil {
			return err
		}
		fmt.Printf("Staged pallet bundle %d will be deleted...\n", deleteIndex)
		preNext, preHasNext := store.GetNext()
		preCurrent, preHasCurrent := store.GetCurrent()

		store.RemoveBundleNames(deleteIndex)
		store.RemoveBundleHistory(deleteIndex)
		postCurrent, postHasCurrent := store.GetCurrent()
		switch {
		case preHasCurrent && !postHasCurrent:
			fmt.Println(
				"Warning: now there will be no staged pallet bundle known to have been successfully " +
					"applied!",
			)
		case postHasCurrent && postCurrent != preCurrent:
			fmt.Printf(
				"The last staged pallet bundle known to have been successfully applied will change "+
					"from %d to %d!\n",
				preCurrent, postCurrent,
			)
		}
		switch {
		case preHasNext && (preNext == deleteIndex) && postHasCurrent:
			store.SetNext(postCurrent)
			fmt.Printf(
				"Because pallet bundle %d will be deleted, %d will now be the next staged pallet bundle "+
					"to be applied!\n",
				deleteIndex, postCurrent,
			)
		case preHasNext && (preNext == deleteIndex) && !postHasCurrent:
			store.SetNext(0)
			fmt.Printf(
				"Because bundle %d will be deleted, and there is no remaining successfully-applied pallet "+
					"bundle in the store's history, now no bundle will be applied next!\n",
				deleteIndex,
			)
		}
		fmt.Println("Saving the updated state of the stage store...")
		// Note: we commit the state before deleting the stage (rather than the other way around)
		// because it's better to accidentally leave the stage on the filesystem than to have indices
		// of deleted bundles in our history/names/next-state.
		if err = store.CommitState(); err != nil {
			return errors.Wrap(err, "couldn't commit the stage store's new state!")
		}
		fmt.Println("Deleting the staged pallet bundle from the filesystem...")
		return os.RemoveAll(filepath.FromSlash(store.GetBundlePath(deleteIndex)))
	}
}
