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

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// FSPalletCache

// Exists checks whether the cache actually exists on the OS's filesystem.
func (c *FSPalletCache) Exists() bool {
	return Exists(filepath.FromSlash(c.FS.Path()))
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
func (c *FSPalletCache) LoadFSPallet(palletPath string, version string) (*pallets.FSPallet, error) {
	vcsRepoPath, _, err := pallets.SplitRepoPathSubdir(palletPath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't parse pallet path %s", palletPath)
	}
	// The pallet subdirectory path in the pallet path (under the VCS repo path) might not match the
	// filesystem directory path with the forklift-pallet.yml file, so we must check every
	// forklift-pallet.yml file to find the actual pallet path
	searchPattern := fmt.Sprintf("%s@%s/**", vcsRepoPath, version)
	loadedPallets, err := c.LoadFSPallets(searchPattern)
	if err != nil {
		return nil, err
	}

	candidatePallets := make([]*pallets.FSPallet, 0)
	for _, pallet := range loadedPallets {
		if pallet.Path() != palletPath {
			continue
		}

		if len(candidatePallets) > 0 {
			return nil, errors.Errorf(
				"version %s of pallet %s was found in multiple different locations: %s, %s",
				version, palletPath, candidatePallets[0].FS.Path(), pallet.FS.Path(),
			)
		}
		candidatePallets = append(candidatePallets, pallet)
	}
	if len(candidatePallets) == 0 {
		return nil, errors.Errorf("no cached pallets were found matching %s@%s", palletPath, version)
	}
	return candidatePallets[0], nil
}

// LoadFSPallets loads all FSPallets from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching pallet directories to
// search for.
// The loaded FSPallet instances are fully initialized.
func (c *FSPalletCache) LoadFSPallets(searchPattern string) ([]*pallets.FSPallet, error) {
	pallets, err := pallets.LoadFSPallets(c.FS, searchPattern, c.processLoadedFSPallet)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load pallets from cache")
	}

	return pallets, nil
}

// processLoadedFSPallet sets the Version field of the pallet based on its path in the cache.
func (c *FSPalletCache) processLoadedFSPallet(pallet *pallets.FSPallet) (err error) {
	var vcsRepoPath string
	if vcsRepoPath, pallet.Version, err = getRepoPathVersion(
		pallets.GetSubdirPath(c, pallet.FS.Path()),
	); err != nil {
		return errors.Wrapf(
			err, "couldn't parse path of cached pallet configured at %s", pallet.FS.Path(),
		)
	}
	if vcsRepoPath != pallet.VCSRepoPath {
		return errors.Errorf(
			"cached pallet %s is in cache at %s@%s instead of %s@%s",
			pallet.Path(), vcsRepoPath, pallet.Version, pallet.VCSRepoPath, pallet.Version,
		)
	}
	return nil
}

// getRepoPathVersion splits paths of form github.com/user-name/git-repo-name/etc@version into
// github.com/user-name/git-repo-name and version.
func getRepoPathVersion(palletPath string) (vcsRepoPath, version string, err error) {
	const sep = "/"
	pathParts := strings.Split(palletPath, sep)
	if pathParts[0] != "github.com" {
		return "", "", errors.Errorf(
			"pallet path %s does not begin with github.com, and handling of non-Github repositories is "+
				"not yet implemented",
			palletPath,
		)
	}
	vcsRepoName, version, ok := strings.Cut(pathParts[2], "@")
	if !ok {
		return "", "", errors.Errorf(
			"Couldn't parse Github repository name %s as name@version", pathParts[2],
		)
	}
	vcsRepoPath = strings.Join([]string{pathParts[0], pathParts[1], vcsRepoName}, sep)
	return vcsRepoPath, version, nil
}

// FSPalletCache: FSPkgLoader

// LoadFSPkg loads the FSPkg with the specified path and version.
// The loaded FSPkg instance is fully initialized.
func (c *FSPalletCache) LoadFSPkg(pkgPath string, version string) (*pallets.FSPkg, error) {
	vcsRepoPath, _, err := pallets.SplitRepoPathSubdir(pkgPath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't parse path of package %s", pkgPath)
	}
	pkgInnermostDir := path.Base(pkgPath)
	// The package subdirectory path in the package path (under the VCS repo path) might not match the
	// filesystem directory path with the forklift-package.yml file, so we must check every
	// directory whose name matches the last part of the package path to look for the package
	searchPattern := fmt.Sprintf("%s@%s/**/%s", vcsRepoPath, version, pkgInnermostDir)
	pkgs, err := c.LoadFSPkgs(searchPattern)
	if err != nil {
		return nil, err
	}

	candidatePkgs := make([]*pallets.FSPkg, 0)
	for _, pkg := range pkgs {
		if pkg.Path() != pkgPath {
			continue
		}

		if len(candidatePkgs) > 0 {
			return nil, errors.Errorf(
				"version %s of package %s was found in multiple different locations: %s, %s",
				version, pkgPath, candidatePkgs[0].FS.Path(), pkg.FS.Path(),
			)
		}
		candidatePkgs = append(candidatePkgs, pkg)
	}
	if len(candidatePkgs) == 0 {
		return nil, errors.Errorf("no cached packages were found matching %s@%s", pkgPath, version)
	}
	return candidatePkgs[0], nil
}

// LoadFSPkgs loads all FSPkgs from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching package directories
// to search for.
// The loaded FSPkg instances are fully initialized.
func (c *FSPalletCache) LoadFSPkgs(searchPattern string) ([]*pallets.FSPkg, error) {
	pkgs, err := pallets.LoadFSPkgs(c.FS, searchPattern)
	if err != nil {
		return nil, err
	}

	pkgMap := make(map[string]*pallets.FSPkg)
	for _, pkg := range pkgs {
		pallet, err := c.loadFSPalletContaining(pallets.GetSubdirPath(c, pkg.FS.Path()))
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't find the cached pallet providing the package at %s", pkg.FS.Path(),
			)
		}
		if err = pkg.AttachFSPallet(pallet); err != nil {
			return nil, errors.Wrap(err, "couldn't attach pallet to package")
		}
		pkgCachePath := getPkgCachePath(pkg.Pkg)
		if prevPkg, ok := pkgMap[pkgCachePath]; ok {
			if prevPkg.Pallet.FromSameVCSRepo(pkg.Pallet.Pallet) && prevPkg.FS.Path() != pkg.FS.Path() {
				return nil, errors.Errorf(
					"the same version of package %s was found in multiple different locations: %s, %s",
					pkg.Path(), prevPkg.FS.Path(), pkg.FS.Path(),
				)
			}
		}
		pkgMap[pkgCachePath] = pkg
	}

	return pkgs, nil
}

func getPkgCachePath(pkg pallets.Pkg) string {
	return fmt.Sprintf("%s@%s/%s", pkg.Pallet.Def.Pallet.Path, pkg.Pallet.Version, pkg.Subdir)
}

// loadFSPalletContaining finds and loads the FSPallet which contains the provided subdirectory
// path.
func (c *FSPalletCache) loadFSPalletContaining(
	subdirPath string,
) (pallet *pallets.FSPallet, err error) {
	if pallet, err = pallets.LoadFSPalletContaining(c.FS, subdirPath); err != nil {
		return nil, errors.Wrapf(err, "couldn't find any pallet containing %s", subdirPath)
	}
	if err = c.processLoadedFSPallet(pallet); err != nil {
		return nil, err
	}
	return pallet, nil
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
func (c *LayeredPalletCache) LoadFSPallet(
	palletPath string, version string,
) (*pallets.FSPallet, error) {
	if c.Overlay.IncludesFSPallet(palletPath, version) {
		pallet, err := c.Overlay.LoadFSPallet(palletPath, version)
		return pallet, errors.Wrap(err, "couldn't load pallet from overlay")
	}
	pallet, err := c.Underlay.LoadFSPallet(palletPath, version)
	return pallet, errors.Wrap(err, "couldn't load pallet from underlay")
}

// LoadFSPallets loads all FSPallets from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching pallet directories to
// search for.
// The loaded FSPallet instances are fully initialized.
// All matching pallets from the overlay cache will be included; all matching pallets from the
// underlay cache will also be included, except for those pallets which the overlay cache expected
// to have.
func (c *LayeredPalletCache) LoadFSPallets(searchPattern string) ([]*pallets.FSPallet, error) {
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
		return pallets.ComparePallets(
			loadedPallets[i].Pallet, loadedPallets[j].Pallet,
		) == pallets.CompareLT
	})
	return loadedPallets, nil
}

// LoadFSPkg loads the FSPkg with the specified path and version.
// The loaded FSPkg instance is fully initialized.
// If the overlay cache expects to have the package, it will attempt to load the package; otherwise,
// the underlay cache will attempt to load the package.
func (c *LayeredPalletCache) LoadFSPkg(pkgPath string, version string) (*pallets.FSPkg, error) {
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
func (c *LayeredPalletCache) LoadFSPkgs(searchPattern string) ([]*pallets.FSPkg, error) {
	pkgs, err := c.Overlay.LoadFSPkgs(searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load packages from overlay")
	}

	underlayPkgs, err := c.Underlay.LoadFSPkgs(searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load packages from underlay")
	}
	for _, pkg := range underlayPkgs {
		if c.Overlay.IncludesFSPkg(pkg.Path(), pkg.Pallet.Version) {
			continue
		}
		pkgs = append(pkgs, pkg)
	}

	sort.Slice(pkgs, func(i, j int) bool {
		return pallets.ComparePkgs(pkgs[i].Pkg, pkgs[j].Pkg) == pallets.CompareLT
	})
	return pkgs, nil
}

// PalletOverrideCache

// NewPalletOverrideCache instantiates a new PalletOverrideCache with a given list of pallets, and a
// map associating pallet paths with lists of versions which the respective pallets will be
// associated with.
func NewPalletOverrideCache(
	overridePallets []*pallets.FSPallet, palletVersions map[string][]string,
) (*PalletOverrideCache, error) {
	c := &PalletOverrideCache{
		pallets:           make(map[string]*pallets.FSPallet),
		palletPaths:       make([]string, 0, len(overridePallets)),
		palletVersions:    make(map[string][]string),
		palletVersionSets: make(map[string]map[string]struct{}),
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
			c.palletVersionSets[palletPath] = make(map[string]struct{})
		}
		for _, version := range palletVersions[palletPath] {
			c.palletVersionSets[palletPath][version] = struct{}{}
		}
	}
	sort.Strings(c.palletPaths)
	return c, nil
}

// SetVersions configures the cache to cover the specified versions of the specified pallet.
func (f *PalletOverrideCache) SetVersions(palletPath string, versions map[string]struct{}) {
	if _, ok := f.palletVersionSets[palletPath]; !ok {
		f.palletVersionSets[palletPath] = make(map[string]struct{})
	}
	sortedVersions := make([]string, 0, len(versions))
	for version := range versions {
		sortedVersions = append(sortedVersions, version)
		f.palletVersionSets[palletPath][version] = struct{}{}
	}
	sort.Strings(sortedVersions)
	f.palletVersions[palletPath] = sortedVersions
}

// PalletOverrideCache: OverlayCache

// IncludesFSPallet reports whether the PalletOverrideCache instance has a pallet with the
// specified path and version.
func (f *PalletOverrideCache) IncludesFSPallet(palletPath string, version string) bool {
	_, ok := f.pallets[palletPath]
	if !ok {
		return false
	}
	_, ok = f.palletVersionSets[palletPath][version]
	return ok
}

// LoadFSPallet loads the FSPallet with the specified path, if the version matches any of versions
// for the pallet in the cache.
// The loaded FSPallet instance is fully initialized.
func (f *PalletOverrideCache) LoadFSPallet(
	palletPath string, version string,
) (*pallets.FSPallet, error) {
	pallet, ok := f.pallets[palletPath]
	if !ok {
		return nil, errors.Errorf("couldn't find a pallet with path %s", palletPath)
	}
	_, ok = f.palletVersionSets[palletPath][version]
	if !ok {
		return nil, errors.Errorf("found pallet %s, but not with version %s", palletPath, version)
	}
	return pallet, nil
}

// LoadFSPallets loads all FSPallets matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching pallets to search
// for.
// The loaded FSPallet instances are fully initialized.
func (f *PalletOverrideCache) LoadFSPallets(searchPattern string) ([]*pallets.FSPallet, error) {
	loadedPallets := make(map[string]*pallets.FSPallet) // indexed by pallet cache path
	palletCachePaths := make([]string, 0)
	for _, palletPath := range f.palletPaths {
		pallet := f.pallets[palletPath]
		for _, version := range f.palletVersions[palletPath] {
			palletCachePath, err := getPalletCachePath(palletPath, version)
			if err != nil {
				return nil, err
			}
			palletCachePaths = append(palletCachePaths, palletCachePath)
			loadedPallets[palletCachePath] = pallet
		}
	}

	matchingPalletCachePaths := make([]string, 0, len(palletCachePaths))
	for _, cachePath := range palletCachePaths {
		ok, err := doublestar.Match(searchPattern, cachePath)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't search for pallets using pattern %s", searchPattern,
			)
		}
		if ok {
			matchingPalletCachePaths = append(matchingPalletCachePaths, cachePath)
		}
	}
	sort.Strings(matchingPalletCachePaths)

	matchingPallets := make([]*pallets.FSPallet, 0, len(matchingPalletCachePaths))
	for _, cachePath := range matchingPalletCachePaths {
		matchingPallets = append(matchingPallets, loadedPallets[cachePath])
	}
	return matchingPallets, nil
}

func getPalletCachePath(palletPath, version string) (string, error) {
	vcsRepoPath, palletSubdir, err := pallets.SplitRepoPathSubdir(palletPath)
	if err != nil {
		return "", errors.Wrapf(err, "couldn't parse pallet path %s", palletPath)
	}
	return fmt.Sprintf("%s@%s/%s", vcsRepoPath, version, palletSubdir), nil
}

// IncludesFSPkg reports whether the PalletOverrideCache instance has a pallet with the specified
// version which covers the specified package path.
func (f *PalletOverrideCache) IncludesFSPkg(pkgPath string, version string) bool {
	// Beyond a certain number of pallets, it's probably faster to just recurse down via the subdirs.
	// But we probably don't need to worry about this for now.
	for _, pallet := range f.pallets {
		if !pallets.CoversPath(pallet, pkgPath) {
			continue
		}
		_, ok := f.palletVersionSets[pallet.Path()][version]
		return ok
	}
	return false
}

// LoadFSPkg loads the FSPkg with the specified path, if the version matches any of versions for
// the package's pallet in the cache.
// The loaded FSPkg instance is fully initialized.
func (f *PalletOverrideCache) LoadFSPkg(pkgPath string, version string) (*pallets.FSPkg, error) {
	// Beyond a certain number of pallets, it's probably faster to just recurse down via the subdirs.
	// But we probably don't need to worry about this for now.
	for _, pallet := range f.pallets {
		if !pallets.CoversPath(pallet, pkgPath) {
			continue
		}
		_, ok := f.palletVersionSets[pallet.Path()][version]
		if !ok {
			return nil, errors.Errorf(
				"found pallet %s providing package %s, but not at version %s",
				pallet.Path(), pkgPath, version,
			)
		}
		return pallet.LoadFSPkg(pallets.GetSubdirPath(pallet, pkgPath))
	}
	return nil, errors.Errorf("couldn't find a pallet providing package %s", pkgPath)
}

// LoadFSPkgs loads all FSPkgs matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching package directories
// to search for.
// The loaded FSPkg instances are fully initialized.
func (f *PalletOverrideCache) LoadFSPkgs(searchPattern string) ([]*pallets.FSPkg, error) {
	pkgs := make(map[string]*pallets.FSPkg) // indexed by package cache path
	pkgCachePaths := make([]string, 0)
	for _, palletPath := range f.palletPaths {
		pallet := f.pallets[palletPath]
		loaded, err := pallet.LoadFSPkgs("**")
		if err != nil {
			return nil, errors.Errorf("couldn't list packages in pallet %s", pallet.Path())
		}
		for _, version := range f.palletVersions[palletPath] {
			for _, pkg := range loaded {
				palletCachePath, err := getPalletCachePath(palletPath, version)
				if err != nil {
					return nil, err
				}
				pkgCachePath := path.Join(palletCachePath, pkg.Subdir)
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

	matchingPkgs := make([]*pallets.FSPkg, 0, len(matchingPkgCachePaths))
	for _, cachePath := range matchingPkgCachePaths {
		matchingPkgs = append(matchingPkgs, pkgs[cachePath])
	}
	return matchingPkgs, nil
}
