package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
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
	Version: "v0.1.7",
	Usage:   "Manages Pallet repositories and package deployments",
	Commands: []*cli.Command{
		envCmd,
		cacheCmd,
		deplCmd,
		devCmd,
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "workspace",
			Aliases: []string{"w"},
			Value:   filepath.Join(defaultWorkspaceBase, ".forklift"),
			Usage:   "Path of the forklift workspace",
			EnvVars: []string{"FORKLIFT_WORKSPACE"},
		},
	},
	Suggest: true,
}

// Dev

var defaultWorkingDir, _ = os.Getwd()

var devCmd = &cli.Command{
	Name:    "dev",
	Aliases: []string{"development"},
	Usage:   "Facilitates development and maintenance in the current working directory",
	Subcommands: []*cli.Command{
		devEnvCmd,
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "cwd",
			Value:   defaultWorkingDir,
			Usage:   "Path of the current working directory",
			EnvVars: []string{"FORKLIFT_CWD"},
		},
	},
}
