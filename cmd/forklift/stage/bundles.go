package stage

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	fcli "github.com/forklift-run/forklift/internal/app/forklift/cli"
	"github.com/forklift-run/forklift/pkg/staging"
	"github.com/forklift-run/forklift/pkg/structures"
)

// ls-bun

func lsBunAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), c.String("stage-store"), versions)
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

func getBundleNames(store *staging.FSStageStore) map[int][]string {
	names := make(map[int][]string)
	for name, index := range store.Manifest.Stages.Names {
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

func printBundleSummary(store *staging.FSStageStore, index int, names map[int][]string) {
	bundle, err := store.LoadFSBundle(index)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%d: Error: couldn't load bundle: %s\n", index, err)
		return
	}
	fmt.Print(index)
	if indexNames := names[index]; len(indexNames) > 0 {
		fmt.Printf(" (%s)", strings.Join(indexNames, ", "))
	}
	fmt.Printf(": %s@%s", bundle.Manifest.Pallet.Path, bundle.Manifest.Pallet.Version)
	if !bundle.Manifest.Pallet.Clean {
		fmt.Print(" (staged with uncommitted pallet changes)")
	}
	if bundle.Manifest.Includes.HasOverrides() {
		fmt.Print(" (staged with overridden pallet requirements)")
	}
	fmt.Println()
}

// show-bun

func showBunAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), c.String("stage-store"), versions)
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
		fcli.FprintStagedBundle(0, os.Stdout, store, bundle, index, getBundleNames(store)[index])
		return nil
	}
}

// locate-bun

func locateBunAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), c.String("stage-store"), versions)
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
		fmt.Println(store.GetBundlePath(index))
		return nil
	}
}

// del-bun

const knownSnippet = "last staged pallet bundle known to have been successfully applied"

func delBunAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), c.String("stage-store"), versions)
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
		fmt.Fprintf(os.Stderr, "Staged pallet bundle %d will be deleted...\n", deleteIndex)
		preNext, preHasNext := store.GetNext()
		preCurrent, preHasCurrent := store.GetCurrent()

		store.RemoveBundleNames(deleteIndex)
		store.RemoveBundleHistory(deleteIndex)
		postCurrent, postHasCurrent := store.GetCurrent()
		switch {
		case preHasCurrent && !postHasCurrent:
			fmt.Fprintln(os.Stderr, "Warning: now there will be no "+knownSnippet+"!")
		case postHasCurrent && postCurrent != preCurrent:
			fmt.Fprintf(
				os.Stderr, "The "+knownSnippet+" will change from %d to %d!\n", preCurrent, postCurrent,
			)
		}
		switch {
		case preHasNext && (preNext == deleteIndex) && postHasCurrent:
			store.SetNext(postCurrent)
			fmt.Fprintf(
				os.Stderr,
				"Because pallet bundle %d will be deleted, %d will now be the next staged pallet bundle "+
					"to be applied!\n",
				deleteIndex, postCurrent,
			)
		case preHasNext && (preNext == deleteIndex) && !postHasCurrent:
			store.SetNext(0)
			fmt.Fprintf(
				os.Stderr,
				"Because bundle %d will be deleted, and there is no remaining successfully-applied pallet "+
					"bundle in the store's history, now no bundle will be applied next!\n",
				deleteIndex,
			)
		}
		fmt.Fprintln(os.Stderr, "Saving the updated state of the stage store...")
		// Note: we commit the state before deleting the stage (rather than the other way around)
		// because it's better to accidentally leave the stage on the filesystem than to have indices
		// of deleted bundles in our history/names/next-state.
		if err = store.CommitState(); err != nil {
			return errors.Wrap(err, "couldn't commit the stage store's new state!")
		}
		fmt.Fprintln(os.Stderr, "Deleting the staged pallet bundle from the filesystem...")
		return os.RemoveAll(filepath.FromSlash(store.GetBundlePath(deleteIndex)))
	}
}

// prune-bun

func pruneBunAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), c.String("stage-store"), versions)
		if err != nil {
			return err
		}
		if !store.Exists() {
			return errMissingStore
		}

		bundleNames := getBundleNames(store)
		allIndices, err := store.List()
		if err != nil {
			return err
		}
		historyIndices := make(structures.Set[int])
		for _, index := range store.Manifest.Stages.History {
			historyIndices.Add(index)
		}

		deleteIndices := make([]int, 0, len(allIndices))
		for _, index := range allIndices {
			if len(bundleNames[index]) > 0 || historyIndices.Has(index) {
				continue
			}
			deleteIndices = append(deleteIndices, index)
		}
		if len(deleteIndices) == 0 {
			fmt.Fprintln(os.Stderr, "There are no staged pallet bundles to prune!")
			return nil
		}

		fmt.Fprintf(os.Stderr, "Deleting staged pallet bundles: %+v\n", deleteIndices)
		failedIndices := make([]int, 0, len(deleteIndices))
		for _, index := range deleteIndices {
			if err = os.RemoveAll(filepath.FromSlash(store.GetBundlePath(index))); err != nil {
				failedIndices = append(failedIndices, index)
			}
		}
		if len(failedIndices) > 0 {
			return errors.Errorf("couldn't delete some staged pallet bundles: %+v", failedIndices)
		}
		return nil
	}
}
