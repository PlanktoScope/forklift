// Package stage provides subcommands for the working with staged pallet bundles.
package stage

import (
	"github.com/urfave/cli/v2"
)

type Versions struct {
	Tool              string
	MinSupportedStage string // TODO: add version-checking for stage versions
	NewStageStore     string
}

var Cmd = &cli.Command{
	Subcommands: []*cli.Command{},
}

func MakeCmd(versions Versions) *cli.Command {
	subcommands := []*cli.Command{
		{
			Name:     "ls-bun",
			Aliases:  []string{"list-bundles"},
			Category: "Query the stage store",
			Usage:    "Lists staged pallet bundles",
			Action:   lsBunAction(versions),
		},
		{
			Name:      "show-bun",
			Aliases:   []string{"show-bundle"},
			Category:  "Query the stage store",
			Usage:     "Describes a staged pallet bundle",
			ArgsUsage: "bundle_index",
			Action:    showBunAction(versions),
		},
		{
			Name:      "show-bun-depl",
			Aliases:   []string{"show-bundle-deployment"},
			Category:  "Query the stage store",
			Usage:     "Describes the specified package deployment of the specified staged pallet bundle",
			ArgsUsage: "bundle_index deployment_name",
			Action:    showBunDeplAction(versions),
		},
		{
			Name:     "locate-bun-depl-pkg",
			Aliases:  []string{"locate-bundle-deployment-package"},
			Category: "Query the stage store",
			Usage: "Prints the absolute filesystem path of the package for the specified package " +
				"deployment of the specified staged pallet bundle",
			ArgsUsage: "bundle_index deployment_name",
			Action:    locateBunDeplPkgAction(versions),
		},
	}
	return &cli.Command{
		Name:        "stage",
		Usage:       "Manages the local stage store",
		Subcommands: subcommands,
	}
}
