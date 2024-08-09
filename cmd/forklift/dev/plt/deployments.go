package plt

import (
	"fmt"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-depl

func lsDeplAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		enableOverrides: true,
		merge:           true,
	})
	if err != nil {
		return err
	}

	return fcli.PrintPalletDepls(0, plt, caches.r)
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

	return fcli.PrintDeplInfo(0, plt, caches.r, c.Args().First())
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

	return fcli.PrintDeplPkgLocation(0, plt, caches.r, c.Args().First(), c.Bool("allow-disabled"))
}

// add-depl

func addDeplAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requireRepoCache: true,
			enableOverrides:  true,
			merge:            true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, caches.r, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		deplName := c.Args().Slice()[0]
		pkgPath := c.Args().Slice()[1]
		if err = fcli.AddDepl(
			0, plt, caches.r, deplName, pkgPath, c.StringSlice("feature"), c.Bool("disabled"),
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
			fmt.Println("Done!")
			return nil
		}
	}
}

// rm-depl

func rmDeplAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requireRepoCache: true,
			enableOverrides:  true,
			merge:            true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, caches.r, versions.Core(), c.Bool("ignore-tool-version"),
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
			fmt.Println("Done!")
			return nil
		}
	}
}

// set-depl-pkg

func setDeplPkgAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requireRepoCache: true,
			enableOverrides:  true,
			merge:            true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, caches.r, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		deplName := c.Args().Slice()[0]
		pkgPath := c.Args().Slice()[1]
		if err = fcli.SetDeplPkg(0, plt, caches.r, deplName, pkgPath, c.Bool("force")); err != nil {
			return err
		}

		switch {
		case c.Bool("apply"):
			return applyAction(versions)(c)
		case c.Bool("stage"):
			return stageAction(versions)(c)
		default:
			fmt.Println("Done!")
			return nil
		}
	}
}

// add-depl-feat

func addDeplFeatAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requireRepoCache: true,
			enableOverrides:  true,
			merge:            true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, caches.r, versions.Core(), c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		deplName := c.Args().Slice()[0]
		features := c.Args().Slice()[1:]
		if err = fcli.AddDeplFeat(
			0, plt, caches.r, deplName, features, c.Bool("force"),
		); err != nil {
			return err
		}

		switch {
		case c.Bool("apply"):
			return applyAction(versions)(c)
		case c.Bool("stage"):
			return stageAction(versions)(c)
		default:
			fmt.Println("Done!")
			return nil
		}
	}
}

// rm-depl-feat

func rmDeplFeatAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requireRepoCache: true,
			enableOverrides:  true,
			merge:            true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, caches.r, versions.Core(), c.Bool("ignore-tool-version"),
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
			fmt.Println("Done!")
			return nil
		}
	}
}

// set-depl-disabled

func setDeplDisabledAction(versions Versions, setting bool) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			requireRepoCache: true,
			enableOverrides:  true,
			merge:            true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckDeepCompat(
			plt, caches.p, caches.r, versions.Core(), c.Bool("ignore-tool-version"),
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
			fmt.Println("Done!")
			return nil
		}
	}
}
