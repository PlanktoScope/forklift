package stage

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
)

const (
	rollbackStageName = "rollback"
	currentStageName  = "current"
	nextStageName     = "next"
	pendingStageName  = "pending"
)

// ls-bun-names

func lsBunNamesAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), versions)
		if err != nil {
			return err
		}
		if !store.Exists() {
			return errMissingStore
		}

		names := make([]string, 0, len(store.Def.Stages.Names))
		for name := range store.Def.Stages.Names {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			index := store.Def.Stages.Names[name]
			printNamedBundleSummary(store, name, index)
		}
		if index, ok := store.GetRollback(); ok {
			printNamedBundleSummary(store, rollbackStageName, index)
		}
		if index, ok := store.GetCurrent(); ok {
			printNamedBundleSummary(store, currentStageName, index)
		}
		if index, ok := store.GetNext(); ok {
			printNamedBundleSummary(store, nextStageName, index)
		}
		if index, ok := store.GetPending(); ok {
			printNamedBundleSummary(store, pendingStageName, index)
		}
		return nil
	}
}

func printNamedBundleSummary(store *forklift.FSStageStore, name string, index int) {
	bundle, err := store.LoadFSBundle(index)
	if err != nil {
		fmt.Printf(
			"%s -> %d: Error: couldn't load bundle (was it deleted?): %s\n", name, index, err.Error(),
		)
		return
	}

	fmt.Printf("%s -> %d: %s@%s", name, index, bundle.Def.Pallet.Path, bundle.Def.Pallet.Version)
	if !bundle.Def.Pallet.Clean {
		fmt.Print(" (staged with uncommitted pallet changes)")
	}
	if bundle.Def.Includes.HasOverrides() {
		fmt.Print(" (staged with overridden pallet requirements)")
	}
	fmt.Println()
}

// add-bun-name

func addBunNameAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), versions)
		if err != nil {
			return err
		}
		if !store.Exists() {
			return errMissingStore
		}

		name := c.Args().First()
		if name == rollbackStageName || name == currentStageName ||
			name == nextStageName || name == pendingStageName {
			return errors.Errorf("'%s' is an automatically-set name, so it can't be set manually", name)
		}
		if _, err := strconv.Atoi(name); err == nil {
			return errors.Errorf("integers cannot be used as bundle names: %s", name)
		}

		index, err := resolveBundleIdentifier(c.Args().Get(1), store)
		if err != nil {
			return err
		}
		if _, err = store.LoadFSBundle(index); err != nil {
			return errors.Wrapf(err, "couldn't load staged bundle %d", index)
		}

		store.Def.Stages.Names[name] = index
		return store.CommitState()
	}
}

// rm-bun-name

func rmBunNameAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		store, err := getStageStore(c.String("workspace"), versions)
		if err != nil {
			return err
		}
		if !store.Exists() {
			return errMissingStore
		}

		name := c.Args().First()
		if name == rollbackStageName || name == currentStageName ||
			name == nextStageName || name == pendingStageName {
			return errors.Errorf("'%s' is an automatically-set name, so it can't be removed", name)
		}
		if _, err := strconv.Atoi(name); err == nil {
			return errors.Errorf("integers cannot be used as bundle names: %s", name)
		}

		delete(store.Def.Stages.Names, name)
		return store.CommitState()
	}
}
