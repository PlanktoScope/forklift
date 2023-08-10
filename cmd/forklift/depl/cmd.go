// Package depl provides subcommands for the active deployment
package depl

import (
	"github.com/urfave/cli/v2"
)

var Cmd = &cli.Command{
	Name:    "depl",
	Aliases: []string{"d", "deployments"},
	Usage:   "Manages active package deployments in the local environment",
	Subcommands: []*cli.Command{
		{
			Name:     "ls-app",
			Category: "Query the active deployment",
			Aliases:  []string{"list-applications"},
			Usage:    "Lists running Docker Compose applications",
			Action:   lsAppAction,
		},
		{
			Name:     "ls-con",
			Category: "Query the acative deployment",
			Aliases:  []string{"list-containers"},
			Usage:    "Lists the containers associated with a package deployment",
			Action:   lsConAction,
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
