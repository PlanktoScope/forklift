// Package depl provides subcommands for the active deployment
package depl

import (
	"github.com/urfave/cli/v2"
)

var Cmd = &cli.Command{
	Name:    "depl",
	Aliases: []string{"d", "deployments"},
	Usage:   "Manages active Pallet package deployments in the local environment",
	Subcommands: []*cli.Command{
		{
			Name:     "ls-stack",
			Category: "Query the active deployment",
			Aliases:  []string{"ls-s", "list-stacks"},
			Usage:    "Lists running Docker stacks",
			Action:   lsStackAction,
		},
		{
			Name:     "rm",
			Category: "Modify the active deployment",
			Aliases:  []string{"remove"},
			Usage:    "Removes all Docker stacks",
			Action:   rmAction,
		},
	},
}
