package stage

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// show-bun-depl

func showBunDeplAction(versions Versions) cli.ActionFunc {
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
		deplName := c.Args().Get(1)
		bundle, err := store.LoadFSBundle(index)
		if err != nil {
			return errors.Wrapf(err, "couldn't load staged bundle %d", index)
		}
		resolved, err := bundle.LoadResolvedDepl(deplName)
		if err != nil {
			return errors.Wrapf(err, "couldn't load deployment %s from bundle %d", deplName, index)
		}
		return fcli.PrintResolvedDepl(0, bundle, resolved)
	}
}

// locate-bun-depl-pkg

func locateBunDeplPkgAction(versions Versions) cli.ActionFunc {
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
		deplName := c.Args().Get(1)
		resolved, err := bundle.LoadResolvedDepl(deplName)
		if err != nil {
			return errors.Wrapf(
				err, "couldn't load deployment %s from bundle %s", deplName, bundle.FS.Path(),
			)
		}
		fmt.Println(resolved.Pkg.FS.Path())
		return nil
	}
}
