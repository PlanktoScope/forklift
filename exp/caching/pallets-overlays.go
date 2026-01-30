package caching

import (
	"fmt"
	"path"
	"slices"
	"sort"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	ffs "github.com/forklift-run/forklift/exp/fs"
	fpkg "github.com/forklift-run/forklift/exp/packaging"
	fplt "github.com/forklift-run/forklift/exp/pallets"
	"github.com/forklift-run/forklift/exp/structures"
)

// LayeredPalletCache is a [PathedPalletCache] implementation where selected pallets can be
// overridden by an [OverlayPalletCache], for loading pallets.
// The path of the LayeredPalletCache instance is just the path of the underlying cache.
type LayeredPalletCache struct {
	// Underlay is the underlying cache.
	Underlay PathedPalletCache
	// Overlay is the overlying cache which is used instead of the underlying cache for pallets
	// covered by the overlying cache.
	Overlay OverlayPalletCache
}

// PathedPalletCache is a pallet cache rooted at a single path.
type PathedPalletCache interface {
	ffs.Pather
	fplt.FSPalletLoader
	fplt.FSPkgLoader
}

// OverlayPalletCache is a pallet cache which can report whether it includes any particular pallet.
type OverlayPalletCache interface {
	fplt.FSPalletLoader
	fplt.FSPkgLoader
	// IncludesFSPallet reports whether the cache expects to have the specified pallet.
	// This result does not necessarily correspond to whether the cache actually has it.
	IncludesFSPallet(palletPath string, version string) bool
	// IncludesFSPkg reports whether the cache expects to have the specified package.
	// This result does not necessarily correspond to whether the cache actually has it.
	IncludesFSPkg(pkgPath string, version string) bool
}

// PalletOverrideCache is an [OverlayPalletCache] implementation containing a set of pallets which
// can be retrieved from the root of the cache. A pallet from the cache will be retrieved if it is
// stored in the cache with a matching version, regardless of whether the pallet's own version
// actually matches - in other words, pallets can be stored with fictional versions.
type PalletOverrideCache struct {
	// pallets is a map associating pallet paths to loaded pallets.
	// For each key-value pair, the key must be the path of the pallet which is the value of that
	// key-value pair.
	pallets map[string]*fplt.FSPallet
	// palletPaths is an alphabetically ordered list of the keys of pallets.
	palletPaths []string
	// palletVersions is a map associating pallet paths to pallet version strings.
	palletVersions map[string][]string
	// palletVersionSets is like palletVersions, but every value is a set of versions rather than a
	// list of versions.
	palletVersionSets map[string]structures.Set[string]
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
func (c *LayeredPalletCache) LoadFSPallet(pltPath string, version string) (*fplt.FSPallet, error) {
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
func (c *LayeredPalletCache) LoadFSPallets(searchPattern string) ([]*fplt.FSPallet, error) {
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

	slices.SortFunc(loadedPallets, func(a, b *fplt.FSPallet) int {
		return fplt.ComparePallets(a.Pallet, b.Pallet)
	})
	return loadedPallets, nil
}

// LayeredPalletCache: FSPkgLoader

// LoadFSPkg loads the FSPkg with the specified path and version.
// The loaded FSPkg instance is fully initialized.
// If the overlay cache expects to have the package, it will attempt to load the package; otherwise,
// the underlay cache will attempt to load the package.
func (c *LayeredPalletCache) LoadFSPkg(pkgPath string, version string) (*fpkg.FSPkg, error) {
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
func (c *LayeredPalletCache) LoadFSPkgs(searchPattern string) ([]*fpkg.FSPkg, error) {
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

	slices.SortFunc(pkgs, fpkg.CompareFSPkgs)
	return pkgs, nil
}

// PalletOverrideCache

// NewPalletOverrideCache instantiates a new PalletOverrideCache with a given list of pallets, and a
// map associating pallet paths with lists of versions which the respective pallets will be
// associated with.
func NewPalletOverrideCache(
	overridePallets []*fplt.FSPallet, palletVersions map[string][]string,
) (*PalletOverrideCache, error) {
	c := &PalletOverrideCache{
		pallets:           make(map[string]*fplt.FSPallet),
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

// PalletOverrideCache: OverlayPalletCache: FSPalletLoader

// LoadFSPallet loads the FSPallet with the specified path, if the version matches any of versions
// for the pallet in the cache.
// The loaded FSPallet instance is fully initialized.
func (c *PalletOverrideCache) LoadFSPallet(
	palletPath string,
	version string,
) (*fplt.FSPallet, error) {
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
func (c *PalletOverrideCache) LoadFSPallets(searchPattern string) ([]*fplt.FSPallet, error) {
	if c == nil {
		return nil, nil
	}

	loadedPallets := make(map[string]*fplt.FSPallet) // indexed by pallet cache path
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

	matchingPallets := make([]*fplt.FSPallet, 0, len(matchingPalletCachePaths))
	for _, cachePath := range matchingPalletCachePaths {
		matchingPallets = append(matchingPallets, loadedPallets[cachePath])
	}
	return matchingPallets, nil
}

// PalletOverrideCache: OverlayPalletCache: FSPkgLoader

// LoadFSPkg loads the FSPkg with the specified path, if the version matches any of versions for
// the package's pallet in the cache.
// The loaded FSPkg instance is fully initialized.
func (c *PalletOverrideCache) LoadFSPkg(pkgPath string, version string) (*fpkg.FSPkg, error) {
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
func (c *PalletOverrideCache) LoadFSPkgs(searchPattern string) ([]*fpkg.FSPkg, error) {
	if c == nil {
		return nil, nil
	}

	pkgs := make(map[string]*fpkg.FSPkg) // indexed by package cache path
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

	matchingPkgs := make([]*fpkg.FSPkg, 0, len(matchingPkgCachePaths))
	for _, cachePath := range matchingPkgCachePaths {
		matchingPkgs = append(matchingPkgs, pkgs[cachePath])
	}
	return matchingPkgs, nil
}
