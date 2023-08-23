// Package plt provides subcommands for the development pallet
package plt

import (
	"github.com/urfave/cli/v2"
)

var Cmd = &cli.Command{
	Name:    "plt",
	Aliases: []string{"pallet"},
	Usage: "Facilitates development and maintenance of a Forklift pallet in the current working " +
		"directory",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name: "repos",
			Usage: "Replaces version-locked repos from the cache with the corresponding repos in " +
				"the specified directory paths",
		},
	},
	Subcommands: []*cli.Command{
		{
			Name:     "cache-repo",
			Aliases:  []string{"cache-repositories"},
			Category: "Use the pallet",
			Usage:    "Updates the cache with the repos available in the development pallet",
			Action:   cacheRepoAction,
		},
		{
			Name:     "cache-img",
			Aliases:  []string{"cache-images"},
			Category: "Use the pallet",
			Usage:    "Pre-downloads the Docker container images required by the development pallet",
			Action:   cacheImgAction,
		},
		{
			Name:     "check",
			Category: "Use the pallet",
			Usage:    "Checks whether the development pallet's resource constraints are satisfied",
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
			Usage:    "Updates the Docker host to match the deployments specified by the development pallet",
			Action:   applyAction,
		},
		{
			Name:     "show",
			Category: "Query the pallet",
			Usage:    "Describes the development pallet",
			Action:   showAction,
		},
		{
			Name:     "ls-repo",
			Aliases:  []string{"list-repositories"},
			Category: "Query the pallet",
			Usage:    "Lists repos specified by the pallet",
			Action:   lsRepoAction,
		},
		{
			Name:      "show-repo",
			Aliases:   []string{"show-repository"},
			Category:  "Query the pallet",
			Usage:     "Describes a repo available in the development pallet",
			ArgsUsage: "repo_path",
			Action:    showRepoAction,
		},
		{
			Name:     "ls-pkg",
			Aliases:  []string{"list-packages"},
			Category: "Query the pallet",
			Usage:    "Lists packages available in the development pallet",
			Action:   lsPkgAction,
		},
		{
			Name:      "show-pkg",
			Aliases:   []string{"show-package"},
			Category:  "Query the pallet",
			Usage:     "Describes a package available in the development pallet",
			ArgsUsage: "package_path",
			Action:    showPkgAction,
		},
		{
			Name:     "ls-depl",
			Aliases:  []string{"list-deployments"},
			Category: "Query the pallet",
			Usage:    "Lists package deployments specified by the development pallet",
			Action:   lsDeplAction,
		},
		{
			Name:      "show-depl",
			Aliases:   []string{"show-deployment"},
			Category:  "Query the pallet",
			Usage:     "Describes a package deployment specified by the development pallet",
			ArgsUsage: "package_path",
			Action:    showDeplAction,
		},
		{
			Name:      "add-repo",
			Aliases:   []string{"add-repositories"},
			Category:  "Modify the pallet",
			Usage:     "Adds repos to the pallet, tracking specified versions or branches",
			ArgsUsage: "[repo_path@version_query]...",
			Action:    addRepoAction,
		},
		// TODO: add an upgrade-repo action?
		// {
		// 	Name:      "rm-repo",
		// 	Aliases:   []string{"remove-repositories},
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
