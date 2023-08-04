// Package cache provides subcommands for the local cache
package cache

import (
	"github.com/urfave/cli/v2"
)

var Cmd = &cli.Command{
	Name:  "cache",
	Usage: "Manages the local cache of pallets and packages",
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
			Name:     "ls-pkg",
			Aliases:  []string{"list-packages"},
			Category: "Query the cache",
			Usage:    "Lists packages offered by cached pallets",
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
			Name:     "rm",
			Aliases:  []string{"remove"},
			Category: "Modify the cache",
			Usage:    "Removes the locally-cached pallets and Docker container images",
			Action:   rmAction,
		},
	},
}
