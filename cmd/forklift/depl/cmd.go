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
			Aliases:  []string{"list-applications"},
			Category: "Query the active deployment",
			Usage:    "Lists running Docker Compose applications",
			Action:   lsAppAction,
		},
		{
			Name:     "ls-con",
			Aliases:  []string{"list-containers"},
			Category: "Query the active deployment",
			Usage:    "Lists the containers associated with a package deployment",
			Action:   lsConAction,
		},
		{
			Name:     "rm",
			Aliases:  []string{"remove"},
			Category: "Modify the active deployment",
			Usage:    "Removes all Docker stacks",
			Action:   rmAction,
		},
	},
}
