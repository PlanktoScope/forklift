// Package cache provides subcommands for the local cache
package cache

import (
	"github.com/urfave/cli/v2"
)

var Cmd = &cli.Command{
	Name:  "cache",
	Usage: "Manages the local cache of Pallet repositories and packages",
	Subcommands: []*cli.Command{
		{
			Name:     "ls-repo",
			Aliases:  []string{"ls-r", "list-repositories"},
			Category: "Query the cache",
			Usage:    "Lists cached repositories",
			Action:   lsRepoAction,
		},
		{
			Name:      "show-repo",
			Aliases:   []string{"s-r", "show-repository"},
			Category:  "Query the cache",
			Usage:     "Describes a cached repository",
			ArgsUsage: "repository_path@version",
			Action:    showRepoAction,
		},
		{
			Name:     "ls-pkg",
			Aliases:  []string{"ls-p", "list-packages"},
			Category: "Query the cache",
			Usage:    "Lists packages offered by cached repositories",
			Action:   lsPkgAction,
		},
		{
			Name:      "show-pkg",
			Aliases:   []string{"s-p", "show-package"},
			Category:  "Query the cache",
			Usage:     "Describes a cached package",
			ArgsUsage: "package_path@version",
			Action:    showPkgAction,
		},
		{
			Name:     "ls-img",
			Aliases:  []string{"ls-i", "list-images"},
			Category: "Query the cache",
			Usage:    "Lists Docker container images in the local cache",
			Action:   lsImgAction,
		},
		{
			Name:      "show-img",
			Aliases:   []string{"s-i", "show-image"},
			Category:  "Query the cache",
			Usage:     "Describes a cached Docker container image",
			ArgsUsage: "image_sha",
			Action:    showImgAction,
		},
		{
			Name:     "rm",
			Aliases:  []string{"remove"},
			Category: "Modify the cache",
			Usage:    "Removes the locally-cached repositories and Docker container images",
			Action:   rmAction,
		},
	},
}