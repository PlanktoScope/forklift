package stage

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-bundle

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
