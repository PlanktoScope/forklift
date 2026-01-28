// Package plt provides subcommands for the local pallet
package plt

import (
	"slices"

	"github.com/urfave/cli/v2"

	fcli "github.com/forklift-run/forklift/internal/app/forklift/cli"
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
		Usage:   "Manages the local pallet",
		Subcommands: slices.Concat(
			[]*cli.Command{
				{
					Name: "switch",
					Usage: "Initializes or replaces the local pallet with the specified pallet, and " +
						"stages the specified pallet",
					ArgsUsage: "[[pallet_path]@[version_query]]",
					Action:    switchAction(versions),
					Flags: []cli.Flag{
						&cli.BoolFlag{
							Name: "set-upgrade-query",
							Usage: "Remember the pallet path and version query used in this operation for " +
								"future clone/switch/upgrade operations",
							Value: true,
						},
						&cli.BoolFlag{
							Name: "force",
							Usage: "Even if the local pallet already exists and has uncommitted/unpushed " +
								"changes, replace it",
						},
						&cli.BoolFlag{
							Name:  "cache-img",
							Usage: "Download container images (this flag is ignored if --apply is set)",
							Value: true,
						},
						&cli.BoolFlag{
							Name:  "apply",
							Usage: "Immediately apply the pallet after staging it",
						},
					},
				},
			},
			makeUpgradeSubcmds(versions),
			makeUseSubcmds(versions),
			makeQuerySubcmds(),
			makeModifySubcmds(versions),
		),
	}
}

func makeUpgradeSubcmds(versions Versions) []*cli.Command {
	const category = "Upgrade the pallet"
	return []*cli.Command{
		{
			Name:     "upgrade",
			Category: category,
			Usage: "Replaces the local pallet with an upgraded version, updates the cache, and " +
				"stages the pallet",
			ArgsUsage: "[[pallet_path]@[version_query]]",
			Action:    upgradeAction(versions),
			Flags:     upgradeFlags,
		},
		{
			Name:     "check-upgrade",
			Category: category,
			// TODO: also check whether the upgrade is cached
			Usage:     "Checks whether an upgrade is available",
			ArgsUsage: "[[pallet_path]@[version_query]]",
			Action:    checkUpgradeAction,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "allow-downgrade",
					Usage: "Allow upgrading to an older version (i.e. performing a downgrade)",
				},
				// TODO: add a --require-cached flag
			},
		},
		// TODO: add a cache-upgrade command
		{
			Name:     "show-upgrade-query",
			Category: category,
			Usage:    "Shows the query used for pallet upgrades",
			Action:   showUpgradeQueryAction,
		},
		{
			Name:      "set-upgrade-query",
			Category:  category,
			Usage:     "Changes the query used for pallet upgrades",
			ArgsUsage: "[[pallet_path]@[version_query]]",
			Action:    setUpgradeQueryAction,
		},
	}
}

var upgradeFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:  "allow-downgrade",
		Usage: "Allow upgrading to an older version (i.e. performing a downgrade)",
	},
	&cli.BoolFlag{
		Name: "set-upgrade-query",
		Usage: "Remember the pallet path and version query used in this operation for future " +
			"clone/switch/upgrade operations",
		Value: true,
	},
	&cli.BoolFlag{
		Name: "force",
		Usage: "Even if the local pallet has uncommitted/unpushed changes, replace it with the " +
			"upgraded version",
	},
	&cli.BoolFlag{
		Name:  "cache-img",
		Usage: "Download container images (this flag is ignored if --apply is set)",
		Value: true,
	},
	&cli.BoolFlag{
		Name:  "apply",
		Usage: "Immediately apply the upgraded pallet after staging it",
	},
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
					Name:  "cache-img",
					Usage: "Download container images",
					Value: true,
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
			Name:     "cache-plt",
			Aliases:  []string{"cache-pallets"},
			Category: category,
			Usage:    "Updates the cache with the pallets required by the local pallet",
			Action:   cachePltAction(versions),
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
	return slices.Concat(
		[]*cli.Command{
			{
				Name:     "show",
				Category: category,
				Usage:    "Describes the local pallet",
				Action:   showAction,
			},
		},
		makeQueryPltReqSubcmds(category),
		makeQueryImportSubcmds(category),
		makeQueryFileSubcmds(category),
		makeQueryPkgSubcmds(category),
		makeQueryDeplSubcmds(category),
		makeQueryFeatSubcmds(category),
		[]*cli.Command{
			{
				Name:     "ls-dl",
				Aliases:  []string{"list-downloads"},
				Category: category,
				Usage:    "Lists the files to be downloaded for export by the local pallet",
				Action:   lsDlAction,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "include-disabled",
						Usage: "Also list images for disabled package deployments",
					},
				},
			},
			{
				Name:     "ls-img",
				Aliases:  []string{"list-images"},
				Category: category,
				Usage:    "Lists the Docker container images required by the local pallet",
				Action:   lsImgAction,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "include-disabled",
						Usage: "Also list images for disabled package deployments",
					},
				},
			},
		},
	)
}

func makeQueryPltReqSubcmds(category string) []*cli.Command {
	return slices.Concat(
		[]*cli.Command{
			{
				Name:     "ls-plt",
				Aliases:  []string{"list-pallets"},
				Category: category,
				Usage:    "Lists available pallets which the local pallet may import files from",
				Action:   lsPltAction,
			},
			{
				Name:      "show-plt",
				Aliases:   []string{"show-pallet"},
				Category:  category,
				Usage:     "Describes an available pallet which the local pallet may import files from",
				ArgsUsage: "plt_path",
				Action:    showPltAction,
			},
			{
				Name:      "show-plt-version",
				Aliases:   []string{"show-pallet-version"},
				Category:  category,
				Usage:     "Prints the required version of the available pallet",
				ArgsUsage: "plt_path",
				Action:    showPltVersionAction,
			},
		},
		makeQueryPltFileSubcmds(category),
		makeQueryPltFeatSubcmds(category),
	)
}

func makeQueryPltFileSubcmds(category string) []*cli.Command {
	return []*cli.Command{
		{
			Name:     "ls-plt-file",
			Aliases:  []string{"list-pallet-files"},
			Category: category,
			Usage: "Lists non-directory files in the specified pallet which the local pallet may " +
				"import files from",
			ArgsUsage: "pallet_path [path_glob]",
			Action:    lsPltFileAction,
		},
		{
			Name:     "locate-plt-file",
			Aliases:  []string{"locate-pallet-files"},
			Category: category,
			Usage: "Prints the absolute filesystem path of the specified file in the specified " +
				"pallet which the local pallet may import files from",
			ArgsUsage: "pallet_path file_path",
			Action:    locatePltFileAction,
		},
		{
			Name:     "show-plt-file",
			Aliases:  []string{"show-pallet-files"},
			Category: category,
			Usage: "Prints the specified file in the specified pallet which the local pallet may " +
				"import files from",
			ArgsUsage: "pallet_path file_path",
			Action:    showPltFileAction,
		},
	}
}

func makeQueryPltFeatSubcmds(category string) []*cli.Command {
	return []*cli.Command{
		{
			Name:     "ls-plt-feat",
			Aliases:  []string{"list-pallet-features"},
			Category: category,
			Usage: "Lists features flags exposed by the specified pallet which the local pallet may " +
				"import files from",
			ArgsUsage: "pallet_path",
			Action:    lsPltFeatAction,
		},
		{
			Name:     "show-plt-feat",
			Aliases:  []string{"show-pallet-feature"},
			Category: category,
			Usage: "Prints the specified feature exposed by the specified pallet which the local " +
				"pallet may import files from",
			ArgsUsage: "pallet_path feature_name",
			Action:    showPltFeatAction,
		},
	}
}

func makeQueryImportSubcmds(category string) []*cli.Command {
	return []*cli.Command{
		{
			Name:     "ls-imp",
			Aliases:  []string{"list-imports"},
			Category: category,
			Usage:    "Lists import groups specified by the local pallet",
			Action:   lsImpAction,
		},
		{
			Name:      "show-imp",
			Aliases:   []string{"show-import"},
			Category:  category,
			Usage:     "Describes an import group specified by the local pallet",
			ArgsUsage: "import_name",
			Action:    showImpAction,
		},
	}
}

func makeQueryFileSubcmds(category string) []*cli.Command {
	return []*cli.Command{
		{
			Name:      "ls-file",
			Aliases:   []string{"list-files"},
			Category:  category,
			Usage:     "Lists non-directory files in the local pallet",
			ArgsUsage: "[path_glob]",
			Action:    lsFileAction,
		},
		{
			Name:      "locate-file",
			Category:  category,
			Usage:     "Prints the absolute filesystem path of the specified file in the local pallet",
			ArgsUsage: "file_path",
			Action:    locateFileAction,
		},
		{
			Name:      "show-file",
			Category:  category,
			Usage:     "Prints the specified file in the local pallet",
			ArgsUsage: "file_path",
			Action:    showFileAction,
		},
	}
}

func makeQueryPkgSubcmds(category string) []*cli.Command {
	return []*cli.Command{
		{
			Name:     "ls-pkg",
			Aliases:  []string{"list-packages"},
			Category: category,
			Usage:    "Lists packages available in the local pallet",
			Action:   lsPkgAction,
		},
		{
			Name:      "locate-pkg",
			Aliases:   []string{"locate-package"},
			Category:  category,
			Usage:     "Prints the absolute filesystem path of a package available in the local pallet",
			ArgsUsage: "package_path",
			Action:    locatePkgAction,
		},
		{
			Name:      "show-pkg",
			Aliases:   []string{"show-package"},
			Category:  category,
			Usage:     "Describes a package available in the local pallet",
			ArgsUsage: "package_path",
			Action:    showPkgAction,
		},
	}
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

func makeQueryFeatSubcmds(category string) []*cli.Command {
	return []*cli.Command{
		{
			Name:     "ls-feat",
			Aliases:  []string{"list-features"},
			Category: category,
			Usage:    "Lists the feature flags exposed by the local pallet for other pallets to import",
			Action:   lsFeatAction,
		},
		{
			Name:      "show-feat",
			Aliases:   []string{"show-feature"},
			Category:  category,
			Usage:     "Describes a feature exposed by the local pallet for other pallets to import",
			ArgsUsage: "feature_name",
			Action:    showFeatAction,
		},
	}
}

func makeModifySubcmds(versions Versions) []*cli.Command {
	const category = "Modify the pallet"
	return slices.Concat(
		makeModifyGitSubcmds(versions),
		[]*cli.Command{
			{
				Name:     "del",
				Aliases:  []string{"delete"},
				Category: category,
				Usage:    "Removes the local pallet",
				Action:   delAction,
			},
		},
		makeModifyFileSubcmds(),
		makeModifyPltSubcmds(versions),
		// TODO: add `add-imp`, `del-imp`, `set-imp-disabled`, `unset-imp-disabled`,
		// `add-imp-mod`, and `del-imp-mod` subcommands
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
			ArgsUsage: "[[pallet_path]@[version_query]]",
			Flags: slices.Concat(
				[]cli.Flag{
					&cli.BoolFlag{
						Name: "set-upgrade-query",
						Usage: "Remember the pallet path and version query used in this operation for " +
							"future clone/switch/upgrade operations",
						Value: true,
					},
					&cli.BoolFlag{
						Name: "force",
						Usage: "If a local pallet already exists, delete it to replace it with the specified" +
							"pallet",
					},
					&cli.BoolFlag{
						Name:  "cache-req",
						Usage: "Download repositories and pallets required by this pallet after cloning",
						Value: true,
					},
				},
				modifyBaseFlags,
			),
			Action: cloneAction(versions),
		},
		// TODO: add a "checkout @version_query" action; it needs a --force flag to overwrite a dirty
		// working directory
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
			Flags: slices.Concat(
				[]cli.Flag{
					&cli.BoolFlag{
						Name:  "cache-req",
						Usage: "Download repositories and pallets required by this pallet after pulling",
						Value: true,
					},
					// TODO: add an option to fall back to a rebase if a fast-forward is not possible
				},
				modifyBaseFlags,
			),
			Action: pullAction(versions),
		},
		// TODO: add a "push" action?
		// remoteCmd,
	}
}

var modifyBaseFlags []cli.Flag = []cli.Flag{
	&cli.BoolFlag{
		Name: "stage",
		Usage: "Immediately stage the pallet after updating it (this flag is ignored if --apply " +
			"is set)",
	},
	&cli.BoolFlag{
		Name:  "cache-img",
		Usage: "Download container images (this flag is only used if --stage is set)",
		Value: true,
	},
	&cli.BoolFlag{
		Name:  "apply",
		Usage: "Immediately stage and apply the pallet after updating it",
	},
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

func makeModifyFileSubcmds() []*cli.Command {
	const category = "Modify the pallet's files"
	return []*cli.Command{
		{
			Name:      "edit-file",
			Category:  category,
			Usage:     "Edits the specified file in the development pallet",
			ArgsUsage: "file_path",
			Action:    editFileAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "editor",
					Usage:   "Path of text editor",
					EnvVars: []string{"EDITOR"},
				},
			},
		},
		{
			Name:      "del-file",
			Aliases:   []string{"delete-file"},
			Category:  category,
			Usage:     "Removes the specified file in the development pallet",
			ArgsUsage: "file_path",
			Action:    delFileAction,
		},
	}
}

func makeModifyPltSubcmds(versions Versions) []*cli.Command {
	const category = "Modify the pallet's requirements"
	return []*cli.Command{
		{
			Name: "add-plt",
			Aliases: []string{
				"add-pallet", "add-pallets",
				"req-plt", "require-pallet", "require-pallets",
			},
			Category: category,
			Usage: "Adds (or re-adds) pallet requirements to the pallet, tracking specified versions " +
				"or branches",
			ArgsUsage: "[plt_path@version_query]...",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name: "cache-req",
					Usage: "Download repositories and pallets required by this pallet after adding the " +
						"pallet",
					Value: true,
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
			Name: "del-plt",
			Aliases: []string{
				"delete-pallet", "delete-pallets", "drop-plt", "drop-pallet", "drop-pallets",
			},
			Category:  category,
			Usage:     "Removes pallet requirements from the pallet",
			ArgsUsage: "plt_path...",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name: "force",
					Usage: "Remove specified pallet requirements even if some declared file imports" +
						"depend on them",
				},
			},
			Action: delPltAction(versions),
		},
	}
}

func makeModifyDeplSubcmds( //nolint:funlen // this is already decomposed; it's hard to split more
	versions Versions,
) []*cli.Command {
	const category = "Modify the pallet's package deployments"
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
						Name:    "feat",
						Aliases: []string{"feature", "features"},
						Usage:   "Enable the specified feature in the package deployment",
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
				modifyDeplBaseFlags,
			),
			Action: addDeplAction(versions),
		},
		{
			Name:      "del-depl",
			Aliases:   []string{"delete-deployment", "delete-deployments"},
			Category:  category,
			Usage:     "Removes deployment from the pallet",
			ArgsUsage: "deployment_name...",
			Flags:     modifyDeplBaseFlags,
			Action:    delDeplAction(versions),
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
							"enabled package features invalid",
					},
				},
				modifyDeplBaseFlags,
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
						Usage: "Enable the specified package features even if they're not allowed by the " +
							"deployment's package",
					},
				},
				modifyDeplBaseFlags,
			),
			Action: addDeplFeatAction(versions),
		},
		{
			Name: "del-depl-feat",
			Aliases: []string{
				"delete-deployment-feature", "delete-deployment-features",
				"disable-depl-feat", "disable-deployment-feature", "disable-deployment-features",
			},
			Category:  category,
			Usage:     "Disables the specified package features in the specified deployment",
			ArgsUsage: "deployment_name feature_name...",
			Flags:     modifyDeplBaseFlags,
			Action:    delDeplFeatAction(versions),
		},
		{
			Name:      "set-depl-disabled",
			Aliases:   []string{"set-deployment-disabled", "disable-depl", "disable-deployment"},
			Category:  category,
			Usage:     "Disables the specified deployment",
			ArgsUsage: "deployment_name",
			Flags:     modifyDeplBaseFlags,
			Action:    setDeplDisabledAction(versions, true),
		},
		{
			Name:      "unset-depl-disabled",
			Aliases:   []string{"unset-deployment-disabled", "enable-depl", "enable-deployment"},
			Category:  category,
			Usage:     "Enables the specified deployment",
			ArgsUsage: "deployment_name",
			Flags:     modifyDeplBaseFlags,
			Action:    setDeplDisabledAction(versions, false),
		},
	}
}

var modifyDeplBaseFlags []cli.Flag = []cli.Flag{
	&cli.BoolFlag{
		Name: "stage",
		Usage: "Immediately stage the pallet after making the modification (this flag is ignored " +
			"if --apply is set)",
	},
	&cli.BoolFlag{
		Name:  "cache-img",
		Usage: "Download container images (this flag is only used if --stage is set)",
		Value: true,
	},
	&cli.BoolFlag{
		Name:  "apply",
		Usage: "Immediately stage and apply the pallet after making the modification",
	},
}
