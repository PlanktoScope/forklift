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
		/*{
			Name:     "rm-bun",
			Aliases:  []string{"remove-bundles"},
			Category: "Modify the stage store",
			Usage:    "Removes all staged pallet bundles",
			Action:   rmAction,
		},*/
	}
	return &cli.Command{
		Name:        "stage",
		Usage:       "Manages the local stage store",
		Subcommands: subcommands,
	}
}
