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

// Env

var envCmd = &cli.Command{
	Name:    "env",
	Aliases: []string{"environment"},
	Usage:   "Manages the local environment",
	Subcommands: []*cli.Command{
		{
			Name:      "clone",
			Category:  "Modify the environment",
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
			Name:     "fetch",
			Category: "Modify the environment",
			Usage:    "Updates information about the remote release",
			Action:   envFetchAction,
		},
		{
			Name:     "pull",
			Category: "Modify the environment",
			Usage:    "Fast-forwards the local environment to match the remote release",
			Action:   envPullAction,
		},
		// {
		// 	Name:  "push",
		// 	Category:  "Modify the environment",
		// 	Usage: "Updates the remote release from the local environment",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("pushing to remote origin")
		// 		return nil
		// 	},
		// },
		{
			Name:     "rm",
			Aliases:  []string{"remove"},
			Category: "Modify the environment",
			Usage:    "Removes the local environment",
			Action:   envRmAction,
		},
		// envRemoteCmd,
		{
			Name:     "cache-repo",
			Aliases:  []string{"c-r", "cache-repositories"},
			Category: "Use the environment",
			Usage:    "Updates the cache with the repositories available in the local environment",
			Action:   envCacheRepoAction,
		},
		{
			Name:     "cache-img",
			Aliases:  []string{"c-i", "cache-images"},
			Category: "Use the environment",
			Usage:    "Pre-downloads the Docker container images required by the local environment",
			Action:   envCacheImgAction,
		},
		{
			Name:     "check",
			Aliases:  []string{"c"},
			Category: "Use the environment",
			Usage:    "Checks whether the local environment's resource constraints are satisfied",
			Action:   envCheckAction,
		},
		{
			Name:     "apply",
			Aliases:  []string{"a"},
			Category: "Use the environment",
			Usage: "Updates the Docker Swarm to match the deployments specified by the " +
				"local environment",
			Action: envApplyAction,
		},
		{
			Name:     "show",
			Aliases:  []string{"s"},
			Category: "Query the environment",
			Usage:    "Describes the local environment",
			Action:   envShowAction,
		},
		{
			Name:     "ls-repo",
			Aliases:  []string{"ls-r", "list-repositories"},
			Category: "Query the environment",
			Usage:    "Lists repositories available in the local environment",
			Action:   envLsRepoAction,
		},
		{
			Name:      "show-repo",
			Aliases:   []string{"s-r", "show-repository"},
			Category:  "Query the environment",
			Usage:     "Describes a repository available in the local environment",
			ArgsUsage: "repository_path",
			Action:    envShowRepoAction,
		},
		{
			Name:     "ls-pkg",
			Aliases:  []string{"ls-p", "list-packages"},
			Category: "Query the environment",
			Usage:    "Lists packages available in the local environment",
			Action:   envLsPkgAction,
		},
		{
			Name:      "show-pkg",
			Aliases:   []string{"s-p", "show-package"},
			Category:  "Query the environment",
			Usage:     "Describes a package available in the local environment",
			ArgsUsage: "package_path",
			Action:    envShowPkgAction,
		},
		{
			Name:     "ls-depl",
			Aliases:  []string{"ls-d", "list-deployments"},
			Category: "Query the environment",
			Usage:    "Lists package deployments specified by the local environment",
			Action:   envLsDeplAction,
		},
		{
			Name:      "show-depl",
			Aliases:   []string{"s-d", "show-deployment"},
			Category:  "Query the environment",
			Usage:     "Describes a package deployment specified by the local environment",
			ArgsUsage: "package_path",
			Action:    envShowDeplAction,
		},
		// {
		// 	Name:      "add-repo",
		// 	Aliases:   []string{"add-r", "add-repositories"},
		// 	Category:  "Modify the environment",
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
		// 	Aliases:   []string{"rm-r", "remove-repositories"},
		// 	Category:  "Modify the environment",
		// 	Usage:     "Removes a repository from the environment",
		// 	ArgsUsage: "repository_path",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("removing repository", c.Args().First())
		// 		return nil
		// 	},
		// },
		// {
		// 	Name:  "add-depl",
		// 	Aliases:   []string{"add-d", "add-deployments"},
		// 	Category:  "Modify the environment",
		// 	Usage: "Adds a package deployment to the environment",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("adding package deployment", c.Args().First())
		// 		return nil
		// 	},
		// },
		// {
		// 	Name:      "rm-depl",
		// 	Aliases:   []string{"rm-d", "remove-deployments"},
		// 	Category:  "Modify the environment",
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
			Name:     "ls-repo",
			Aliases:  []string{"ls-r", "list-repositories"},
			Category: "Query the cache",
			Usage:    "Lists cached repositories",
			Action:   cacheLsRepoAction,
		},
		{
			Name:      "show-repo",
			Aliases:   []string{"s-r", "show-repository"},
			Category:  "Query the cache",
			Usage:     "Describes a cached repository",
			ArgsUsage: "repository_path@version",
			Action:    cacheShowRepoAction,
		},
		{
			Name:     "ls-pkg",
			Aliases:  []string{"ls-p", "list-packages"},
			Category: "Query the cache",
			Usage:    "Lists packages offered by cached repositories",
			Action:   cacheLsPkgAction,
		},
		{
			Name:      "show-pkg",
			Aliases:   []string{"s-p", "show-package"},
			Category:  "Query the cache",
			Usage:     "Describes a cached package",
			ArgsUsage: "package_path@version",
			Action:    cacheShowPkgAction,
		},
		{
			Name:     "ls-img",
			Aliases:  []string{"ls-i", "list-images"},
			Category: "Query the cache",
			Usage:    "Lists Docker container images in the local cache",
			Action:   cacheLsImgAction,
		},
		{
			Name:      "show-img",
			Aliases:   []string{"s-i", "show-image"},
			Category:  "Query the cache",
			Usage:     "Describes a cached Docker container image",
			ArgsUsage: "image_sha",
			Action:    cacheShowImgAction,
		},
		{
			Name:     "rm",
			Aliases:  []string{"remove"},
			Category: "Modify the cache",
			Usage:    "Removes the locally-cached repositories and Docker container images",
			Action:   cacheRmAction,
		},
	},
}

// Depl

var deplCmd = &cli.Command{
	Name:    "depl",
	Aliases: []string{"d", "deployments"},
	Usage:   "Manages active Pallet package deployments in the local environment",
	Subcommands: []*cli.Command{
		{
			Name:     "ls-stack",
			Category: "Query the active deployment",
			Aliases:  []string{"ls-s", "list-stacks"},
			Usage:    "Lists running Docker stacks",
			Action:   deplLsStackAction,
		},
		{
			Name:     "rm",
			Category: "Modify the active deployment",
			Aliases:  []string{"remove"},
			Usage:    "Removes all Docker stacks",
			Action:   deplRmAction,
		},
	},
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

var devEnvCmd = &cli.Command{
	Name:    "env",
	Aliases: []string{"environment"},
	Usage:   "Facilitates development and maintenance of a Forklift environment",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:    "repo",
			Aliases: []string{"r"},
			Usage: "Replaces version-locked repos from the cache with the corresponding repos in " +
				"the specified directory paths",
		},
	},
	Subcommands: []*cli.Command{
		{
			Name:     "cache-repo",
			Aliases:  []string{"c-r", "cache-repositories"},
			Category: "Use the environment",
			Usage:    "Updates the cache with the repositories available in the development environment",
			Action:   devEnvCacheRepoAction,
		},
		{
			Name:     "cache-img",
			Aliases:  []string{"c-i", "cache-images"},
			Category: "Use the environment",
			Usage:    "Pre-downloads the Docker container images required by the development environment",
			Action:   devEnvCacheImgAction,
		},
		{
			Name:     "check",
			Aliases:  []string{"c"},
			Category: "Use the environment",
			Usage:    "Checks whether the development environment's resource constraints are satisfied",
			Action:   devEnvCheckAction,
		},
		{
			Name:     "apply",
			Aliases:  []string{"a"},
			Category: "Use the environment",
			Usage: "Updates the Docker Swarm to match the deployments specified by the " +
				"development environment",
			Action: devEnvApplyAction,
		},
		{
			Name:     "show",
			Aliases:  []string{"s"},
			Category: "Query the environment",
			Usage:    "Describes the development environment",
			Action:   devEnvShowAction,
		},
		{
			Name:     "ls-repo",
			Aliases:  []string{"ls-r", "list-repositories"},
			Category: "Query the environment",
			Usage:    "Lists repositories specified by the environment",
			Action:   devEnvLsRepoAction,
		},
		{
			Name:      "show-repo",
			Aliases:   []string{"s-r", "show-repository"},
			Category:  "Query the environment",
			Usage:     "Describes a repository available in the development environment",
			ArgsUsage: "repository_path",
			Action:    devEnvShowRepoAction,
		},
		{
			Name:     "ls-pkg",
			Aliases:  []string{"ls-p", "list-packages"},
			Category: "Query the environment",
			Usage:    "Lists packages available in the development environment",
			Action:   devEnvLsPkgAction,
		},
		{
			Name:      "show-pkg",
			Aliases:   []string{"s-p", "show-package"},
			Category:  "Query the environment",
			Usage:     "Describes a package available in the development environment",
			ArgsUsage: "package_path",
			Action:    devEnvShowPkgAction,
		},
		{
			Name:    "ls-depl",
			Aliases: []string{"ls-d", "list-deployments"},
			Usage:   "Lists package deployments specified by the development environment",
			Action:  devEnvLsDeplAction,
		},
		{
			Name:      "show-depl",
			Aliases:   []string{"s-d", "show-deployment"},
			Category:  "Query the environment",
			Usage:     "Describes a package deployment specified by the development environment",
			ArgsUsage: "package_path",
			Action:    devEnvShowDeplAction,
		},
		{
			Name:      "add-repo",
			Aliases:   []string{"add-r", "add-repositories"},
			Category:  "Query the environment",
			Usage:     "Adds repositories to the environment, tracking specified versions or branches",
			ArgsUsage: "[pallet_repository_path@version_query]...",
			Action:    devEnvAddRepoAction,
		},
		// TODO: add an upgrade-repo action?
		// {
		// 	Name:      "rm-repo",
		// 	Aliases:   []string{"rm-r", "remove-repositories},
		// 	Category:  "Modify the environment",
		// 	Usage:     "Removes a repository from the environment",
		// 	ArgsUsage: "repository_path",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("removing repository", c.Args().First())
		// 		return nil
		// 	},
		// },
		// {
		// 	Name:  "add-depl",
		// 	Aliases:   []string{"add-d, "add-deployments"},
		// 	Category:  "Modify the environment",
		// 	Usage: "Adds a package deployment to the environment",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("adding package deployment", c.Args().First())
		// 		return nil
		// 	},
		// },
		// {
		// 	Name:      "rm-depl",
		// 	Aliases:   []string{"rm-d", "remove-deployments"},
		// 	Category:  "Modify the environment",
		// 	Usage:     "Removes a package deployment from the environment",
		// 	ArgsUsage: "package_path",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("removing package deployment", c.Args().First())
		// 		return nil
		// 	},
		// },
	},
}
