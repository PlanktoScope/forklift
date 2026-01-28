package plt

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	fcli "github.com/forklift-run/forklift/internal/app/forklift/cli"
)

// ls-depl

func lsDeplAction(c *cli.Context) error {
	plt, _, err := processFullBaseArgs(c, processingOptions{
		enableOverrides: true,
		merge:           true,
	})
	if err != nil {
		return err
	}

	return fcli.FprintPalletDepls(0, os.Stdout, plt)
}

// show-depl

func showDeplAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		requireRepoCache: true,
		enableOverrides:  true,
		merge:            true,
	})
	if err != nil {
		return err
	}

	return fcli.FprintDeplInfo(0, os.Stdout, plt, caches.p, c.Args().First())
}

// locate-depl-pkg

func locateDeplPkgAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		requireRepoCache: true,
		enableOverrides:  true,
		merge:            true,
	})
	if err != nil {
		return err
	}

	return fcli.FprintDeplPkgLocation(
		0, os.Stdout, plt, caches.p, c.Args().First(), c.Bool("allow-disabled"),
	)
}

// add-depl

func addDeplAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requirePalletCache: true,
			requireRepoCache:   true,
			enableOverrides:    true,
			merge:              true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		deplName := c.Args().Slice()[0]
		pkgPath := c.Args().Slice()[1]
		if err = fcli.AddDepl(
			0, plt, caches.p, deplName, pkgPath, c.StringSlice("feat"), c.Bool("disabled"),
			c.Bool("force"),
		); err != nil {
			return err
		}

		switch {
		case c.Bool("apply"):
			return applyAction(versions)(c)
		case c.Bool("stage"):
			return stageAction(versions)(c)
		default:
			fmt.Fprintln(os.Stderr, "Done!")
			return nil
		}
	}
}

// del-depl

func delDeplAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requirePalletCache: true,
			requireRepoCache:   true,
			enableOverrides:    true,
			merge:              true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if err = fcli.RemoveDepls(0, plt, c.Args().Slice()); err != nil {
			return err
		}

		switch {
		case c.Bool("apply"):
			return applyAction(versions)(c)
		case c.Bool("stage"):
			return stageAction(versions)(c)
		default:
			fmt.Fprintln(os.Stderr, "Done!")
			return nil
		}
	}
}

// set-depl-pkg

func setDeplPkgAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requirePalletCache: true,
			requireRepoCache:   true,
			enableOverrides:    true,
			merge:              true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		deplName := c.Args().Slice()[0]
		pkgPath := c.Args().Slice()[1]
		if err = fcli.SetDeplPkg(0, plt, caches.p, deplName, pkgPath, c.Bool("force")); err != nil {
			return err
		}

		switch {
		case c.Bool("apply"):
			return applyAction(versions)(c)
		case c.Bool("stage"):
			return stageAction(versions)(c)
		default:
			fmt.Fprintln(os.Stderr, "Done!")
			return nil
		}
	}
}

// add-depl-feat

func addDeplFeatAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requirePalletCache: true,
			requireRepoCache:   true,
			enableOverrides:    true,
			merge:              true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		deplName := c.Args().Slice()[0]
		features := c.Args().Slice()[1:]
		if err = fcli.AddDeplFeat(0, plt, caches.p, deplName, features, c.Bool("force")); err != nil {
			return err
		}

		switch {
		case c.Bool("apply"):
			return applyAction(versions)(c)
		case c.Bool("stage"):
			return stageAction(versions)(c)
		default:
			fmt.Fprintln(os.Stderr, "Done!")
			return nil
		}
	}
}

// del-depl-feat

func delDeplFeatAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requirePalletCache: true,
			requireRepoCache:   true,
			enableOverrides:    true,
			merge:              true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		deplName := c.Args().Slice()[0]
		features := c.Args().Slice()[1:]
		if err = fcli.RemoveDeplFeat(0, plt, deplName, features); err != nil {
			return err
		}

		switch {
		case c.Bool("apply"):
			return applyAction(versions)(c)
		case c.Bool("stage"):
			return stageAction(versions)(c)
		default:
			fmt.Fprintln(os.Stderr, "Done!")
			return nil
		}
	}
}

// set-depl-disabled

func setDeplDisabledAction(versions Versions, setting bool) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requirePalletCache: true,
			requireRepoCache:   true,
			enableOverrides:    true,
			merge:              true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		deplName := c.Args().Slice()[0]
		if err = fcli.SetDeplDisabled(0, plt, deplName, setting); err != nil {
			return err
		}

		switch {
		case c.Bool("apply"):
			return applyAction(versions)(c)
		case c.Bool("stage"):
			return stageAction(versions)(c)
		default:
			fmt.Fprintln(os.Stderr, "Done!")
			return nil
		}
	}
}
