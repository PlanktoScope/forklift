// Package plt provides subcommands for the local pallet
package plt

import (
	"github.com/urfave/cli/v2"
)

var Cmd = &cli.Command{
	Name:    "plt",
	Aliases: []string{"pallet"},
	Usage:   "Manages the local pallet",
	Subcommands: []*cli.Command{
		{
			Name:      "clone",
			Category:  "Modify the pallet",
			Usage:     "Initializes the local pallet from a remote release",
			ArgsUsage: "[github_repository_path@release]",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "force",
					Usage: "Deletes the local pallet if it already exists",
				},
			},
			Action: cloneAction,
		},
		{
			Name:     "fetch",
			Category: "Modify the pallet",
			Usage:    "Updates information about the remote release",
			Action:   fetchAction,
		},
		{
			Name:     "pull",
			Category: "Modify the pallet",
			Usage:    "Fast-forwards the local pallet to match the remote release",
			Action:   pullAction,
		},
		// {
		// 	Name:  "push",
		// 	Category:  "Modify the pallet",
		// 	Usage: "Updates the remote release from the local pallet",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("pushing to remote origin")
		// 		return nil
		// 	},
		// },
		{
			Name:     "rm",
			Aliases:  []string{"remove"},
			Category: "Modify the pallet",
			Usage:    "Removes the local pallet",
			Action:   rmAction,
		},
		// remoteCmd,
		{
			Name:     "cache-repo",
			Aliases:  []string{"cache-repositories"},
			Category: "Use the pallet",
			Usage:    "Updates the cache with the repos available in the local pallet",
			Action:   cacheRepoAction,
		},
		{
			Name:     "cache-img",
			Aliases:  []string{"cache-images"},
			Category: "Use the pallet",
			Usage:    "Pre-downloads the Docker container images required by the local pallet",
			Action:   cacheImgAction,
		},
		{
			Name:     "check",
			Category: "Use the pallet",
			Usage:    "Checks whether the local pallet's resource constraints are satisfied",
			Action:   checkAction,
		},
		{
			Name:     "plan",
			Category: "Use the pallet",
			Usage: "Determines the changes needed to update the Docker host to match the deployments " +
				"specified by the local pallet",
			Action: planAction,
		},
		{
			Name:     "apply",
			Category: "Use the pallet",
			Usage:    "Updates the Docker host to match the deployments specified by the local pallet",
			Action:   applyAction,
		},
		{
			Name:     "show",
			Category: "Query the pallet",
			Usage:    "Describes the local pallet",
			Action:   showAction,
		},
		{
			Name:     "ls-repo",
			Aliases:  []string{"list-repositories"},
			Category: "Query the pallet",
			Usage:    "Lists repos available in the local pallet",
			Action:   lsRepoAction,
		},
		{
			Name:      "show-repo",
			Aliases:   []string{"show-repository"},
			Category:  "Query the pallet",
			Usage:     "Describes a repo available in the local pallet",
			ArgsUsage: "repo_path",
			Action:    showRepoAction,
		},
		{
			Name:     "ls-pkg",
			Aliases:  []string{"list-packages"},
			Category: "Query the pallet",
			Usage:    "Lists packages available in the local pallet",
			Action:   lsPkgAction,
		},
		{
			Name:      "show-pkg",
			Aliases:   []string{"show-package"},
			Category:  "Query the pallet",
			Usage:     "Describes a package available in the local pallet",
			ArgsUsage: "package_path",
			Action:    showPkgAction,
		},
		{
			Name:     "ls-depl",
			Aliases:  []string{"list-deployments"},
			Category: "Query the pallet",
			Usage:    "Lists package deployments specified by the local pallet",
			Action:   lsDeplAction,
		},
		{
			Name:      "show-depl",
			Aliases:   []string{"show-deployment"},
			Category:  "Query the pallet",
			Usage:     "Describes a package deployment specified by the local pallet",
			ArgsUsage: "package_path",
			Action:    showDeplAction,
		},
		// {
		// 	Name:      "add-repo",
		// 	Aliases:   []string{"add-repositories"},
		// 	Category:  "Modify the pallet",
		// 	Usage:     "Adds repos to the pallet, tracking specified versions or branches",
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
		// 	Category:  "Modify the pallet",
		// 	Usage:     "Removes a repo from the pallet",
		// 	ArgsUsage: "repo_path",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("removing repo", c.Args().First())
		// 		return nil
		// 	},
		// },
		// {
		// 	Name:  "add-depl",
		// 	Aliases:   []string{"add-deployments"},
		// 	Category:  "Modify the pallet",
		// 	Usage: "Adds a package deployment to the pallet",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("adding package deployment", c.Args().First())
		// 		return nil
		// 	},
		// },
		// {
		// 	Name:      "rm-depl",
		// 	Aliases:   []string{"remove-deployments"},
		// 	Category:  "Modify the pallet",
		// 	Usage:     "Removes a package deployment from the pallet",
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
// 	Usage: "Manages the local pallet's relationship to the remote source",
// 	Subcommands: []*cli.Command{
// 		{
// 			Name:  "set",
// 			Usage: "Sets the remote source for the local pallet",
// 			Action: func(c *cli.Context) error {
// 				fmt.Println("setting remote source to", c.Args().First())
// 				return nil
// 			},
// 		},
// 	},
// }
