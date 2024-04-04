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
			Name:     "check",
			Category: category,
			Usage:    "Checks whether the next staged pallet's resource constraints are satisfied",
			Action:   checkAction(versions),
		},
		{
			Name:     "plan",
			Category: category,
			Usage: "Determines the changes needed to update the host to match the deployments " +
				"specified by the next staged pallet",
			Action: planAction(versions),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "parallel",
					Usage: "parallelize downloading of images",
				},
			},
		},
		{
			Name:     "apply",
			Category: category,
			Usage: "Updates the host to match the package deployments specified by the next staged " +
				"pallet",
			Action: applyAction(versions),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "parallel",
					Usage: "parallelize updating of package deployments",
				},
			},
		},
	}
}

func makeQuerySubcmds(versions Versions) []*cli.Command {
	const category = "Query the stage store"
	return []*cli.Command{
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
			Name:      "set-next",
			Category:  category,
			Usage:     "Sets the specified staged pallet bundle as the next one to be applied.",
			ArgsUsage: "bundle_index_or_name",
			Action:    setNextAction(versions),
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
		// TODO: add a prune command which deletes bundles not in the history and not in any names
	}
}
