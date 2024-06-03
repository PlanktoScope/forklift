// Package host provides subcommands for the local Docker host
package host

import (
	"github.com/urfave/cli/v2"
)

var Cmd = &cli.Command{
	Name:  "host",
	Usage: "Manages the local Docker host",
	Subcommands: []*cli.Command{
		{
			Name:     "ls-app",
			Aliases:  []string{"list-applications"},
			Category: "Query the Docker host",
			Usage:    "Lists running Docker Compose applications",
			Action:   lsAppAction,
		},
		{
			Name:     "ls-con",
			Aliases:  []string{"list-containers"},
			Category: "Query the Docker host",
			Usage:    "Lists the containers associated with a package deployment",
			Action:   lsConAction,
		},
		{
			Name:     "rm",
			Aliases:  []string{"remove", "del", "delete"},
			Category: "Modify the Docker host",
			Usage:    "Removes all Docker Compose applications",
			Action:   rmAction,
		},
	},
}
