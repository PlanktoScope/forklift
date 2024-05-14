// Package plt provides subcommands for the local pallet
package plt

import (
	"slices"

	"github.com/urfave/cli/v2"
)

type Versions struct {
	Tool               string
	MinSupportedRepo   string
	MinSupportedPallet string
	MinSupportedBundle string
	NewBundle          string
	NewStageStore      string
}

func MakeCmd(versions Versions) *cli.Command {
	return &cli.Command{
		Name:    "plt",
		Aliases: []string{"pallet"},
		Usage:   "Manages the local pallet",
		Subcommands: slices.Concat(
			[]*cli.Command{
				{
					Name:      "switch",
					Usage:     "(Re)initializes the local pallet, updates the cache, and stages the pallet",
					ArgsUsage: "[github_repository_path@release]",
					Action:    switchAction(versions),
					Flags: []cli.Flag{
						&cli.BoolFlag{
							Name:  "no-cache-img",
							Usage: "Don't download container images (this flag is ignored if --apply is set)",
						},
						&cli.BoolFlag{
							Name:  "apply",
							Usage: "Immediately apply the pallet after staging it",
						},
					},
				},
			},
			makeUseSubcmds(versions),
			makeQuerySubcmds(),
			makeModifySubcmds(versions),
		),
	}
}

func makeUseSubcmds(versions Versions) []*cli.Command {
	const category = "Use the pallet"
	return append(
		makeUseCacheSubcmds(versions),
		&cli.Command{
			Name:     "check",
			Category: category,
			Usage:    "Checks whether the local pallet's resource constraints are satisfied",
			Action:   checkAction(versions),
		},
		&cli.Command{
			Name:     "plan",
			Category: category,
			Usage: "Determines the changes needed to update the host to match the deployments " +
				"specified by the local pallet",
			Action: planAction(versions),
		},
		&cli.Command{
			Name:     "stage",
			Category: category,
			Usage:    "Builds and stages a bundle of the local pallet to be applied later",
			Action:   stageAction(versions),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "no-cache-img",
					Usage: "Don't download container images",
				},
			},
		},
		&cli.Command{
			Name:     "apply",
			Category: category,
			Usage: "Builds, stages, and immediately applies a bundle of the local pallet to update the " +
				"host to match the deployments specified by the local pallet",
			Action: applyAction(versions),
		},
	)
}

func makeUseCacheSubcmds(versions Versions) []*cli.Command {
	const category = "Use the pallet"
	return []*cli.Command{
		{
			Name:     "cache-all",
			Category: category,
			Usage:    "Updates the cache with everything needed by the local pallet",
			Action:   cacheAllAction(versions),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "include-disabled",
					Usage: "Also cache things needed for disabled package deployments",
				},
			},
		},
		{
			Name:     "cache-repo",
			Aliases:  []string{"cache-repositories"},
			Category: category,
			Usage:    "Updates the cache with the repos available in the local pallet",
			Action:   cacheRepoAction(versions),
		},
		{
			Name:     "cache-dl",
			Aliases:  []string{"cache-downloads"},
			Category: category,
			Usage:    "Pre-downloads files to be exported by the local pallet",
			Action:   cacheDlAction(versions),
		},
		{
			Name:     "cache-img",
			Aliases:  []string{"cache-images"},
			Category: category,
			Usage:    "Pre-downloads the Docker container images required by the local pallet",
			Action:   cacheImgAction(versions),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "include-disabled",
					Usage: "Also download images for disabled package deployments",
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
			Usage:    "Lists package deployments specified by the local pallet",
			Action:   lsDeplAction,
		},
		{
			Name:      "show-depl",
			Aliases:   []string{"show-deployment"},
			Category:  category,
			Usage:     "Describes a package deployment specified by the local pallet",
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
					Usage: "Locates the package even if the specified deployment is disabled",
				},
			},
		},
	}
}

func makeModifySubcmds(versions Versions) []*cli.Command {
	const category = "Modify the pallet"
	return slices.Concat(
		makeModifyGitSubcmds(versions),
		[]*cli.Command{
			{
				Name:     "rm",
				Aliases:  []string{"remove"},
				Category: category,
				Usage:    "Removes the local pallet",
				Action:   rmAction,
			},
		},
		makeModifyRepoSubcmds(versions),
		makeModifyDeplSubcmds(versions),
	)
}

func makeModifyGitSubcmds(versions Versions) []*cli.Command {
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
				&cli.BoolFlag{
					Name:  "no-cache-req",
					Usage: "Don't download repositories and pallets required by this pallet",
				},
			},
			Action: cloneAction(versions),
		},
		// TODO: add a "checkout" action
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
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name: "no-cache-req",
					Usage: "Don't download repositories and pallets required by this pallet after adding " +
						"the repo",
				},
			},
			Action: pullAction(versions),
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
		// remoteCmd,
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

func makeModifyRepoSubcmds(versions Versions) []*cli.Command {
	const category = "Modify the pallet"
	return []*cli.Command{
		{
			Name:     "add-repo",
			Aliases:  []string{"add-repositories", "require-repo", "require-repositories"},
			Category: category,
			Usage: "Adds (or re-adds) repo requirements to the pallet, tracking specified versions " +
				"or branches",
			ArgsUsage: "[repo_path@version_query]...",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name: "no-cache-req",
					Usage: "Don't download repositories and pallets required by this pallet after adding " +
						"the repo",
				},
			},
			Action: addRepoAction(versions),
		},
		{
			Name:      "rm-repo",
			Aliases:   []string{"remove-repositories", "drop-repo", "drop-repositories"},
			Category:  category,
			Usage:     "Removes repo requirements from the pallet",
			ArgsUsage: "repo_path...",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name: "force",
					Usage: "Remove specified repo requirements even if some declared package deployments " +
						"depend on them",
				},
			},
			Action: rmRepoAction(versions),
		},
	}
}

func makeModifyDeplSubcmds( //nolint:funlen // this is already decomposed; it's hard to split more
	versions Versions,
) []*cli.Command {
	const category = "Modify the pallet"
	baseFlags := []cli.Flag{
		&cli.BoolFlag{
			Name: "stage",
			Usage: "Immediately stage the pallet after making the modification (this flag is ignored " +
				"if --apply is set)",
		},
		&cli.BoolFlag{
			Name:  "no-cache-img",
			Usage: "Don't download container images (this flag is only used if --stage is set)",
		},
		&cli.BoolFlag{
			Name:  "apply",
			Usage: "Immediately apply the pallet after staging it",
		},
	}
	return []*cli.Command{
		{
			Name:      "add-depl",
			Aliases:   []string{"add-deployment"},
			Category:  category,
			Usage:     "Adds (or re-adds) a package deployment to the pallet",
			ArgsUsage: "deployment_name package_path...",
			Flags: slices.Concat(
				[]cli.Flag{
					&cli.StringSliceFlag{
						Name:  "feature",
						Usage: "Enable the specified feature flag in the package deployment",
					},
					&cli.BoolFlag{
						Name:  "disabled",
						Usage: "Add a disabled package deployment",
					},
					&cli.BoolFlag{
						Name: "force",
						Usage: "Add specified deployment even if package_path cannot be resolved or the " +
							"specified feature flags are not allowed for it",
					},
				},
				baseFlags,
			),
			Action: addDeplAction(versions),
		},
		{
			Name:      "rm-depl",
			Aliases:   []string{"remove-deployment", "remove-deployments"},
			Category:  category,
			Usage:     "Removes deployment from the pallet",
			ArgsUsage: "deployment_name...",
			Flags:     baseFlags,
			Action:    rmDeplAction(versions),
		},
		{
			Name:      "set-depl-pkg",
			Aliases:   []string{"set-deployment-package"},
			Category:  category,
			Usage:     "Sets the path of the package to deploy in the specified deployment",
			ArgsUsage: "deployment_name package_path...",
			Flags: slices.Concat(
				[]cli.Flag{
					&cli.BoolFlag{
						Name: "force",
						Usage: "Use the specified package path even if it cannot be resolved or makes the " +
							"enabled feature flags invalid",
					},
				},
				baseFlags,
			),
			Action: setDeplPkgAction(versions),
		},
		{
			Name:      "add-depl-feat",
			Aliases:   []string{"add-deployment-feature", "add-deployment-features"},
			Category:  category,
			Usage:     "Enables the specified package features in the specified deployment",
			ArgsUsage: "deployment_name feature_name...",
			Flags: slices.Concat(
				[]cli.Flag{
					&cli.BoolFlag{
						Name: "force",
						Usage: "Enable the specified feature flags even if they're not allowed by the  " +
							"deployment's package",
					},
				},
				baseFlags,
			),
			Action: addDeplFeatAction(versions),
		},
		{
			Name:      "rm-depl-feat",
			Aliases:   []string{"remove-deployment-feature", "remove-deployment-features"},
			Category:  category,
			Usage:     "Disables the specified package features in the specified deployment",
			ArgsUsage: "deployment_name feature_name...",
			Flags:     baseFlags,
			Action:    rmDeplFeatAction(versions),
		},
		{
			Name:      "set-depl-disabled",
			Aliases:   []string{"set-deployment-disabled", "disable-depl", "disable-deployment"},
			Category:  category,
			Usage:     "Disables the specified deployment",
			ArgsUsage: "deployment_name",
			Flags:     baseFlags,
			Action:    setDeplDisabledAction(versions, true),
		},
		{
			Name:      "unset-depl-disabled",
			Aliases:   []string{"unset-deployment-disabled", "enable-depl", "enable-deployment"},
			Category:  category,
			Usage:     "Enables the specified deployment",
			ArgsUsage: "deployment_name",
			Flags:     baseFlags,
			Action:    setDeplDisabledAction(versions, false),
		},
	}
}
