package cache

import (
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
)

// ls-plt

func lsPltAction(c *cli.Context) error {
	cache, err := getPalletCache(c.String("workspace"))
	if err != nil {
		return err
	}
	if !cache.Exists() {
		return errMissingCache
	}

	// TODO: add a --pattern cli flag for the pattern
	return lsGitRepo("pallet", "**", cache.LoadFSPallets, func(r, s *forklift.FSPallet) int {
		return forklift.ComparePallets(r.Pallet, s.Pallet)
	})
}

// show-plt

func showPltAction(c *cli.Context) error {
	cache, err := getPalletCache(c.String("workspace"))
	if err != nil {
		return err
	}
	if !cache.Exists() {
		return errMissingCache
	}

	return showGitRepo(cache, c.Args().First(), cache.LoadFSPallet, fcli.PrintCachedPallet)
}
