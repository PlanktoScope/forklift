package packaging

import (
	"cmp"

	"github.com/pkg/errors"

	ffs "github.com/forklift-run/forklift/exp/fs"
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
func (t *FSPkgTree) LoadFSPkg(pkgSubdir string) (pkg *FSPkg, err error) {
	if t == nil {
		return nil, errors.New("pkg tree is nil")
	}

	if pkg, err = loadFSPkg(t.FS, pkgSubdir); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load package %s from pkg tree %s", pkgSubdir, t.FS.Path(),
		)
	}
	if err = pkg.AttachFSPkgTree(t); err != nil {
		return nil, errors.Wrap(err, "couldn't attach pkg tree to package")
	}
	return pkg, nil
}

// LoadFSPkgs loads all packages in the [FSPkgTree] instance.
// The loaded packages are fully initialized.
func (t *FSPkgTree) LoadFSPkgs(searchPattern string) ([]*FSPkg, error) {
	if t == nil {
		return nil, errors.New("pkg tree is nil")
	}

	pkgs, err := loadFSPkgs(t.FS, searchPattern)
	if err != nil {
		return nil, err
	}
	for _, pkg := range pkgs {
		if err = pkg.AttachFSPkgTree(t); err != nil {
			return nil, errors.Wrap(err, "couldn't attach pkg tree to package")
		}
	}
	return pkgs, nil
}

// FSPkgTree: Pather

// Path returns either the [FSPkgTree] instance's root path (if specified) or its path on the
// filesystem.
func (t *FSPkgTree) Path() string {
	return cmp.Or(t.RootPath, t.FS.Path())
}
