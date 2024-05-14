package plt

import (
	"fmt"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-depl

func lsDeplAction(c *cli.Context) error {
	pallet, cache, _, err := processFullBaseArgs(c.String("workspace"), true)
	if err != nil {
		return err
	}

	return fcli.PrintPalletDepls(0, pallet, cache)
}

// show-depl

func showDeplAction(c *cli.Context) error {
	pallet, cache, _, err := processFullBaseArgs(c.String("workspace"), true)
	if err != nil {
		return err
	}

	deplName := c.Args().First()
	return fcli.PrintDeplInfo(0, pallet, cache, deplName)
}

// locate-depl-pkg

func locateDeplPkgAction(c *cli.Context) error {
	pallet, cache, _, err := processFullBaseArgs(c.String("workspace"), true)
	if err != nil {
		return err
	}

	deplName := c.Args().First()
	return fcli.PrintDeplPkgPath(0, pallet, cache, deplName, c.Bool("allow-disabled"))
}

// add-depl

func addDeplAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, repoCache, _, err := processFullBaseArgs(c.String("workspace"), true)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, repoCache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		deplName := c.Args().Slice()[0]
		pkgPath := c.Args().Slice()[1]
		if err = fcli.AddDepl(
			0, pallet, repoCache, deplName, pkgPath, c.StringSlice("feature"), c.Bool("disabled"),
			c.Bool("force"),
		); err != nil {
			return err
		}

		fmt.Println("Done!")
		return nil
	}
}

// rm-depl

func rmDeplAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, repoCache, _, err := processFullBaseArgs(c.String("workspace"), false)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, repoCache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if err = fcli.RemoveDepls(0, pallet, c.Args().Slice()); err != nil {
			return err
		}

		fmt.Println("Done!")
		return nil
	}
}

// add-depl-feat

func addDeplFeatAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, repoCache, _, err := processFullBaseArgs(c.String("workspace"), true)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, repoCache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		deplName := c.Args().Slice()[0]
		features := c.Args().Slice()[1:]
		if err = fcli.AddDeplFeat(
			0, pallet, repoCache, deplName, features, c.Bool("force"),
		); err != nil {
			return err
		}

		fmt.Println("Done!")
		return nil
	}
}

// rm-depl-feat

func rmDeplFeatAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, repoCache, _, err := processFullBaseArgs(c.String("workspace"), true)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, repoCache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		deplName := c.Args().Slice()[0]
		features := c.Args().Slice()[1:]
		if err = fcli.RemoveDeplFeat(0, pallet, deplName, features); err != nil {
			return err
		}

		fmt.Println("Done!")
		return nil
	}
}
