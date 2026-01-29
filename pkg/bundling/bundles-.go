// Package bundling implements the Forklift bundling spec for exporting Forklift pallets.
package bundling

import (
	ffs "github.com/forklift-run/forklift/pkg/fs"
	fpkg "github.com/forklift-run/forklift/pkg/packaging"
	"github.com/pkg/errors"
)

// A FSBundle is a Forklift pallet bundle stored at the root of a [fs.FS] filesystem.
type FSBundle struct {
	// Bundle is the pallet bundle at the root of the filesystem.
	Bundle
	// FSPkgTree is the package tree at the root of the bundle's packages directory.
	FSPkgTree *fpkg.FSPkgTree
	// FS is a filesystem which contains the bundle's contents.
	FS ffs.PathedFS
}

// A Bundle is a Forklift pallet bundle, a complete compilation of all files (except container
// images) needed for a pallet to be applied to a Docker host. Required pallets and packages are
// included directly in the bundle.
type Bundle struct {
	// Manifest is the Forklift bundle manifest for the pallet bundle.
	Manifest BundleManifest
}

// FSBundle

func NewFSBundle(path string) (b *FSBundle, err error) {
	fsys := ffs.DirFS(path)
	pkgfs, err := fsys.Sub(packagesDirName)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't load package tree from bundle")
	}

	return &FSBundle{
		FS: fsys,
		FSPkgTree: &fpkg.FSPkgTree{
			FS: pkgfs,
		},
	}, nil
}

// LoadFSBundle loads a FSBundle from a specified directory path in the provided base filesystem.
func LoadFSBundle(fsys ffs.PathedFS, subdirPath string) (b *FSBundle, err error) {
	b = &FSBundle{}
	if b.FS, err = fsys.Sub(subdirPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", subdirPath, fsys.Path(),
		)
	}

	pkgfs, err := b.FS.Sub(packagesDirName)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't load package tree from bundle")
	}
	b.FSPkgTree = &fpkg.FSPkgTree{
		FS: pkgfs,
	}

	if b.Bundle.Manifest, err = loadBundleManifest(b.FS, BundleManifestFile); err != nil {
		return nil, errors.Errorf("couldn't load bundle manifest")
	}
	for path, req := range b.Bundle.Manifest.Includes.Pallets {
		if req.Req.VersionLock.Version, err = req.Req.VersionLock.Decl.Version(); err != nil {
			return nil, errors.Wrapf(
				err, "couldn't determine requirement version of included pallet %s", path,
			)
		}
		b.Bundle.Manifest.Includes.Pallets[path] = req
	}
	return b, nil
}

func (b *FSBundle) Path() string {
	return b.FS.Path()
}
