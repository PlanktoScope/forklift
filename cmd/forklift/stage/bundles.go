package stage

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

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

		indices, err := store.List()
		if err != nil {
			return err
		}
		for _, index := range indices {
			bundle, err := store.LoadFSBundle(index)
			if err != nil {
				fmt.Printf("%d: Error: couldn't load bundle: %s\n", index, err)
				continue
			}
			fmt.Printf("%d: %s@%s", index, bundle.Def.Pallet.Path, bundle.Def.Pallet.Version)
			if !bundle.Def.Pallet.Clean {
				fmt.Print(" (staged with uncommitted pallet changes)")
			}
			if bundle.Def.Includes.HasOverrides() {
				fmt.Print(" (staged with overridden pallet requirements)")
			}
			// TODO: add label for the last successfully-applied bundle, and the next one staged to be
			// deployed, i.e. the pending apply (if it exists), and the rollback bundle (if it exists)
			fmt.Println()
		}
		return nil
	}
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

		rawIndex := c.Args().First()
		index, err := strconv.Atoi(rawIndex)
		if err != nil {
			return errors.Wrapf(err, "Couldn't parse staged bundle index %s as an integer", rawIndex)
		}
		bundle, err := store.LoadFSBundle(index)
		if err != nil {
			return errors.Wrapf(err, "couldn't load staged bundle %d", index)
		}
		fcli.PrintStagedBundle(0, store, bundle, index)
		return nil
	}
}
