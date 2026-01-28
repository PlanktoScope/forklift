package core

import (
	"cmp"

	ffs "github.com/forklift-run/forklift/pkg/fs"
	"github.com/pkg/errors"
)

// A FSPkgTree is a [fs.FS] filesystem whose packages are identified by their subdirectory paths
// relative to the filesystem root.
type FSPkgTree struct {
	// FS is a filesystem which contains the package tree's contents.
	FS ffs.PathedFS
	// RootPath is an optional path identifying the package tree.
	RootPath string
	// Version is an optional tree version.
	Version string
}

// LoadFSPkg loads a package at the specified filesystem path from the [FSPkgTree] instance
// The loaded package is fully initialized.
func (r *FSPkgTree) LoadFSPkg(pkgSubdir string) (pkg *FSPkg, err error) {
	if pkg, err = LoadFSPkg(r.FS, pkgSubdir); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load package %s from pkg tree %s", pkgSubdir, r.FS.Path(),
		)
	}
	if err = pkg.AttachFSPkgTree(r); err != nil {
		return nil, errors.Wrap(err, "couldn't attach pkg tree to package")
	}
	return pkg, nil
}

// LoadFSPkgs loads all packages in the [FSPkgTree] instance.
// The loaded packages are fully initialized.
func (r *FSPkgTree) LoadFSPkgs(searchPattern string) ([]*FSPkg, error) {
	pkgs, err := LoadFSPkgs(r.FS, searchPattern)
	if err != nil {
		return nil, err
	}
	for _, pkg := range pkgs {
		if err = pkg.AttachFSPkgTree(r); err != nil {
			return nil, errors.Wrap(err, "couldn't attach pkg tree to package")
		}
	}
	return pkgs, nil
}

// FSPkgTree: Pather

// Path returns either the [FSPkgTree] instance's root path (if specified) or its path on the
// filesystem.
func (r *FSPkgTree) Path() string {
	return cmp.Or(r.RootPath, r.FS.Path())
}
