package plt

import (
	"fmt"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// cache-repo

func cacheRepoAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, cache, _, err := processFullBaseArgs(c, false, false)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		fmt.Printf("Downloading repos specified by the development pallet...\n")
		changed, err := fcli.DownloadRequiredRepos(0, pallet, cache.Underlay.Path())
		if err != nil {
			return err
		}
		if !changed {
			fmt.Println("Done! No further actions are needed at this time.")
			return nil
		}

		// TODO: warn if any downloaded repo doesn't appear to be an actual repo, or if any repo's
		// forklift version is incompatible or ahead of the pallet version
		fmt.Println("Done!")
		return nil
	}
}

// ls-repo

func lsRepoAction(c *cli.Context) error {
	pallet, err := getPallet(c.String("cwd"))
	if err != nil {
		return err
	}
	return fcli.PrintPalletRepos(0, pallet)
}

// show-repo

func showRepoAction(c *cli.Context) error {
	pallet, cache, _, err := processFullBaseArgs(c, true, true)
	if err != nil {
		return err
	}

	return fcli.PrintRepoInfo(0, pallet, cache, c.Args().First())
}

// add-repo

func addRepoAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, repoCache, dlCache, err := processFullBaseArgs(c, false, false)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if err = fcli.AddRepoRequirements(
			0, pallet, repoCache.Underlay.Path(), c.Args().Slice(),
		); err != nil {
			return err
		}

		if !c.Bool("no-cache-req") {
			if err = fcli.CacheStagingRequirements(
				0, pallet, repoCache.Path(), repoCache, dlCache, false, c.Bool("parallel"),
			); err != nil {
				return err
			}
		}
		fmt.Println("Done!")
		return nil
	}
}

// rm-repo

func rmRepoAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, _, _, err := processFullBaseArgs(c, false, false)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if err = fcli.RemoveRepoRequirements(0, pallet, c.Args().Slice(), c.Bool("force")); err != nil {
			return err
		}

		fmt.Println("Done!")
		return nil
	}
}
