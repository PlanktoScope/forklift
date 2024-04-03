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
			makeQuerySubcmds(versions),
			makeModifySubcmds(versions),
		),
	}
}

func makeQuerySubcmds(versions Versions) []*cli.Command {
	const category = "Query the stage store"
	return []*cli.Command{
		// TODO: add a show-history command
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
			ArgsUsage: "bundle_index",
			Action:    showBunAction(versions),
		},
		{
			Name:      "show-bun-depl",
			Aliases:   []string{"show-bundle-deployment"},
			Category:  category,
			Usage:     "Describes the specified package deployment of the specified staged pallet bundle",
			ArgsUsage: "bundle_index deployment_name",
			Action:    showBunDeplAction(versions),
		},
		{
			Name:     "locate-bun-depl-pkg",
			Aliases:  []string{"locate-bundle-deployment-package"},
			Category: category,
			Usage: "Prints the absolute filesystem path of the package for the specified package " +
				"deployment of the specified staged pallet bundle",
			ArgsUsage: "bundle_index deployment_name",
			Action:    locateBunDeplPkgAction(versions),
		},
		{
			Name:     "ls-bun-names",
			Category: category,
			Usage:    "Lists all named staged pallet bundles",
			Action:   lsBunNamesAction(versions),
		},
	}
}

func makeModifySubcmds(versions Versions) []*cli.Command {
	return []*cli.Command{
		{
			Name:     "add-bun-name",
			Aliases:  []string{"add-bundle-name"},
			Category: "Modify the stage store",
			Usage: "Assigns a name to the specified staged pallet bundle; if the name was already " +
				"assigned, it's reassigned",
			ArgsUsage: "bundle_name bundle_index",
			Action:    addBunNameAction(versions),
		},
		{
			Name:      "rm-bun-name",
			Aliases:   []string{"remove-bundle-name"},
			Category:  "Modify the stage store",
			Usage:     "Unsets a name for a staged pallet bundle",
			ArgsUsage: "bundle_name",
			Action:    rmBunNameAction(versions),
		},
		// TODO: add a prune command which deletes bundles not in the history and not in any names
	}
}
