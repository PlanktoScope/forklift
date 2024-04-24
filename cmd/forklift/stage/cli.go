// Package stage provides subcommands for the working with staged pallet bundles.
package stage

import (
	"slices"

	"github.com/urfave/cli/v2"
)

type Versions struct {
	Tool               string
	MinSupportedBundle string // TODO: use this for version-checking when applying a bundle
	NewStageStore      string
}

func MakeCmd(versions Versions) *cli.Command {
	return &cli.Command{
		Name:  "stage",
		Usage: "Manages the local stage store",
		Subcommands: slices.Concat(
			makeUseSubcmds(versions),
			makeQuerySubcmds(versions),
			makeModifySubcmds(versions),
		),
	}
}

func makeUseSubcmds(versions Versions) []*cli.Command {
	const category = "Use the stage store"
	return []*cli.Command{
		{
			Name:     "cache-img",
			Aliases:  []string{"cache-images"},
			Category: category,
			Usage:    "Pre-downloads the Docker container images required for the next apply",
			Action:   cacheImgAction(versions),
		},
		{
			Name:     "check",
			Category: category,
			Usage:    "Checks whether the next resource constraints are satisfied for the next apply",
			Action:   checkAction(versions),
		},
		{
			Name:     "plan",
			Category: category,
			Usage:    "Determines the changes needed to update the host for the next apply",
			Action:   planAction(versions),
		},
		{
			Name:     "apply",
			Category: category,
			Usage: "Updates the host according to the next staged pallet, falling back to the last " +
				"successfully-staged pallet if the next one already failed",
			Action: applyAction(versions),
		},
	}
}

func makeQuerySubcmds(versions Versions) []*cli.Command {
	const category = "Query the stage store"
	return append(
		[]*cli.Command{
			{
				Name:     "show",
				Category: category,
				Usage:    "Describes the state of the stage store",
				Action:   showAction(versions),
			},
			{
				Name:     "show-hist",
				Aliases:  []string{"show-history"},
				Category: category,
				Usage:    "Shows the history of successfully-applied staged pallet bundles",
				Action:   showHistAction(versions),
			},
		},
		makeQueryBunSubcmds(versions)...,
	)
}

func makeQueryBunSubcmds(versions Versions) []*cli.Command {
	const category = "Query the stage store"
	return []*cli.Command{
		{
			Name:     "ls-bun-names",
			Aliases:  []string{"list-bundle-names"},
			Category: category,
			Usage:    "Lists all named staged pallet bundles",
			Action:   lsBunNamesAction(versions),
		},
		{
			Name:     "ls-bun",
			Aliases:  []string{"list-bundles"},
			Category: category,
			Usage:    "Lists staged pallet bundles",
			Action:   lsBunAction(versions),
		},
		{
			Name:      "show-bun",
			Aliases:   []string{"show-bundle"},
			Category:  category,
			Usage:     "Describes a staged pallet bundle",
			ArgsUsage: "bundle_index_or_name",
			Action:    showBunAction(versions),
		},
		{
			Name:      "show-bun-depl",
			Aliases:   []string{"show-bundle-deployment"},
			Category:  category,
			Usage:     "Describes the specified package deployment of the specified staged pallet bundle",
			ArgsUsage: "bundle_index_or_name deployment_name",
			Action:    showBunDeplAction(versions),
		},
		{
			Name:      "locate-bun",
			Aliases:   []string{"locate-bundle"},
			Category:  category,
			Usage:     "Prints the absolute filesystem path of the specified staged pallet bundle",
			ArgsUsage: "bundle_index_or_name",
			Action:    locateBunAction(versions),
		},
		{
			Name:     "locate-bun-depl-pkg",
			Aliases:  []string{"locate-bundle-deployment-package"},
			Category: category,
			Usage: "Prints the absolute filesystem path of the package for the specified package " +
				"deployment of the specified staged pallet bundle",
			ArgsUsage: "bundle_index_or_name deployment_name",
			Action:    locateBunDeplPkgAction(versions),
		},
	}
}

func makeModifySubcmds(versions Versions) []*cli.Command {
	category := "Modify the stage store"
	return []*cli.Command{
		{
			Name:     "set-next",
			Category: category,
			Usage: "Sets the specified staged pallet bundle as the next one to be applied, then " +
				"caches required images",
			ArgsUsage: "bundle_index_or_name",
			Action:    setNextAction(versions),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "no-cache-img",
					Usage: "don't download container images",
				},
			},
		},
		{
			Name:     "add-bun-name",
			Aliases:  []string{"add-bundle-name"},
			Category: category,
			Usage: "Assigns the specified name to the specified staged pallet bundle; if the name was " +
				"already assigned, it's reassigned",
			ArgsUsage: "bundle_name_to_assign bundle_index_or_name",
			Action:    addBunNameAction(versions),
		},
		{
			Name:      "rm-bun-name",
			Aliases:   []string{"remove-bundle-name"},
			Category:  category,
			Usage:     "Unsets a name for a staged pallet bundle",
			ArgsUsage: "bundle_name",
			Action:    rmBunNameAction(versions),
		},
		{
			Name:      "rm-bun",
			Aliases:   []string{"remove-bundle"},
			Category:  category,
			Usage:     "Deletes the specified staged pallet bundle",
			ArgsUsage: "bundle_index_or_name",
			Action:    rmBunAction(versions),
		},
		{
			Name:     "prune-bun",
			Aliases:  []string{"prune-bundles"},
			Category: category,
			Usage:    "Deletes all staged pallet bundles not referenced in names or in the history",
			Action:   pruneBunAction(versions),
		},
	}
}
