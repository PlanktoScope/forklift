package plt

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// cache-repo

func cacheRepoAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			enableOverrides: true,
			merge:           true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckPltCompat(plt, versions.Core(), c.Bool("ignore-tool-version")); err != nil {
			return err
		}

		changed, err := fcli.DownloadAllRequiredRepos(
			0, plt, caches.m, caches.p, caches.r.Underlay, nil,
		)
		if err != nil {
			return err
		}
		if !changed {
			fmt.Fprintln(os.Stderr, "Done! No further actions are needed at this time.")
			return nil
		}

		// TODO: warn if any downloaded repo doesn't appear to be an actual repo, or if any repo's
		// forklift version is incompatible or ahead of the pallet version
		fmt.Fprintln(os.Stderr, "Done!")
		return nil
	}
}

// ls-repo

func lsRepoAction(c *cli.Context) error {
	plt, _, err := processFullBaseArgs(c, processingOptions{
		enableOverrides: true,
		merge:           true,
	})
	if err != nil {
		return err
	}

	return fcli.FprintRequiredRepos(0, os.Stdout, plt)
}

// locate-repo

func locateRepoAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		requireRepoCache: true,
		enableOverrides:  true,
		merge:            true,
	})
	if err != nil {
		return err
	}

	return fcli.FprintRequiredRepoLocation(os.Stdout, plt, caches.r, c.Args().First())
}

// show-repo

func showRepoAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		requireRepoCache: true,
		enableOverrides:  true,
		merge:            true,
	})
	if err != nil {
		return err
	}

	return fcli.FprintRequiredRepoInfo(0, os.Stdout, plt, caches.r, c.Args().First())
}

// show-repo-version

func showRepoVersionAction(c *cli.Context) error {
	plt, caches, err := processFullBaseArgs(c, processingOptions{
		enableOverrides: true,
		merge:           true,
	})
	if err != nil {
		return err
	}

	return fcli.FprintRequiredRepoVersion(0, os.Stdout, plt, caches.r, c.Args().First())
}

// add-repo

func addRepoAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, caches, err := processFullBaseArgs(c, processingOptions{
			enableOverrides: true,
			merge:           true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckPltCompat(plt, versions.Core(), c.Bool("ignore-tool-version")); err != nil {
			return err
		}

		if err = fcli.AddRepoReqs(0, plt, caches.m.Path(), c.Args().Slice()); err != nil {
			return err
		}
		if !c.Bool("no-cache-req") {
			if _, _, err = fcli.CacheStagingReqs(
				0, plt, caches.m, caches.p, caches.r, caches.d,
				c.String("platform"), false, c.Bool("parallel"),
			); err != nil {
				return err
			}
			// TODO: check version compatibility between the pallet and the added repo!
		}
		fmt.Fprintln(os.Stderr, "Done!")
		return nil
	}
}

// del-repo

func delRepoAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		plt, _, err := processFullBaseArgs(c, processingOptions{
			enableOverrides: true,
		})
		if err != nil {
			return err
		}
		if err = fcli.CheckPltCompat(plt, versions.Core(), c.Bool("ignore-tool-version")); err != nil {
			return err
		}

		if err = fcli.RemoveRepoReqs(0, plt, c.Args().Slice(), c.Bool("force")); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Done!")
		return nil
	}
}
