// Package cache provides subcommands for the local cache
package cache

import (
	"github.com/urfave/cli/v2"
)

var Cmd = &cli.Command{
	Name:  "cache",
	Usage: "Manages the local cache of repos and packages",
	Subcommands: []*cli.Command{
		{
			Name:     "ls-plt",
			Aliases:  []string{"list-pallets"},
			Category: "Query the cache",
			Usage:    "Lists cached pallets",
			Action:   lsPltAction,
		},
		{
			Name:      "show-plt",
			Aliases:   []string{"show-pallet"},
			Category:  "Query the cache",
			Usage:     "Describes a cached pallet",
			ArgsUsage: "pallet_path@version",
			Action:    showPltAction,
		},
		{
			Name:     "ls-repo",
			Aliases:  []string{"list-repositories"},
			Category: "Query the cache",
			Usage:    "Lists cached repos",
			Action:   lsRepoAction,
		},
		{
			Name:      "show-repo",
			Aliases:   []string{"show-repository"},
			Category:  "Query the cache",
			Usage:     "Describes a cached repo",
			ArgsUsage: "repo_path@version",
			Action:    showRepoAction,
		},
		{
			Name:     "ls-pkg",
			Aliases:  []string{"list-packages"},
			Category: "Query the cache",
			Usage:    "Lists packages offered by cached repos",
			// TODO: allow only listing packages matching a glob pattern
			Action: lsPkgAction,
		},
		{
			Name:      "show-pkg",
			Aliases:   []string{"show-package"},
			Category:  "Query the cache",
			Usage:     "Describes a cached package",
			ArgsUsage: "package_path@version",
			Action:    showPkgAction,
		},
		{
			Name:     "ls-dl",
			Aliases:  []string{"list-downloads"},
			Category: "Query the cache",
			Usage:    "Lists cached file downloads",
			Action:   lsDlAction,
		},
		{
			Name:     "ls-img",
			Aliases:  []string{"list-images"},
			Category: "Query the cache",
			Usage:    "Lists Docker container images in the local cache",
			// TODO: allow only listing images matching a glob pattern
			Action: lsImgAction,
		},
		{
			Name:      "show-img",
			Aliases:   []string{"show-image"},
			Category:  "Query the cache",
			Usage:     "Describes a cached Docker container image",
			ArgsUsage: "image_sha",
			Action:    showImgAction,
		},
		{
			Name:      "add-plt",
			Aliases:   []string{"add-pallets"},
			Category:  "Modify the cache",
			Usage:     "Downloads local copies of pallets from remote releases",
			ArgsUsage: "[pallet_path@release]...",
			Action:    addGitRepoAction(getPalletCache),
		},
		{
			Name:      "add-repo",
			Aliases:   []string{"add-repositories"},
			Category:  "Modify the cache",
			Usage:     "Downloads local copies of repos from remote releases",
			ArgsUsage: "[repo_path@release]...",
			Action:    addGitRepoAction(getRepoCache),
		},
		{
			Name:     "rm-all",
			Aliases:  []string{"remove-all"},
			Category: "Modify the cache",
			Usage:    "Removes all cached resources",
			Action:   rmAllAction,
		},
		{
			Name:     "rm-plt",
			Aliases:  []string{"remove-pallets"},
			Category: "Modify the cache",
			Usage:    "Removes locally-cached pallets",
			// TODO: allow only removing repos matching a glob pattern
			Action: rmGitRepoAction("pallet", getPalletCache),
		},
		{
			Name:     "rm-repo",
			Aliases:  []string{"remove-repositories"},
			Category: "Modify the cache",
			Usage:    "Removes locally-cached repos",
			// TODO: allow only removing repos matching a glob pattern
			Action: rmGitRepoAction("repo", getRepoCache),
		},
		{
			Name:     "rm-img",
			Aliases:  []string{"remove-images"},
			Category: "Modify the cache",
			Usage:    "Removes unused Docker container images",
			Action:   rmImgAction,
		},
	},
}
