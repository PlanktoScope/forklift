// Package env provides subcommands for the development environment
package env

import (
	"github.com/urfave/cli/v2"
)

var Cmd = &cli.Command{
	Name:    "env",
	Aliases: []string{"environment"},
	Usage:   "Facilitates development and maintenance of a Forklift environment",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:    "plt",
			Aliases: []string{"pallets"},
			Usage: "Replaces version-locked pallets from the cache with the corresponding pallets in " +
				"the specified directory paths",
		},
	},
	Subcommands: []*cli.Command{
		{
			Name:     "cache-plt",
			Aliases:  []string{"cache-pallets"},
			Category: "Use the environment",
			Usage:    "Updates the cache with the pallets available in the development environment",
			Action:   cachePalletAction,
		},
		{
			Name:     "cache-img",
			Aliases:  []string{"cache-images"},
			Category: "Use the environment",
			Usage:    "Pre-downloads the Docker container images required by the development environment",
			Action:   cacheImgAction,
		},
		{
			Name:     "check",
			Category: "Use the environment",
			Usage:    "Checks whether the development environment's resource constraints are satisfied",
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
				"development environment",
			Action: applyAction,
		},
		{
			Name:     "show",
			Category: "Query the environment",
			Usage:    "Describes the development environment",
			Action:   showAction,
		},
		{
			Name:     "ls-plt",
			Aliases:  []string{"list-pallets"},
			Category: "Query the environment",
			Usage:    "Lists pallets specified by the environment",
			Action:   lsPalletAction,
		},
		{
			Name:      "show-plt",
			Aliases:   []string{"show-pallet"},
			Category:  "Query the environment",
			Usage:     "Describes a pallet available in the development environment",
			ArgsUsage: "pallet_path",
			Action:    showPalletAction,
		},
		{
			Name:     "ls-pkg",
			Aliases:  []string{"list-packages"},
			Category: "Query the environment",
			Usage:    "Lists packages available in the development environment",
			Action:   lsPkgAction,
		},
		{
			Name:      "show-pkg",
			Aliases:   []string{"show-package"},
			Category:  "Query the environment",
			Usage:     "Describes a package available in the development environment",
			ArgsUsage: "package_path",
			Action:    showPkgAction,
		},
		{
			Name:    "ls-depl",
			Aliases: []string{"list-deployments"},
			Usage:   "Lists package deployments specified by the development environment",
			Action:  lsDeplAction,
		},
		{
			Name:      "show-depl",
			Aliases:   []string{"show-deployment"},
			Category:  "Query the environment",
			Usage:     "Describes a package deployment specified by the development environment",
			ArgsUsage: "package_path",
			Action:    showDeplAction,
		},
		{
			Name:      "add-plt",
			Aliases:   []string{"add-pallets"},
			Category:  "Query the environment",
			Usage:     "Adds pallets to the environment, tracking specified versions or branches",
			ArgsUsage: "[pallet_path@version_query]...",
			Action:    addPalletAction,
		},
		// TODO: add an upgrade-pallet action?
		// {
		// 	Name:      "rm-plt",
		// 	Aliases:   []string{"remove-pallets},
		// 	Category:  "Modify the environment",
		// 	Usage:     "Removes a pallet from the environment",
		// 	ArgsUsage: "pallet_path",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("removing pallet", c.Args().First())
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
