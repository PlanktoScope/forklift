// Package plt provides subcommands for the development pallet
package plt

import (
	"github.com/urfave/cli/v2"
)

func MakeCmd(toolVersion, minVersion string) *cli.Command {
	var subcommands []*cli.Command
	for _, group := range makeSubcommandGroups(toolVersion, minVersion) {
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

func makeSubcommandGroups(toolVersion, minVersion string) [][]*cli.Command {
	return [][]*cli.Command{
		makeUseSubcmds(toolVersion, minVersion),
		makeQuerySubcmds(),
		makeModifySubcmds(toolVersion, minVersion),
	}
}

func makeUseSubcmds(toolVersion, minVersion string) []*cli.Command {
	const category = "Use the pallet"
	return []*cli.Command{
		{
			Name:     "cache-repo",
			Aliases:  []string{"cache-repositories"},
			Category: category,
			Usage:    "Updates the cache with the repos available in the development pallet",
			Action:   cacheRepoAction(toolVersion, minVersion),
		},
		{
			Name:     "cache-img",
			Aliases:  []string{"cache-images"},
			Category: category,
			Usage:    "Pre-downloads the Docker container images required by the development pallet",
			Action:   cacheImgAction(toolVersion, minVersion),
		},
		{
			Name:     "check",
			Category: category,
			Usage:    "Checks whether the development pallet's resource constraints are satisfied",
			Action:   checkAction(toolVersion, minVersion),
		},
		{
			Name:     "plan",
			Category: category,
			Usage: "Determines the changes needed to update the Docker host to match the deployments " +
				"specified by the local pallet",
			Action: planAction(toolVersion, minVersion),
		},
		{
			Name:     "apply",
			Category: category,
			Usage:    "Updates the Docker host to match the deployments specified by the development pallet",
			Action:   applyAction(toolVersion, minVersion),
		},
	}
}

func makeQuerySubcmds() []*cli.Command {
	const category = "Query the pallet"
	return []*cli.Command{
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
			ArgsUsage: "package_path",
			Action:    showDeplAction,
		},
	}
}

func makeModifySubcmds(toolVersion, minVersion string) []*cli.Command {
	const category = "Modify the pallet"
	return []*cli.Command{
		{
			Name:      "add-repo",
			Aliases:   []string{"add-repositories"},
			Category:  category,
			Usage:     "Adds repos to the pallet, tracking specified versions or branches",
			ArgsUsage: "[repo_path@version_query]...",
			Action:    addRepoAction(toolVersion, minVersion),
		},
		// TODO: add an rm-repo action
		// TODO: add an add-depl action
		// TODO: add an rm-depl action
	}
}
