package stage

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
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
		if index, ok := store.GetNext(); ok {
			printNamedBundleSummary(store, "next", index)
		}
		if index, ok := store.GetCurrent(); ok {
			printNamedBundleSummary(store, "current", index)
		}
		if index, ok := store.GetRollback(); ok {
			printNamedBundleSummary(store, "rollback", index)
		}
		for _, name := range names {
			index := store.Def.Stages.Names[name]
			printNamedBundleSummary(store, name, index)
			// TODO: add label for the last successfully-applied bundle, and the next one staged to be
			// deployed, i.e. the pending apply (if it exists), and the rollback bundle (if it exists)
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
		if name == "next" || name == "current" || name == "rollback" {
			return errors.Errorf("'%s' is an automatically-set name, so it can't be set manually", name)
		}
		if _, err := strconv.Atoi(name); err == nil {
			return errors.Errorf("integers cannot be used as bundle names: %s", name)
		}

		rawIndex := c.Args().Get(1)
		index, err := strconv.Atoi(rawIndex)
		if err != nil {
			return errors.Wrapf(err, "Couldn't parse staged bundle index %s as an integer", rawIndex)
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
		if name == "next" || name == "current" || name == "rollback" {
			return errors.Errorf("'%s' is not allowed to be manually set as a name", name)
		}
		if _, err := strconv.Atoi(name); err == nil {
			return errors.Errorf("integers cannot be used as bundle names: %s", name)
		}

		delete(store.Def.Stages.Names, name)
		return store.CommitState()
	}
}
