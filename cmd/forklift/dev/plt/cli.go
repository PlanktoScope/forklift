// Package plt provides subcommands for the development pallet
package plt

import (
	"slices"

	"github.com/urfave/cli/v2"

	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

type Versions struct {
	Staging       fcli.StagingVersions
	NewStageStore string
}

func (v Versions) Core() fcli.Versions {
	return v.Staging.Core
}

func MakeCmd(versions Versions) *cli.Command {
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
		Subcommands: slices.Concat(
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
			Usage:    "Checks whether the development pallet's resource constraints are satisfied",
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
			Usage:    "Builds and stages a bundle of the development pallet to be applied later",
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
			Usage: "Builds, stages, and immediately applies a bundle of the development pallet to " +
				"update the host to match the deployments specified by the development pallet",
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
			Usage:    "Updates the cache with everything needed to apply the development pallet",
			Action:   cacheAllAction(versions),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "include-disabled",
					Usage: "Also cache things needed for disabled package deployments",
				},
			},
		},
		{
			Name:     "cache-plt",
			Aliases:  []string{"cache-pallets"},
			Category: category,
			Usage:    "Updates the cache with the pallets required by the development pallet",
			Action:   cachePltAction(versions),
		},
		{
			Name:     "cache-repo",
			Aliases:  []string{"cache-repositories"},
			Category: category,
			Usage:    "Updates the cache with the repos required by the development pallet",
			Action:   cacheRepoAction(versions),
		},
		{
			Name:     "cache-dl",
			Aliases:  []string{"cache-downloads"},
			Category: category,
			Usage:    "Pre-downloads files to be exported by the development pallet",
			Action:   cacheDlAction(versions),
		},
		{
			Name:     "cache-img",
			Aliases:  []string{"cache-images"},
			Category: category,
			Usage:    "Pre-downloads the Docker container images required by the development pallet",
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
	return slices.Concat(
		[]*cli.Command{
			{
				Name:     "show",
				Category: category,
				Usage:    "Describes the development pallet",
				Action:   showAction,
			},
		},
		makeQueryReqSubcmds(category),
		[]*cli.Command{
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
		makeQueryDeplSubcmds(category),
		[]*cli.Command{
			{
				Name:     "ls-img",
				Aliases:  []string{"list-images"},
				Category: category,
				Usage:    "Lists the Docker container images required by the development pallet",
				Action:   lsImgAction,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "include-disabled",
						Usage: "Also download images for disabled package deployments",
					},
				},
			},
		},
	)
}

func makeQueryReqSubcmds(category string) []*cli.Command {
	return []*cli.Command{
		{
			Name:     "ls-plt",
			Aliases:  []string{"list-pallets"},
			Category: category,
			Usage:    "Lists pallets which the development pallet may import files from",
			Action:   lsPltAction,
		},
		{
			Name:      "show-plt",
			Aliases:   []string{"show-pallet"},
			Category:  category,
			Usage:     "Describes a pallet which the development pallet may import files from",
			ArgsUsage: "plt_path",
			Action:    showPltAction,
		},
		{
			Name:     "ls-repo",
			Aliases:  []string{"list-repositories"},
			Category: category,
			Usage:    "Lists repos specified by the development pallet",
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
	}
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
					Usage: "Locates the package even if the specified deployment is disabled",
				},
			},
		},
	}
}

func makeModifySubcmds(versions Versions) []*cli.Command {
	return slices.Concat(
		makeModifyPltSubcmds(versions),
		makeModifyRepoSubcmds(versions),
		makeModifyDeplSubcmds(versions),
	)
}

func makeModifyPltSubcmds(versions Versions) []*cli.Command {
	const category = "Modify the pallet's pallet requirements"
	return []*cli.Command{
		{
			Name: "add-plt",
			Aliases: []string{
				"add-pallet", "add-pallets",
				"req-plt", "req-pallet", "req-pallets",
				"require-plt", "require-pallet", "require-pallets",
			},
			Category: category,
			Usage: "Adds (or re-adds) pallet requirements to the pallet, tracking specified versions " +
				"or branches",
			ArgsUsage: "[plt_path@version_query]...",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name: "no-cache-req",
					Usage: "Don't download repositories and pallets required by this pallet after adding " +
						"the pallet",
				},
			},
			Action: addPltAction(versions),
		},
		// TODO: add an upgrade-plt [plt_path]... command (upgrade all if no args)
		// TODO: add a check-upgrade-plt [plt_path]... command (check all upgrades if no args)
		// TODO: add a cache-upgrade-plt plt_path command (cache all upgrades if no args)
		// TODO: add a show-upgrade-plt-query plt_path[@] command
		// TODO: add a set-upgrade-plt-query plt_path@version_query command
		{
			Name: "rm-plt",
			Aliases: []string{
				"remove-pallet", "remove-pallets",
				"del-plt", "delete-pallet", "delete-pallets",
				"drop-plt", "drop-pallet", "drop-pallets",
			},
			Category:  category,
			Usage:     "Removes pallet requirements from the pallet",
			ArgsUsage: "plt_path...",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name: "force",
					Usage: "Remove specified pallet requirements even if some declared file imports " +
						"depend on them",
				},
			},
			Action: rmPltAction(versions),
		},
	}
}

func makeModifyRepoSubcmds(versions Versions) []*cli.Command {
	const category = "Modify the pallet's package repository requirements"
	return []*cli.Command{
		{
			Name: "add-repo",
			Aliases: []string{
				"add-repository", "add-repositories",
				"req-repo", "req-repository", "req-repositories",
				"require-repo", "require-repository", "require-repositories",
			},
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
			// TODO: add an upgrade-repo [repo_path]... command (upgrade all if no args)
			// TODO: add a check-upgrade-repo [repo_path]... command (check all upgrades if no args)
			// TODO: add a cache-upgrade-repo repo_path command (cache all upgrades if no args)
			// TODO: add a show-upgrade-repo-query repo_path[@] command
			// TODO: add a set-upgrade-repo-query repo_path@version_query command
		},
		{
			Name: "rm-repo",
			Aliases: []string{
				"remove-repository", "remove-repositories",
				"del-repo", "delete-repository", "delete-repositories",
				"drop-repo", "drop-repository", "drop-repositories",
			},
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
	const category = "Modify the pallet's package deployments"
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
			Name: "rm-depl",
			Aliases: []string{
				"remove-deployment", "remove-deployments",
				"del-depl", "delete-deployment", "delete-deployments",
			},
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
			Name: "add-depl-feat",
			Aliases: []string{
				"add-deployment-feature",
				"add-deployment-features",
				"enable-depl-feat",
				"enable-deployment-feature",
				"enable-deployment-features",
			},
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
			Name: "rm-depl-feat",
			Aliases: []string{
				"remove-deployment-feature",
				"remove-deployment-features",
				"del-depl-feat",
				"delete-deployment-feature",
				"delete-deployment-features",
				"disable-depl-feat",
				"disable-deployment-feature",
				"disable-deployment-features",
			},
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
