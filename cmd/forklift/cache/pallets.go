package cache

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	fcli "github.com/PlanktoScope/forklift/internal/app/forklift/cli"
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// ls-plt

func lsPltAction(c *cli.Context) error {
	cache, err := getCache(c.String("workspace"))
	if err != nil {
		return err
	}
	if !cache.Exists() {
		return errMissingCache
	}

	// TODO: add a --pattern cli flag for the pattern
	loadedPallets, err := cache.LoadFSPallets("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't identify pallets")
	}
	sort.Slice(loadedPallets, func(i, j int) bool {
		return pallets.ComparePallets(loadedPallets[i].Pallet, loadedPallets[j].Pallet) < 0
	})
	for _, pallet := range loadedPallets {
		fmt.Printf("%s@%s\n", pallet.Def.Pallet.Path, pallet.Version)
	}
	return nil
}

// show-plt

func showPltAction(c *cli.Context) error {
	cache, err := getCache(c.String("workspace"))
	if err != nil {
		return err
	}
	if !cache.Exists() {
		return errMissingCache
	}

	versionedPalletPath := c.Args().First()
	palletPath, version, ok := strings.Cut(versionedPalletPath, "@")
	if !ok {
		return errors.Errorf(
			"Couldn't parse pallet query %s as pallet_path@version", versionedPalletPath,
		)
	}
	pallet, err := cache.LoadFSPallet(palletPath, version)
	if err != nil {
		return errors.Wrapf(err, "couldn't find pallet %s@%s", palletPath, version)
	}
	return printCachedPallet(0, cache, pallet)
}

func printCachedPallet(indent int, cache *forklift.FSPalletCache, pallet *pallets.FSPallet) error {
	fcli.IndentedPrintf(indent, "Cached pallet: %s\n", pallet.Path())
	indent++

	fcli.IndentedPrintf(indent, "Version: %s\n", pallet.Version)
	fcli.IndentedPrintf(indent, "Provided by Git repository: %s\n", pallet.VCSRepoPath)
	fcli.IndentedPrintf(indent, "Path in cache: %s\n", pallets.GetSubdirPath(cache, pallet.FS.Path()))
	fcli.IndentedPrintf(indent, "Description: %s\n", pallet.Def.Pallet.Description)

	readme, err := pallet.LoadReadme()
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load readme file for pallet %s@%s from cache", pallet.Path(), pallet.Version,
		)
	}
	fcli.IndentedPrintln(indent, "Readme:")
	const widthLimit = 100
	fcli.PrintReadme(indent+1, readme, widthLimit)
	return nil
}
