// Package env provides subcommands for the local environment
package env

import (
	"github.com/urfave/cli/v2"
)

var Cmd = &cli.Command{
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
					Name:  "force",
					Usage: "Deletes the local environment if it already exists",
				},
			},
			Action: cloneAction,
		},
		{
			Name:     "fetch",
			Category: "Modify the environment",
			Usage:    "Updates information about the remote release",
			Action:   fetchAction,
		},
		{
			Name:     "pull",
			Category: "Modify the environment",
			Usage:    "Fast-forwards the local environment to match the remote release",
			Action:   pullAction,
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
			Action:   rmAction,
		},
		// remoteCmd,
		{
			Name:     "cache-repo",
			Aliases:  []string{"cache-repositories"},
			Category: "Use the environment",
			Usage:    "Updates the cache with the repos available in the local environment",
			Action:   cacheRepoAction,
		},
		{
			Name:     "cache-img",
			Aliases:  []string{"cache-images"},
			Category: "Use the environment",
			Usage:    "Pre-downloads the Docker container images required by the local environment",
			Action:   cacheImgAction,
		},
		{
			Name:     "check",
			Category: "Use the environment",
			Usage:    "Checks whether the local environment's resource constraints are satisfied",
			Action:   checkAction,
		},
		{
			Name:     "plan",
			Category: "Use the environment",
			Usage: "Determines the changes needed to update the Docker Swarm to match the deployments " +
				"specified by the local environment",
			Action: planAction,
		},
		{
			Name:     "apply",
			Category: "Use the environment",
			Usage: "Updates the Docker Swarm to match the deployments specified by the " +
				"local environment",
			Action: applyAction,
		},
		{
			Name:     "show",
			Category: "Query the environment",
			Usage:    "Describes the local environment",
			Action:   showAction,
		},
		{
			Name:     "ls-repo",
			Aliases:  []string{"list-repositories"},
			Category: "Query the environment",
			Usage:    "Lists repos available in the local environment",
			Action:   lsRepoAction,
		},
		{
			Name:      "show-repo",
			Aliases:   []string{"show-repository"},
			Category:  "Query the environment",
			Usage:     "Describes a repo available in the local environment",
			ArgsUsage: "repo_path",
			Action:    showRepoAction,
		},
		{
			Name:     "ls-pkg",
			Aliases:  []string{"list-packages"},
			Category: "Query the environment",
			Usage:    "Lists packages available in the local environment",
			Action:   lsPkgAction,
		},
		{
			Name:      "show-pkg",
			Aliases:   []string{"show-package"},
			Category:  "Query the environment",
			Usage:     "Describes a package available in the local environment",
			ArgsUsage: "package_path",
			Action:    showPkgAction,
		},
		{
			Name:     "ls-depl",
			Aliases:  []string{"list-deployments"},
			Category: "Query the environment",
			Usage:    "Lists package deployments specified by the local environment",
			Action:   lsDeplAction,
		},
		{
			Name:      "show-depl",
			Aliases:   []string{"show-deployment"},
			Category:  "Query the environment",
			Usage:     "Describes a package deployment specified by the local environment",
			ArgsUsage: "package_path",
			Action:    showDeplAction,
		},
		// {
		// 	Name:      "add-repo",
		// 	Aliases:   []string{"add-repositories"},
		// 	Category:  "Modify the environment",
		// 	Usage:     "Adds repos to the environment, tracking specified versions or branches",
		// 	ArgsUsage: "[repo_path@version_query]...",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("adding repos", c.Args())
		// 		// TODO: implement version queries - see https://go.dev/ref/mod#vcs-branch
		// 		return nil
		// 	},
		// },
		// TODO: add an upgrade-repo action
		// {
		// 	Name:      "rm-repo",
		// 	Aliases:   []string{"remove-repositories"},
		// 	Category:  "Modify the environment",
		// 	Usage:     "Removes a repo from the environment",
		// 	ArgsUsage: "repo_path",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("removing repo", c.Args().First())
		// 		return nil
		// 	},
		// },
		// {
		// 	Name:  "add-depl",
		// 	Aliases:   []string{"add-deployments"},
		// 	Category:  "Modify the environment",
		// 	Usage: "Adds a package deployment to the environment",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("adding package deployment", c.Args().First())
		// 		return nil
		// 	},
		// },
		// {
		// 	Name:      "rm-depl",
		// 	Aliases:   []string{"remove-deployments"},
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

// var remoteCmd = &cli.Command{
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
