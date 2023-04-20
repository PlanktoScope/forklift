package main

import (
	"fmt"
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
	Name:    "forklift",
	Version: "v0.1.0",
	Usage:   "Manages Pallet repositories and package deployments",
	Commands: []*cli.Command{
		envCmd,
		cacheCmd,
		deplCmd,
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

// Env

var envCmd = &cli.Command{
	Name:    "env",
	Aliases: []string{"environment"},
	Usage:   "Manages the local environment",
	Subcommands: []*cli.Command{
		{
			Name:      "clone",
			Usage:     "Initializes the local environment from a remote release",
			ArgsUsage: "[github_repository_path@release]",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "force",
					Aliases: []string{"f"},
					Usage:   "Deletes the local environment if it already exists",
				},
			},
			Action: envCloneAction,
		},
		{
			Name:   "fetch",
			Usage:  "Updates information about the remote release",
			Action: envFetchAction,
		},
		{
			Name:   "pull",
			Usage:  "Fast-forwards the local environment to match the remote release",
			Action: envPullAction,
		},
		// {
		// 	Name:  "push",
		// 	Usage: "Updates the remote release from the local environment",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("pushing to remote origin")
		// 		return nil
		// 	},
		// },
		{
			Name:    "rm",
			Aliases: []string{"remove"},
			Usage:   "Removes the local environment",
			Action:  envRmAction,
		},
		// TODO: move these into a repos subcommand
		// {
		// 	Name:      "add",
		// 	Usage:     "Adds repositories to the environment, tracking specified versions or branches",
		// 	ArgsUsage: "[pallet_repository_path@version_query]...",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("adding repositories", c.Args())
		// 		return nil
		// 	},
		// },
		// {
		// 	Name:      "rm",
		// 	Aliases:   []string{"remove"},
		// 	Usage:     "Removes a repository from the environment",
		// 	ArgsUsage: "repository_path",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("removing repository", c.Args().First())
		// 		return nil
		// 	},
		// },
		// envRemoteCmd,
	},
}

// var envRemoteCmd = &cli.Command{
// 	Name:  "remote",
// 	Usage: "Manages the local environment's relationship to the remote source",
// 	Subcommands: []*cli.Command{
// 		{
// 			Name:  "set",
// 			Usage: "Sets the remote source for the local environment",
// 			Action: func(c *cli.Context) error {
// 				fmt.Println("setting remote source to", c.Args().First())
// 				return nil
// 			},
// 		},
// 	},
// }

// Cache

var cacheCmd = &cli.Command{
	Name:  "cache",
	Usage: "Manages the local cache of Pallet repositories and packages",
	Subcommands: []*cli.Command{
		{
			Name:    "ls-repo",
			Aliases: []string{"ls-r", "list-repo"},
			Usage:   "Lists cached repositories",
			Action:  cacheLsRepoAction,
		},
		{
			Name:    "ls-pkg",
			Aliases: []string{"ls-p", "list-package"},
			Usage:   "Lists packages offered by repositories in the local environment",
			Action: func(c *cli.Context) error {
				fmt.Println("packages:")
				return nil
			},
		},
		{
			Name:      "info-pkg",
			Aliases:   []string{"info-p", "info-package"},
			Usage:     "Describes a package",
			ArgsUsage: "package_path",
			Action: func(c *cli.Context) error {
				fmt.Println("package", c.Args().First())
				return nil
			},
		},
		{
			Name:    "update",
			Aliases: []string{"up"},
			Usage:   "Updates the cache with the repositories specified by the local environment",
			Action:  cacheUpdateAction,
		},
		{
			Name:    "rm",
			Aliases: []string{"remove"},
			Usage:   "Removes the local cache",
			Action:  cacheRmAction,
		},
	},
}

// Depl

var deplCmd = &cli.Command{
	Name:    "depl",
	Aliases: []string{"deployment"},
	Usage:   "Manages Pallet package deployments in the local environment",
	Subcommands: []*cli.Command{
		{
			Name:    "ls",
			Aliases: []string{"list"},
			Usage:   "Lists package deployments in the environment",
			Action: func(c *cli.Context) error {
				fmt.Println("package deployments:")
				return nil
			},
		},
		{
			Name:  "apply",
			Usage: "Updates the Docker Swarm to exactly match the local environment",
			Action: func(c *cli.Context) error {
				fmt.Println("applying local environment")
				return nil
			},
		},
		{
			Name:  "add",
			Usage: "Adds a package deployment to the environment",
			Action: func(c *cli.Context) error {
				fmt.Println("adding package deployment", c.Args().First())
				return nil
			},
		},
		{
			Name:      "rm",
			Aliases:   []string{"remove"},
			Usage:     "Removes a package deployment from the environment",
			ArgsUsage: "package_path",
			Action: func(c *cli.Context) error {
				fmt.Println("removing package deployment", c.Args().First())
				return nil
			},
		},
	},
}
