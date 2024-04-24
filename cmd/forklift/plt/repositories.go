package plt

import (
	"fmt"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// cache-repo

func cacheRepoAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, cache, err := processFullBaseArgs(c.String("workspace"), false)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, cache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		fmt.Println("Downloading repos specified by the local pallet...")
		changed, err := fcli.DownloadRequiredRepos(0, pallet, cache.Path())
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
	pallet, err := getPallet(c.String("workspace"))
	if err != nil {
		return err
	}

	return fcli.PrintPalletRepos(0, pallet)
}

// show-repo

func showRepoAction(c *cli.Context) error {
	pallet, cache, err := processFullBaseArgs(c.String("workspace"), true)
	if err != nil {
		return err
	}

	return fcli.PrintRepoInfo(0, pallet, cache, c.Args().First())
}

// add-repo

func addRepoAction(versions Versions) cli.ActionFunc {
	return func(c *cli.Context) error {
		pallet, repoCache, err := processFullBaseArgs(c.String("workspace"), false)
		if err != nil {
			return err
		}
		if err = fcli.CheckShallowCompatibility(
			pallet, repoCache, versions.Tool, versions.MinSupportedRepo, versions.MinSupportedPallet,
			c.Bool("ignore-tool-version"),
		); err != nil {
			return err
		}

		if err = fcli.AddRepoRequirements(0, pallet, repoCache.Path(), c.Args().Slice()); err != nil {
			return err
		}

		if !c.Bool("no-cache-req") {
			if _, err = fcli.CacheStagingRequirements(pallet, repoCache.Path()); err != nil {
				return err
			}
		}
		fmt.Println("Done!")
		return nil
	}
}
