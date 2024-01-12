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
			Action:   lsPkgAction,
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
			Name:     "ls-img",
			Aliases:  []string{"list-images"},
			Category: "Query the cache",
			Usage:    "Lists Docker container images in the local cache",
			Action:   lsImgAction,
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
			Name:     "rm-all",
			Aliases:  []string{"remove-all"},
			Category: "Modify the cache",
			Usage:    "Removes all cached resources",
			Action:   rmAllAction,
		},
		{
			Name:     "rm-repo",
			Aliases:  []string{"remove-repositories"},
			Category: "Modify the cache",
			Usage:    "Removes locally-cached repos",
			Action:   rmRepoAction,
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
