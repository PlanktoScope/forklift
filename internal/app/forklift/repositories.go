package forklift

import (
	"slices"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/pkg/core"
	ffs "github.com/forklift-run/forklift/pkg/fs"
)

// LoadFSPkgTrees loads all FSPkgTrees from the provided base filesystem matching the specified search
// pattern, appropriately handling pkgTrees defined (implicitly or explicitly) as potentially-layered
// pallets. The search pattern should be a [doublestar] pattern, such as `**`, matching pkgTree
// directories to search for.
// In the embedded [PkgTree] of each loaded FSPkgTree, the version is *not* initialized.
func LoadFSPkgTrees(
	fsys ffs.PathedFS, searchPattern string, palletLoader FSPalletLoader,
) ([]*core.FSPkgTree, error) {
	allPkgTrees := make(map[string]*core.FSPkgTree) // pkgTree FS path -> pkgTree
	pallets, err := LoadFSPallets(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load pallets (which are also pkg trees)")
	}
	for _, pallet := range pallets {
		merged, err := MergeFSPallet(pallet, palletLoader, nil)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't merge pallet %s with any pallets required by it, to use it as a pkg tree",
				pallet.FS.Path(),
			)
		}
		allPkgTrees[pallet.FS.Path()] = merged.PkgTree
	}

	pkgTrees, err := core.LoadFSPkgTrees(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load pkg trees")
	}
	for _, pkgTree := range pkgTrees {
		if _, ok := allPkgTrees[pkgTree.FS.Path()]; ok { // the pkg tree might've already been added as a pallet
			continue
		}
		allPkgTrees[pkgTree.FS.Path()] = pkgTree
	}

	pkgTrees = make([]*core.FSPkgTree, 0, len(allPkgTrees))
	for _, pkgTree := range allPkgTrees {
		pkgTrees = append(pkgTrees, pkgTree)
	}
	slices.SortFunc(pkgTrees, func(a, b *core.FSPkgTree) int {
		return core.ComparePkgTrees(a.PkgTree, b.PkgTree)
	})
	return pkgTrees, nil
}
