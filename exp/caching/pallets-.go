package caching

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	ffs "github.com/forklift-run/forklift/exp/fs"
	fpkg "github.com/forklift-run/forklift/exp/packaging"
	fplt "github.com/forklift-run/forklift/exp/pallets"
)

// FSPalletCache is a [PathedPalletCache] implementation with copies of pallets stored in a
// [fpkg.PathedFS] filesystem.
type FSPalletCache struct {
	// pkgTree is the filesystem which corresponds to the cache of pallets.
	pkgTree *fpkg.FSPkgTree
}

// FSPalletCache

func NewFSPalletCache(fsys ffs.PathedFS) *FSPalletCache {
	return &FSPalletCache{
		pkgTree: &fpkg.FSPkgTree{
			FS: fsys,
		},
	}
}

// Exists checks whether the cache actually exists on the OS's filesystem.
func (c *FSPalletCache) Exists() bool {
	return ffs.DirExists(filepath.FromSlash(c.pkgTree.FS.Path()))
}

// Remove deletes the cache from the OS's filesystem, if it exists.
func (c *FSPalletCache) Remove() error {
	return os.RemoveAll(filepath.FromSlash(c.pkgTree.FS.Path()))
}

// Path returns the path of the cache's filesystem.
func (c *FSPalletCache) Path() string {
	return c.pkgTree.FS.Path()
}

// FSPalletCache: FSPalletLoader

// LoadFSPallet loads the FSPallet with the specified path and version.
// The loaded FSPallet instance is fully initialized.
func (c *FSPalletCache) LoadFSPallet(pltPath string, version string) (*fplt.FSPallet, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	plt, err := fplt.LoadFSPallet(c.pkgTree.FS, fmt.Sprintf("%s@%s", pltPath, version))
	if err != nil {
		return nil, err
	}
	plt.Version = version
	plt.FSPkgTree.Version = version
	return plt, nil
}

// LoadFSPallets loads all FSPallets from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching pallet directories to
// search for.
// The loaded FSPallet instances are fully initialized.
func (c *FSPalletCache) LoadFSPallets(searchPattern string) ([]*fplt.FSPallet, error) {
	if c == nil {
		return nil, nil
	}

	plts, err := fplt.LoadFSPallets(c.pkgTree.FS, searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load pallets from cache")
	}

	// set the Version field of the pallet based on its path in the cache
	for _, plt := range plts {
		var pltPath string
		var ok bool
		if pltPath, plt.Version, ok = strings.Cut(ffs.GetSubdirPath(c, plt.FS.Path()), "@"); !ok {
			return nil, errors.Wrapf(
				err, "couldn't parse path of cached pallet configured at %s as pallet_path@version",
				plt.FS.Path(),
			)
		}
		if pltPath != plt.Path() {
			return nil, errors.Errorf(
				"cached pallet %s is in cache at %s@%s instead of %s@%s",
				plt.Path(), pltPath, plt.Version, plt.Path(), plt.Version,
			)
		}
	}

	return plts, nil
}

// FSPalletCache: FSPkgLoader

// LoadFSPkg loads the FSPkg with the specified path and version.
// The loaded FSPkg instance is fully initialized.
func (c *FSPalletCache) LoadFSPkg(pkgPath string, version string) (*fpkg.FSPkg, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	// Search for the package by starting with the shortest possible package subdirectory path and the
	// longest possible pkg tree path, and shifting path components from the pkg tree path to the package
	// subdirectory path until we successfully load the package.
	palletPath := path.Dir(pkgPath)
	pkgSubdir := path.Base(pkgPath)
	for palletPath != "." && palletPath != "/" {
		pallet, err := c.LoadFSPallet(palletPath, version)
		if err != nil {
			pkgSubdir = path.Join(path.Base(palletPath), pkgSubdir)
			palletPath = path.Dir(palletPath)
			continue
		}

		// FIXME: we must merge the pallet first!
		pkg, err := pallet.LoadFSPkg(pkgSubdir)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't load package %s from pallet %s at version %s", pkgPath, palletPath, version,
			)
		}
		return pkg, nil
	}
	return nil, errors.Errorf("no cached packages were found matching %s@%s", pkgPath, version)
}

// LoadFSPkgs loads all FSPkgs from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching package directories
// to search for.
// The loaded FSPkg instances are fully initialized.
func (c *FSPalletCache) LoadFSPkgs(searchPattern string) ([]*fpkg.FSPkg, error) {
	if c == nil {
		return nil, nil
	}

	pkgs, err := c.pkgTree.LoadFSPkgs(searchPattern)
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgs {
		pallet, err := c.loadFSPalletContaining(ffs.GetSubdirPath(c, pkg.FS.Path()))
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't find the cached pallet providing the cached package at %s", pkg.FS.Path(),
			)
		}
		if err = pkg.AttachFSPkgTree(pallet.FSPkgTree); err != nil {
			return nil, errors.Wrap(err, "couldn't attach cached pallet to cached package")
		}
	}
	return pkgs, nil
}

// loadFSPalletContaining finds and loads the FSPallet which contains the provided subdirectory
// path.
func (c *FSPalletCache) loadFSPalletContaining(
	subdirPath string,
) (pallet *fplt.FSPallet, err error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	if pallet, err = loadFSPalletContaining(c.pkgTree.FS, subdirPath); err != nil {
		return nil, errors.Wrapf(err, "couldn't find any pallet containing %s", subdirPath)
	}
	var palletPath string
	var ok bool
	if palletPath, pallet.Version, ok = strings.Cut(ffs.GetSubdirPath(c, pallet.FS.Path()), "@"); !ok {
		return nil, errors.Wrapf(
			err, "couldn't parse path of cached pallet configured at %s as pallet_path@version",
			pallet.FS.Path(),
		)
	}
	pallet.FSPkgTree.Version = pallet.Version
	if palletPath != pallet.Path() {
		return nil, errors.Errorf(
			"cached pallet %s is in cache at %s@%s instead of %s@%s",
			pallet.Path(), palletPath, pallet.Version, pallet.Path(), pallet.Version,
		)
	}
	return pallet, nil
}

// loadFSPalletContaining loads the FSPallet containing the specified sub-directory path in the
// provided base filesystem.
// The sub-directory path does not have to actually exist.
// In the loaded FSPallet's embedded [Pallet], the version is *not* initialized.
func loadFSPalletContaining(fsys ffs.PathedFS, subdirPath string) (*fplt.FSPallet, error) {
	repoCandidatePath := subdirPath
	for {
		if repo, err := fplt.LoadFSPallet(fsys, repoCandidatePath); err == nil {
			return repo, nil
		}
		repoCandidatePath = path.Dir(repoCandidatePath)
		if repoCandidatePath == "/" || repoCandidatePath == "." {
			// we can't go up anymore!
			return nil, errors.Errorf(
				"no repo declaration file was found in any parent directory of %s", subdirPath,
			)
		}
	}
}
