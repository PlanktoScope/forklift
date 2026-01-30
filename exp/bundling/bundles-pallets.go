package bundling

import (
	"path"
	"path/filepath"

	"github.com/pkg/errors"

	ffs "github.com/forklift-run/forklift/exp/fs"
	fplt "github.com/forklift-run/forklift/exp/pallets"
)

const (
	// bundledPalletDirName is the name of the directory containing the bundled pallet.
	bundledPalletDirName = "pallet"
	// bundledMergedPalletDirName is the name of the directory containing the bundled pallet, merged
	// with file imports from its required pallets.
	bundledMergedPalletDirName = "merged-pallet"
)

// FSBundle: Pallets

func (b *FSBundle) SetBundledPallet(pallet *fplt.FSPallet) error {
	shallow := pallet.FS
	for {
		merged, ok := shallow.(*ffs.MergeFS)
		if !ok {
			break
		}
		shallow = merged.Overlay
	}
	if shallow == nil {
		return errors.Errorf("pallet %s was not merged before bundling!", pallet.Path())
	}

	if err := ffs.CopyFS(shallow, filepath.FromSlash(b.getBundledPalletPath())); err != nil {
		return errors.Wrapf(
			err, "couldn't bundle files for unmerged pallet %s from %s", pallet.Path(), pallet.FS.Path(),
		)
	}

	if err := ffs.CopyFS(pallet.FS, filepath.FromSlash(b.getBundledMergedPalletPath())); err != nil {
		return errors.Wrapf(
			err, "couldn't bundle files for merged pallet %s from %s", pallet.Path(), pallet.FS.Path(),
		)
	}
	return nil
}

func (b *FSBundle) getBundledPalletPath() string {
	return path.Join(b.FS.Path(), bundledPalletDirName)
}

func (b *FSBundle) getBundledMergedPalletPath() string {
	return path.Join(b.FS.Path(), bundledMergedPalletDirName)
}

// FSBundle: FSPalletLoader

func (b *FSBundle) LoadFSPallet(palletPath string, version string) (*fplt.FSPallet, error) {
	if b == nil {
		return nil, errors.New("bundle is nil")
	}

	return fplt.LoadFSPallet(b.FS, path.Join(packagesDirName, palletPath))
}

func (b *FSBundle) LoadFSPallets(searchPattern string) ([]*fplt.FSPallet, error) {
	if b == nil {
		return nil, errors.New("bundle is nil")
	}

	return fplt.LoadFSPallets(b.FS, path.Join(packagesDirName, searchPattern))
}
