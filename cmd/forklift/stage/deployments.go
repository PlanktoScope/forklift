package stage

import (
	"strconv"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// show-bun-depl

func showBunDeplAction(c *cli.Context) error {
	store, err := getStageStore(c.String("workspace"))
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
	deplName := c.Args().Get(1)
	bundle, err := store.LoadFSBundle(index)
	if err != nil {
		return errors.Wrapf(err, "couldn't load staged bundle %d", index)
	}
	resolved, err := bundle.LoadDepl(deplName)
	if err != nil {
		return errors.Wrapf(err, "couldn't load deployment %s from bundle %d", deplName, index)
	}
	return fcli.PrintResolvedDepl(0, bundle, resolved)
}

// locate-bun-depl-pkg

func locateBunDeplPkgAction(c *cli.Context) error {
	store, err := getStageStore(c.String("workspace"))
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
	deplName := c.Args().Get(1)
	return fcli.PrintBundleDeplPkgPath(0, bundle, deplName)
}
