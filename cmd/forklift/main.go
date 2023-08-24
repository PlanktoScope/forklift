package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/cmd/forklift/cache"
	"github.com/PlanktoScope/forklift/cmd/forklift/dev"
	"github.com/PlanktoScope/forklift/cmd/forklift/host"
	"github.com/PlanktoScope/forklift/cmd/forklift/plt"
)

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

var defaultWorkspaceBase, _ = os.UserHomeDir()

var app = &cli.App{
	Name: "forklift",
	// TODO: see if there's a way to get the version from a build tag, so that we don't have to update
	// this manually
	Version: "v0.3.1",
	Usage:   "Manages pallets and package deployments",
	Commands: []*cli.Command{
		plt.Cmd,
		cache.Cmd,
		host.Cmd,
		dev.Cmd,
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "workspace",
			Aliases: []string{"ws"},
			Value:   filepath.Join(defaultWorkspaceBase, ".forklift"),
			Usage:   "Path of the forklift workspace",
			EnvVars: []string{"FORKLIFT_WORKSPACE"},
		},
	},
	Suggest: true,
}
