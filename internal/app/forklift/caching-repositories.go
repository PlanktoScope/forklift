package forklift

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/pkg/core"
	ffs "github.com/forklift-run/forklift/pkg/fs"
)

// FSPkgTreeCache

// Exists checks whether the cache actually exists on the OS's filesystem.
func (c *FSPkgTreeCache) Exists() bool {
	return DirExists(filepath.FromSlash(c.FS.Path()))
}

// Remove deletes the cache from the OS's filesystem, if it exists.
func (c *FSPkgTreeCache) Remove() error {
	return os.RemoveAll(filepath.FromSlash(c.FS.Path()))
}

// Path returns the path of the cache's filesystem.
func (c *FSPkgTreeCache) Path() string {
	return c.FS.Path()
}

// FSPkgTreeCache: FSPkgTreeLoader

// LoadFSPkgTree loads the FSPkgTree with the specified path and version.
// The loaded FSPkgTree instance is fully initialized.
func (c *FSPkgTreeCache) LoadFSPkgTree(
	pkgTreePath string, version string,
) (*core.FSPkgTree, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	pkgTree, err := core.LoadFSPkgTree(c.FS, fmt.Sprintf("%s@%s", pkgTreePath, version))
	if err != nil {
		return nil, err
	}
	pkgTree.Version = version
	return pkgTree, nil
}

// LoadFSPkgTrees loads all FSPkgTrees from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching pkg tree directories to
// search for.
// The loaded FSPkgTree instances are fully initialized.
func (c *FSPkgTreeCache) LoadFSPkgTrees(searchPattern string) ([]*core.FSPkgTree, error) {
	if c == nil {
		return nil, nil
	}

	pkgTrees, err := core.LoadFSPkgTrees(c.FS, searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load pkg trees from cache")
	}

	// set the Version field of the pkg tree based on its path in the cache
	for _, pkgTree := range pkgTrees {
		var pkgTreePath string
		var ok bool
		if pkgTreePath, pkgTree.Version, ok = strings.Cut(ffs.GetSubdirPath(c, pkgTree.FS.Path()), "@"); !ok {
			return nil, errors.Wrapf(
				err, "couldn't parse path of cached pkg tree configured at %s as pkgTree_path@version",
				pkgTree.FS.Path(),
			)
		}
		if pkgTreePath != pkgTree.Path() {
			return nil, errors.Errorf(
				"cached pkg tree %s is in cache at %s@%s instead of %s@%s",
				pkgTree.Path(), pkgTreePath, pkgTree.Version, pkgTree.Path(), pkgTree.Version,
			)
		}
	}

	return pkgTrees, nil
}

// FSPkgTreeCache: FSPkgLoader

// LoadFSPkg loads the FSPkg with the specified path and version.
// The loaded FSPkg instance is fully initialized.
func (c *FSPkgTreeCache) LoadFSPkg(pkgPath string, version string) (*core.FSPkg, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	// Search for the package by starting with the shortest possible package subdirectory path and the
	// longest possible pkg tree path, and shifting path components from the pkg tree path to the package
	// subdirectory path until we successfully load the package.
	pkgTreePath := path.Dir(pkgPath)
	pkgSubdir := path.Base(pkgPath)
	for pkgTreePath != "." && pkgTreePath != "/" {
		pkgTree, err := c.LoadFSPkgTree(pkgTreePath, version)
		if err != nil {
			pkgSubdir = path.Join(path.Base(pkgTreePath), pkgSubdir)
			pkgTreePath = path.Dir(pkgTreePath)
			continue
		}
		pkg, err := pkgTree.LoadFSPkg(pkgSubdir)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't load package %s from pkg tree %s at version %s",
				pkgPath, pkgTreePath, version,
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
func (c *FSPkgTreeCache) LoadFSPkgs(searchPattern string) ([]*core.FSPkg, error) {
	if c == nil {
		return nil, nil
	}

	pkgs, err := core.LoadFSPkgs(c.FS, searchPattern)
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgs {
		pkgTree, err := c.loadFSpkgTreeContaining(ffs.GetSubdirPath(c, pkg.FS.Path()))
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't find the cached pkg tree providing the cached package at %s", pkg.FS.Path(),
			)
		}
		if err = pkg.AttachFSPkgTree(pkgTree); err != nil {
			return nil, errors.Wrap(err, "couldn't attach cached pkg tree to cached package")
		}
	}
	return pkgs, nil
}

// loadFSpkgTreeContaining finds and loads the FSPkgTree which contains the provided subdirectory
// path.
func (c *FSPkgTreeCache) loadFSpkgTreeContaining(
	subdirPath string,
) (pkgTree *core.FSPkgTree, err error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	if pkgTree, err = core.LoadFSPkgTreeContaining(c.FS, subdirPath); err != nil {
		return nil, errors.Wrapf(err, "couldn't find any pkg tree containing %s", subdirPath)
	}
	var pkgTreePath string
	var ok bool
	if pkgTreePath, pkgTree.Version, ok = strings.Cut(ffs.GetSubdirPath(c, pkgTree.FS.Path()), "@"); !ok {
		return nil, errors.Wrapf(
			err, "couldn't parse path of cached pkg tree configured at %s as pkgTree_path@version",
			pkgTree.FS.Path(),
		)
	}
	if pkgTreePath != pkgTree.Path() {
		return nil, errors.Errorf(
			"cached pkg tree %s is in cache at %s@%s instead of %s@%s",
			pkgTree.Path(), pkgTreePath, pkgTree.Version, pkgTree.Path(), pkgTree.Version,
		)
	}
	return pkgTree, nil
}

// LayeredPkgTreeCache

// Path returns the path of the underlying cache.
func (c *LayeredPkgTreeCache) Path() string {
	return c.Underlay.Path()
}

// LoadFSPkgTree loads the FSPkgTree with the specified path and version.
// The loaded FSPkgTree instance is fully initialized.
// If the overlay cache expects to have the pkg tree, it will attempt to load the pkg tree; otherwise,
// the underlay cache will attempt to load the pkg tree.
func (c *LayeredPkgTreeCache) LoadFSPkgTree(
	pkgTreePath string, version string,
) (*core.FSPkgTree, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	if c.Overlay.IncludesFSPkgTree(pkgTreePath, version) {
		pkgTree, err := c.Overlay.LoadFSPkgTree(pkgTreePath, version)
		return pkgTree, errors.Wrap(err, "couldn't load pkg tree from overlay")
	}
	pkgTree, err := c.Underlay.LoadFSPkgTree(pkgTreePath, version)
	return pkgTree, errors.Wrap(err, "couldn't load pkg tree from underlay")
}

// LoadFSPkgTrees loads all FSPkgTrees from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching pkg tree directories to
// search for.
// The loaded FSPkgTree instances are fully initialized.
// All matching pkg trees from the overlay cache will be included; all matching pkg trees from the
// underlay cache will also be included, except for those pkg trees which the overlay cache expected
// to have.
func (c *LayeredPkgTreeCache) LoadFSPkgTrees(searchPattern string) ([]*core.FSPkgTree, error) {
	if c == nil {
		return nil, nil
	}

	loadedPkgTrees, err := c.Overlay.LoadFSPkgTrees(searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load pkg trees from overlay")
	}

	underlayPkgTrees, err := c.Underlay.LoadFSPkgTrees(searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load pkg trees from underlay")
	}
	for _, pkgTree := range underlayPkgTrees {
		if c.Overlay.IncludesFSPkgTree(pkgTree.Path(), pkgTree.Version) {
			continue
		}
		loadedPkgTrees = append(loadedPkgTrees, pkgTree)
	}

	sort.Slice(loadedPkgTrees, func(i, j int) bool {
		return core.ComparePkgTrees(
			loadedPkgTrees[i].PkgTree,
			loadedPkgTrees[j].PkgTree,
		) == core.CompareLT
	})
	return loadedPkgTrees, nil
}

// LoadFSPkg loads the FSPkg with the specified path and version.
// The loaded FSPkg instance is fully initialized.
// If the overlay cache expects to have the package, it will attempt to load the package; otherwise,
// the underlay cache will attempt to load the package.
func (c *LayeredPkgTreeCache) LoadFSPkg(pkgPath string, version string) (*core.FSPkg, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	if c.Overlay.IncludesFSPkg(pkgPath, version) {
		pkg, err := c.Overlay.LoadFSPkg(pkgPath, version)
		return pkg, errors.Wrap(err, "couldn't load package from overlay")
	}

	pkg, err := c.Underlay.LoadFSPkg(pkgPath, version)
	return pkg, errors.Wrap(err, "couldn't load package from underlay")
}

// LoadFSPkgs loads all FSPkgs from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching package directories
// to search for.
// The loaded FSPkg instances are fully initialized.
// All matching packages from the overlay cache will be included; all matching packages from the
// underlay cache will also be included, except for those packages which the overlay cache expected
// to have.
func (c *LayeredPkgTreeCache) LoadFSPkgs(searchPattern string) ([]*core.FSPkg, error) {
	if c == nil {
		return nil, nil
	}

	pkgs, err := c.Overlay.LoadFSPkgs(searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load packages from overlay")
	}

	underlayPkgs, err := c.Underlay.LoadFSPkgs(searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load packages from underlay")
	}
	for _, pkg := range underlayPkgs {
		if c.Overlay.IncludesFSPkg(pkg.Path(), pkg.PkgTree.Version) {
			continue
		}
		pkgs = append(pkgs, pkg)
	}

	sort.Slice(pkgs, func(i, j int) bool {
		return core.ComparePkgs(pkgs[i].Pkg, pkgs[j].Pkg) == core.CompareLT
	})
	return pkgs, nil
}
