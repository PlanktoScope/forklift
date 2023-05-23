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
	Version: "v0.1.2",
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
		// envRemoteCmd,
		{
			Name:   "info",
			Usage:  "Describes the local environment",
			Action: envInfoAction,
		},
		{
			Name:   "cache",
			Usage:  "Updates the cache with the repositories available in the local environment",
			Action: envCacheAction,
		},
		{
			Name:   "deploy",
			Usage:  "Updates the Docker Swarm to match the deployments specified by the local environment",
			Action: envDeployAction,
		},
		// TODO: add a "info" command which prints a description of the local environment and indicates
		// whether it has diverged
		{
			Name:    "ls-repo",
			Aliases: []string{"ls-r", "list-repo"},
			Usage:   "Lists repositories available in the local environment",
			Action:  envLsRepoAction,
		},
		{
			Name:      "info-repo",
			Aliases:   []string{"info-r"},
			Usage:     "Describes a repository available in the local environment",
			ArgsUsage: "repository_path",
			Action:    envInfoRepoAction,
		},
		{
			Name:    "ls-pkg",
			Aliases: []string{"ls-p", "list-package"},
			Usage:   "Lists packages available in the local environment",
			Action:  envLsPkgAction,
		},
		{
			Name:      "info-pkg",
			Aliases:   []string{"info-p", "info-package"},
			Usage:     "Describes a package available in the local environment",
			ArgsUsage: "package_path",
			Action:    envInfoPkgAction,
		},
		{
			Name:    "ls-depl",
			Aliases: []string{"ls-d", "list-deploy"},
			Usage:   "Lists package deployments specified by the local environment",
			Action:  envLsDeplAction,
		},
		{
			Name:      "info-depl",
			Aliases:   []string{"info-d", "info-deploy"},
			Usage:     "Describes a package deployment specified by the local environment",
			ArgsUsage: "package_path",
			Action:    envInfoDeplAction,
		},
		// {
		// 	Name:      "add-repo",
		// 	Aliases:   []string{"add-r"},
		// 	Usage:     "Adds repositories to the environment, tracking specified versions or branches",
		// 	ArgsUsage: "[pallet_repository_path@version_query]...",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("adding repositories", c.Args())
		// 		// TODO: implement version queries - see https://go.dev/ref/mod#vcs-branch
		// 		return nil
		// 	},
		// },
		// TODO: add an upgrade-repo action
		// {
		// 	Name:      "rm-repo",
		// 	Aliases:   []string{"rm-r", "remove-repo"},
		// 	Usage:     "Removes a repository from the environment",
		// 	ArgsUsage: "repository_path",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("removing repository", c.Args().First())
		// 		return nil
		// 	},
		// },
		// {
		// 	Name:  "add-pkg",
		// 	Aliases:   []string{"add-p"},
		// 	Usage: "Adds a package deployment to the environment",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("adding package deployment", c.Args().First())
		// 		return nil
		// 	},
		// },
		// {
		// 	Name:      "rm-pkg",
		// 	Aliases:   []string{"rm-p", "remove-package"},
		// 	Usage:     "Removes a package deployment from the environment",
		// 	ArgsUsage: "package_path",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("removing package deployment", c.Args().First())
		// 		return nil
		// 	},
		// },
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
			Name:      "info-repo",
			Aliases:   []string{"info-r"},
			Usage:     "Describes a cached repository",
			ArgsUsage: "repository_path@version",
			Action:    cacheInfoRepoAction,
		},
		{
			Name:    "ls-pkg",
			Aliases: []string{"ls-p", "list-package"},
			Usage:   "Lists packages offered by cached repositories",
			Action:  cacheLsPkgAction,
		},
		{
			Name:      "info-pkg",
			Aliases:   []string{"info-p", "info-package"},
			Usage:     "Describes a cached package",
			ArgsUsage: "package_path@version",
			Action:    cacheInfoPkgAction,
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
			Name:    "ls-stack",
			Aliases: []string{"ls-s", "list-stack"},
			Usage:   "Lists running Docker stacks",
			Action:  deplLsStackAction,
		},
		{
			Name:    "rm",
			Aliases: []string{"remove"},
			Usage:   "Removes all Docker stacks",
			Action:  deplRmAction,
		},
	},
}
