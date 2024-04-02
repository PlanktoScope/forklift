// Package stage provides subcommands for the working with staged pallet bundles.
package stage

import (
	"github.com/urfave/cli/v2"
)

var Cmd = &cli.Command{
	Subcommands: []*cli.Command{},
}

func MakeCmd(toolVersion, newStageVersion string) *cli.Command {
	subcommands := []*cli.Command{
		{
			Name:     "ls-bun",
			Aliases:  []string{"list-bundles"},
			Category: "Query the stage store",
			Usage:    "Lists staged pallet bundles",
			Action:   lsBunAction,
		},
		{
			Name:      "show-bun",
			Aliases:   []string{"show-bundle"},
			Category:  "Query the stage store",
			Usage:     "Describes a staged pallet bundle",
			ArgsUsage: "index",
			Action:    showBunAction,
		},
	}
	return &cli.Command{
		Name:        "stage",
		Usage:       "Manages the local stage store",
		Subcommands: subcommands,
	}
}
