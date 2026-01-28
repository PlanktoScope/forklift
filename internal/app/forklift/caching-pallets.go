package forklift

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/pkg/core"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	"github.com/forklift-run/forklift/pkg/structures"
)

// FSPalletCache

// Exists checks whether the cache actually exists on the OS's filesystem.
func (c *FSPalletCache) Exists() bool {
	return DirExists(filepath.FromSlash(c.FS.Path()))
}

// Remove deletes the cache from the OS's filesystem, if it exists.
func (c *FSPalletCache) Remove() error {
	return os.RemoveAll(filepath.FromSlash(c.FS.Path()))
}

// Path returns the path of the cache's filesystem.
func (c *FSPalletCache) Path() string {
	return c.FS.Path()
}

// FSPalletCache: FSPalletLoader

// LoadFSPallet loads the FSPallet with the specified path and version.
// The loaded FSPallet instance is fully initialized.
func (c *FSPalletCache) LoadFSPallet(pltPath string, version string) (*FSPallet, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	plt, err := LoadFSPallet(c.FS, fmt.Sprintf("%s@%s", pltPath, version))
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
func (c *FSPalletCache) LoadFSPallets(searchPattern string) ([]*FSPallet, error) {
	if c == nil {
		return nil, nil
	}

	plts, err := LoadFSPallets(c.FS, searchPattern)
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
func (c *FSPalletCache) LoadFSPkg(pkgPath string, version string) (*core.FSPkg, error) {
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
func (c *FSPalletCache) LoadFSPkgs(searchPattern string) ([]*core.FSPkg, error) {
	if c == nil {
		return nil, nil
	}

	pkgs, err := core.LoadFSPkgs(c.FS, searchPattern)
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
func (c *FSPalletCache) loadFSPalletContaining(subdirPath string) (pallet *FSPallet, err error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	if pallet, err = loadFSPalletContaining(c.FS, subdirPath); err != nil {
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
func loadFSPalletContaining(fsys ffs.PathedFS, subdirPath string) (*FSPallet, error) {
	repoCandidatePath := subdirPath
	for {
		if repo, err := LoadFSPallet(fsys, repoCandidatePath); err == nil {
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

// LayeredPalletCache

// Path returns the path of the underlying cache.
func (c *LayeredPalletCache) Path() string {
	return c.Underlay.Path()
}

// LoadFSPallet loads the FSPallet with the specified path and version.
// The loaded FSPallet instance is fully initialized.
// If the overlay cache expects to have the pallet, it will attempt to load the pallet; otherwise,
// the underlay cache will attempt to load the pallet.
func (c *LayeredPalletCache) LoadFSPallet(pltPath string, version string) (*FSPallet, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	if c.Overlay != nil && c.Overlay.IncludesFSPallet(pltPath, version) {
		plt, err := c.Overlay.LoadFSPallet(pltPath, version)
		return plt, errors.Wrap(err, "couldn't load pallet from overlay")
	}
	plt, err := c.Underlay.LoadFSPallet(pltPath, version)
	return plt, errors.Wrap(err, "couldn't load pallet from underlay")
}

// LoadFSPallets loads all FSPallets from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching pallet directories to
// search for.
// The loaded FSPallet instances are fully initialized.
// All matching pallets from the overlay cache will be included; all matching pallets from the
// underlay cache will also be included, except for those pallets which the overlay cache expected
// to have.
func (c *LayeredPalletCache) LoadFSPallets(searchPattern string) ([]*FSPallet, error) {
	if c == nil {
		return nil, nil
	}

	loadedPallets, err := c.Overlay.LoadFSPallets(searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load pallets from overlay")
	}

	underlayPallets, err := c.Underlay.LoadFSPallets(searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load pallets from underlay")
	}
	for _, pallet := range underlayPallets {
		if c.Overlay.IncludesFSPallet(pallet.Path(), pallet.Version) {
			continue
		}
		loadedPallets = append(loadedPallets, pallet)
	}

	sort.Slice(loadedPallets, func(i, j int) bool {
		return ComparePallets(loadedPallets[i].Pallet, loadedPallets[j].Pallet) == core.CompareLT
	})
	return loadedPallets, nil
}

// LayeredPalletCache: FSPkgLoader

// LoadFSPkg loads the FSPkg with the specified path and version.
// The loaded FSPkg instance is fully initialized.
// If the overlay cache expects to have the package, it will attempt to load the package; otherwise,
// the underlay cache will attempt to load the package.
func (c *LayeredPalletCache) LoadFSPkg(pkgPath string, version string) (*core.FSPkg, error) {
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
func (c *LayeredPalletCache) LoadFSPkgs(searchPattern string) ([]*core.FSPkg, error) {
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
		if c.Overlay.IncludesFSPkg(pkg.Path(), pkg.FSPkgTree.Version) {
			continue
		}
		pkgs = append(pkgs, pkg)
	}

	sort.Slice(pkgs, func(i, j int) bool {
		return core.CompareFSPkgs(pkgs[i], pkgs[j]) == core.CompareLT
	})
	return pkgs, nil
}

// PalletOverrideCache

// NewPalletOverrideCache instantiates a new PalletOverrideCache with a given list of pallets, and a
// map associating pallet paths with lists of versions which the respective pallets will be
// associated with.
func NewPalletOverrideCache(
	overridePallets []*FSPallet, palletVersions map[string][]string,
) (*PalletOverrideCache, error) {
	c := &PalletOverrideCache{
		pallets:           make(map[string]*FSPallet),
		palletPaths:       make([]string, 0, len(overridePallets)),
		palletVersions:    make(map[string][]string),
		palletVersionSets: make(map[string]structures.Set[string]),
	}
	for _, pallet := range overridePallets {
		palletPath := pallet.Path()
		if _, ok := c.pallets[palletPath]; ok {
			return nil, errors.Errorf("pallet %s was provided multiple times", palletPath)
		}
		c.pallets[palletPath] = pallet
		c.palletPaths = append(c.palletPaths, palletPath)
		if palletVersions == nil {
			continue
		}

		c.palletVersions[palletPath] = append(
			c.palletVersions[palletPath], palletVersions[palletPath]...,
		)
		sort.Strings(c.palletVersions[palletPath])
		if _, ok := c.palletVersionSets[palletPath]; !ok {
			c.palletVersionSets[palletPath] = make(structures.Set[string])
		}
		for _, version := range palletVersions[palletPath] {
			c.palletVersionSets[palletPath].Add(version)
		}
	}
	sort.Strings(c.palletPaths)
	return c, nil
}

// SetVersions configures the cache to cover the specified versions of the specified pallet.
func (c *PalletOverrideCache) SetVersions(palletPath string, versions structures.Set[string]) {
	if _, ok := c.palletVersionSets[palletPath]; !ok {
		c.palletVersionSets[palletPath] = make(structures.Set[string])
	}
	sortedVersions := make([]string, 0, len(versions))
	for version := range versions {
		sortedVersions = append(sortedVersions, version)
		c.palletVersionSets[palletPath].Add(version)
	}
	sort.Strings(sortedVersions)
	c.palletVersions[palletPath] = sortedVersions
}

// PalletOverrideCache: OverlayCache

// IncludesFSPallet reports whether the PalletOverrideCache instance has a pallet with the
// specified path and version.
func (c *PalletOverrideCache) IncludesFSPallet(palletPath string, version string) bool {
	if c == nil {
		return false
	}
	if _, ok := c.pallets[palletPath]; !ok {
		return false
	}
	return c.palletVersionSets[palletPath].Has(version)
}

// LoadFSPallet loads the FSPallet with the specified path, if the version matches any of versions
// for the pallet in the cache.
// The loaded FSPallet instance is fully initialized.
func (c *PalletOverrideCache) LoadFSPallet(palletPath string, version string) (*FSPallet, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	pallet, ok := c.pallets[palletPath]
	if !ok {
		return nil, errors.Errorf("couldn't find a pallet with path %s", palletPath)
	}
	if !c.palletVersionSets[palletPath].Has(version) {
		return nil, errors.Errorf("found pallet %s, but not with version %s", palletPath, version)
	}
	return pallet, nil
}

// LoadFSPallets loads all FSPallets matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching pallets to search
// for.
// The loaded FSPallet instances are fully initialized.
func (c *PalletOverrideCache) LoadFSPallets(searchPattern string) ([]*FSPallet, error) {
	if c == nil {
		return nil, nil
	}

	loadedPallets := make(map[string]*FSPallet) // indexed by pallet cache path
	palletCachePaths := make([]string, 0)
	for _, palletPath := range c.palletPaths {
		pallet := c.pallets[palletPath]
		for _, version := range c.palletVersions[palletPath] {
			palletCachePath := fmt.Sprintf("%s@%s", palletPath, version)
			palletCachePaths = append(palletCachePaths, palletCachePath)
			loadedPallets[palletCachePath] = pallet
		}
	}

	matchingPalletCachePaths := make([]string, 0, len(palletCachePaths))
	for _, cachePath := range palletCachePaths {
		ok, err := doublestar.Match(searchPattern, cachePath)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't search for pallets using pattern %s", searchPattern)
		}
		if ok {
			matchingPalletCachePaths = append(matchingPalletCachePaths, cachePath)
		}
	}
	sort.Strings(matchingPalletCachePaths)

	matchingPallets := make([]*FSPallet, 0, len(matchingPalletCachePaths))
	for _, cachePath := range matchingPalletCachePaths {
		matchingPallets = append(matchingPallets, loadedPallets[cachePath])
	}
	return matchingPallets, nil
}

// IncludesFSPkg reports whether the PalletOverrideCache instance has a pallet with the specified
// version which covers the specified package path.
func (c *PalletOverrideCache) IncludesFSPkg(pkgPath string, version string) bool {
	if c == nil {
		return false
	}

	// Beyond a certain number of pallets, it's probably faster to just recurse down via the subdirs.
	// But we probably don't need to worry about this for now.
	for _, pallet := range c.pallets {
		if !ffs.CoversPath(pallet, pkgPath) {
			continue
		}
		return c.palletVersionSets[pallet.Path()].Has(version)
	}
	return false
}

// LoadFSPkg loads the FSPkg with the specified path, if the version matches any of versions for
// the package's pallet in the cache.
// The loaded FSPkg instance is fully initialized.
func (c *PalletOverrideCache) LoadFSPkg(pkgPath string, version string) (*core.FSPkg, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}

	// Beyond a certain number of pallets, it's probably faster to just recurse down via the subdirs.
	// But we probably don't need to worry about this for now.
	for _, pallet := range c.pallets {
		if !ffs.CoversPath(pallet, pkgPath) {
			continue
		}
		if !c.palletVersionSets[pallet.Path()].Has(version) {
			return nil, errors.Errorf(
				"found pallet %s providing package %s, but not at version %s",
				pallet.Path(), pkgPath, version,
			)
		}
		return pallet.LoadFSPkg(ffs.GetSubdirPath(pallet, pkgPath))
	}
	return nil, errors.Errorf("couldn't find a pallet providing package %s", pkgPath)
}

// LoadFSPkgs loads all FSPkgs matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching package directories
// to search for.
// The loaded FSPkg instances are fully initialized.
func (c *PalletOverrideCache) LoadFSPkgs(searchPattern string) ([]*core.FSPkg, error) {
	if c == nil {
		return nil, nil
	}

	pkgs := make(map[string]*core.FSPkg) // indexed by package cache path
	pkgCachePaths := make([]string, 0)
	for _, palletPath := range c.palletPaths {
		pallet := c.pallets[palletPath]
		loaded, err := pallet.LoadFSPkgs("**")
		if err != nil {
			return nil, errors.Errorf("couldn't list packages in pallet %s", pallet.Path())
		}
		for _, version := range c.palletVersions[palletPath] {
			for _, pkg := range loaded {
				pkgCachePath := path.Join(fmt.Sprintf("%s@%s", palletPath, version), pkg.Subdir)
				pkgCachePaths = append(pkgCachePaths, pkgCachePath)
				pkgs[pkgCachePath] = pkg
			}
		}
	}

	matchingPkgCachePaths := make([]string, 0, len(pkgCachePaths))
	for _, cachePath := range pkgCachePaths {
		ok, err := doublestar.Match(searchPattern, cachePath)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't search for packages using pattern %s", searchPattern)
		}
		if ok {
			matchingPkgCachePaths = append(matchingPkgCachePaths, cachePath)
		}
	}
	sort.Strings(matchingPkgCachePaths)

	matchingPkgs := make([]*core.FSPkg, 0, len(matchingPkgCachePaths))
	for _, cachePath := range matchingPkgCachePaths {
		matchingPkgs = append(matchingPkgs, pkgs[cachePath])
	}
	return matchingPkgs, nil
}
