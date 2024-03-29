// Package plt provides subcommands for the development pallet
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
		Subcommands: subcommands,
	}
}

func makeSubcommandGroups(toolVersion, repoMinVersion, palletMinVersion string) [][]*cli.Command {
	return [][]*cli.Command{
		makeUseSubcmds(toolVersion, repoMinVersion, palletMinVersion),
		makeQuerySubcmds(),
		makeModifySubcmds(toolVersion, repoMinVersion, palletMinVersion),
	}
}

func makeUseSubcmds(toolVersion, repoMinVersion, palletMinVersion string) []*cli.Command {
	const category = "Use the pallet"
	return append(
		makeUseCacheSubcmds(toolVersion, repoMinVersion, palletMinVersion),
		&cli.Command{
			Name:     "check",
			Category: category,
			Usage:    "Checks whether the development pallet's resource constraints are satisfied",
			Action:   checkAction(toolVersion, repoMinVersion, palletMinVersion),
		},
		&cli.Command{
			Name:     "plan",
			Category: category,
			Usage: "Determines the changes needed to update the Docker host to match the deployments " +
				"specified by the local pallet",
			Action: planAction(toolVersion, repoMinVersion, palletMinVersion),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "parallel",
					Usage: "parallelize downloading of images",
				},
			},
		},
		&cli.Command{
			Name:     "apply",
			Category: category,
			Usage:    "Updates the Docker host to match the deployments specified by the development pallet",
			Action:   applyAction(toolVersion, repoMinVersion, palletMinVersion),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "parallel",
					Usage: "parallelize downloading of images",
				},
			},
		},
	)
}

func makeUseCacheSubcmds(toolVersion, repoMinVersion, palletMinVersion string) []*cli.Command {
	const category = "Use the pallet"
	return []*cli.Command{
		{
			Name:     "cache-all",
			Category: category,
			Usage:    "Updates the cache with everything needed by the development pallet",
			Action:   cacheAllAction(toolVersion, repoMinVersion, palletMinVersion),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "include-disabled",
					Usage: "also cache things needed for disabled package deployments",
				},
				&cli.BoolFlag{
					Name:  "parallel",
					Usage: "parallelize downloading of images",
				},
			},
		},
		{
			Name:     "cache-repo",
			Aliases:  []string{"cache-repositories"},
			Category: category,
			Usage:    "Updates the cache with the repos available in the development pallet",
			Action:   cacheRepoAction(toolVersion, repoMinVersion, palletMinVersion),
		},
		{
			Name:     "cache-img",
			Aliases:  []string{"cache-images"},
			Category: category,
			Usage:    "Pre-downloads the Docker container images required by the development pallet",
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
	}
}

func makeQuerySubcmds() []*cli.Command {
	const category = "Query the pallet"
	return append(
		[]*cli.Command{
			{
				Name:     "show",
				Category: category,
				Usage:    "Describes the development pallet",
				Action:   showAction,
			},
			{
				Name:     "ls-repo",
				Aliases:  []string{"list-repositories"},
				Category: category,
				Usage:    "Lists repos specified by the pallet",
				Action:   lsRepoAction,
			},
			{
				Name:      "show-repo",
				Aliases:   []string{"show-repository"},
				Category:  category,
				Usage:     "Describes a repo available in the development pallet",
				ArgsUsage: "repo_path",
				Action:    showRepoAction,
			},
			{
				Name:     "ls-pkg",
				Aliases:  []string{"list-packages"},
				Category: category,
				Usage:    "Lists packages available in the development pallet",
				Action:   lsPkgAction,
			},
			{
				Name:      "show-pkg",
				Aliases:   []string{"show-package"},
				Category:  category,
				Usage:     "Describes a package available in the development pallet",
				ArgsUsage: "package_path",
				Action:    showPkgAction,
			},
		},
		makeQueryDeplSubcmds(category)...,
	)
}

func makeQueryDeplSubcmds(category string) []*cli.Command {
	return []*cli.Command{
		{
			Name:     "ls-depl",
			Aliases:  []string{"list-deployments"},
			Category: category,
			Usage:    "Lists package deployments specified by the development pallet",
			Action:   lsDeplAction,
		},
		{
			Name:      "show-depl",
			Aliases:   []string{"show-deployment"},
			Category:  category,
			Usage:     "Describes a package deployment specified by the development pallet",
			ArgsUsage: "deployment_name",
			Action:    showDeplAction,
		},
		{
			Name:      "locate-depl-pkg",
			Aliases:   []string{"locate-deployment-package"},
			Category:  category,
			Usage:     "Prints the absolute filesystem path of the package for the specified deployment",
			ArgsUsage: "deployment_name",
			Action:    locateDeplPkgAction,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "allow-disabled",
					Usage: "locates the package even if the specified deployment is disabled",
				},
			},
		},
	}
}

func makeModifySubcmds(toolVersion, repoMinVersion, palletMinVersion string) []*cli.Command {
	const category = "Modify the pallet"
	return []*cli.Command{
		{
			Name:      "add-repo",
			Aliases:   []string{"add-repositories"},
			Category:  category,
			Usage:     "Adds repos to the pallet, tracking specified versions or branches",
			ArgsUsage: "[repo_path@version_query]...",
			Action:    addRepoAction(toolVersion, repoMinVersion, palletMinVersion),
		},
		// TODO: add an rm-repo action
		// TODO: add an add-depl action
		// TODO: add an rm-depl action
	}
}
