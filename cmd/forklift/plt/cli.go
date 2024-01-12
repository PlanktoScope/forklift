// Package plt provides subcommands for the local pallet
package plt

import (
	"github.com/urfave/cli/v2"
)

func MakeCmd(toolVersion, repoMinVersion, palletMinVersion string) *cli.Command {
	var subcommands []*cli.Command
	for _, group := range makeSubcommandGroups(toolVersion, repoMinVersion, palletMinVersion) {
		subcommands = append(subcommands, group...)
	}
	return &cli.Command{
		Name:        "plt",
		Aliases:     []string{"pallet"},
		Usage:       "Manages the local pallet",
		Subcommands: subcommands,
	}
}

func makeSubcommandGroups(toolVersion, repoMinVersion, palletMinVersion string) [][]*cli.Command {
	return [][]*cli.Command{
		makeUseSubcmds(toolVersion, repoMinVersion, palletMinVersion),
		makeQuerySubcmds(),
		makeModifySubcmds(),
	}
}

func makeUseSubcmds(toolVersion, repoMinVersion, palletMinVersion string) []*cli.Command {
	const category = "Use the pallet"
	return []*cli.Command{
		{
			Name:     "cache-repo",
			Aliases:  []string{"cache-repositories"},
			Category: category,
			Usage:    "Updates the cache with the repos available in the local pallet",
			Action:   cacheRepoAction(toolVersion, repoMinVersion, palletMinVersion),
		},
		{
			Name:     "cache-img",
			Aliases:  []string{"cache-images"},
			Category: category,
			Usage:    "Pre-downloads the Docker container images required by the local pallet",
			Action:   cacheImgAction(toolVersion, repoMinVersion, palletMinVersion),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "include-disabled",
					Usage: "also download images for disabled package deployments",
				},
				&cli.BoolFlag{
					Name:  "parallel",
					Usage: "parallelize downloading of images",
				},
			},
		},
		{
			Name:     "check",
			Category: category,
			Usage:    "Checks whether the local pallet's resource constraints are satisfied",
			Action:   checkAction(toolVersion, repoMinVersion, palletMinVersion),
		},
		{
			Name:     "plan",
			Category: category,
			Usage: "Determines the changes needed to update the Docker host to match the deployments " +
				"specified by the local pallet",
			Action: planAction(toolVersion, repoMinVersion, palletMinVersion),
		},
		{
			Name:     "apply",
			Category: category,
			Usage:    "Updates the Docker host to match the deployments specified by the local pallet",
			Action:   applyAction(toolVersion, repoMinVersion, palletMinVersion),
		},
	}
}

func makeQuerySubcmds() []*cli.Command {
	const category = "Query the pallet"
	return []*cli.Command{
		{
			Name:     "show",
			Category: category,
			Usage:    "Describes the local pallet",
			Action:   showAction,
		},
		{
			Name:     "ls-repo",
			Aliases:  []string{"list-repositories"},
			Category: category,
			Usage:    "Lists repos available in the local pallet",
			Action:   lsRepoAction,
		},
		{
			Name:      "show-repo",
			Aliases:   []string{"show-repository"},
			Category:  category,
			Usage:     "Describes a repo available in the local pallet",
			ArgsUsage: "repo_path",
			Action:    showRepoAction,
		},
		{
			Name:     "ls-pkg",
			Aliases:  []string{"list-packages"},
			Category: category,
			Usage:    "Lists packages available in the local pallet",
			Action:   lsPkgAction,
		},
		{
			Name:      "show-pkg",
			Aliases:   []string{"show-package"},
			Category:  category,
			Usage:     "Describes a package available in the local pallet",
			ArgsUsage: "package_path",
			Action:    showPkgAction,
		},
		{
			Name:     "ls-depl",
			Aliases:  []string{"list-deployments"},
			Category: category,
			Usage:    "Lists package deployments specified by the local pallet",
			Action:   lsDeplAction,
		},
		{
			Name:      "show-depl",
			Aliases:   []string{"show-deployment"},
			Category:  category,
			Usage:     "Describes a package deployment specified by the local pallet",
			ArgsUsage: "package_path",
			Action:    showDeplAction,
		},
	}
}

func makeModifySubcmds() []*cli.Command {
	const category = "Modify the pallet"
	return []*cli.Command{
		{
			Name:      "clone",
			Category:  category,
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
			Category: category,
			Usage:    "Updates information about the remote release",
			Action:   fetchAction,
		},
		{
			Name:     "pull",
			Category: category,
			Usage:    "Fast-forwards the local pallet to match the remote release",
			Action:   pullAction,
		},
		// {
		// 	Name:  "push",
		// 	Category:  category,
		// 	Usage: "Updates the remote release from the local pallet",
		// 	Action: func(c *cli.Context) error {
		// 		fmt.Println("pushing to remote origin")
		// 		return nil
		// 	},
		// },
		{
			Name:     "rm",
			Aliases:  []string{"remove"},
			Category: category,
			Usage:    "Removes the local pallet",
			Action:   rmAction,
		},
		// remoteCmd,
		// TODO: add an add-repo action
		// TODO: add an rm-repo action
		// TODO: add an add-depl action
		// TODO: add an rm-depl action
	}
}

//	var remoteCmd = &cli.Command{
//		Name:  "remote",
//		Usage: "Manages the local pallet's relationship to the remote source",
//		Subcommands: []*cli.Command{
//			{
//				Name:  "set",
//				Usage: "Sets the remote source for the local pallet",
//				Action: func(c *cli.Context) error {
//					fmt.Println("setting remote source to", c.Args().First())
//					return nil
//				},
//			},
//		},
//	}
