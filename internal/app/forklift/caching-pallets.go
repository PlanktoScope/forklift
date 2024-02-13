package forklift

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/core"
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
func (c *FSPalletCache) LoadFSPallet(repoPath string, version string) (*FSPallet, error) {
	repo, err := LoadFSPallet(c.FS, fmt.Sprintf("%s@%s", repoPath, version))
	if err != nil {
		return nil, err
	}
	repo.Version = version
	return repo, nil
}

// LoadFSPallets loads all FSPallets from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching repo directories to
// search for.
// The loaded FSPallet instances are fully initialized.
func (c *FSPalletCache) LoadFSPallets(searchPattern string) ([]*FSPallet, error) {
	repos, err := LoadFSPallets(c.FS, searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load repos from cache")
	}

	// set the Version field of the repo based on its path in the cache
	for _, repo := range repos {
		var repoPath string
		var ok bool
		if repoPath, repo.Version, ok = strings.Cut(core.GetSubdirPath(c, repo.FS.Path()), "@"); !ok {
			return nil, errors.Wrapf(
				err, "couldn't parse path of cached repo configured at %s as repo_path@version",
				repo.FS.Path(),
			)
		}
		if repoPath != repo.Path() {
			return nil, errors.Errorf(
				"cached repo %s is in cache at %s@%s instead of %s@%s",
				repo.Path(), repoPath, repo.Version, repo.Path(), repo.Version,
			)
		}
	}

	return repos, nil
}

// LayeredPalletCache

// Path returns the path of the underlying cache.
func (c *LayeredPalletCache) Path() string {
	return c.Underlay.Path()
}

// LoadFSPallet loads the FSPallet with the specified path and version.
// The loaded FSPallet instance is fully initialized.
// If the overlay cache expects to have the repo, it will attempt to load the repo; otherwise,
// the underlay cache will attempt to load the repo.
func (c *LayeredPalletCache) LoadFSPallet(repoPath string, version string) (*FSPallet, error) {
	if c.Overlay.IncludesFSPallet(repoPath, version) {
		repo, err := c.Overlay.LoadFSPallet(repoPath, version)
		return repo, errors.Wrap(err, "couldn't load repo from overlay")
	}
	repo, err := c.Underlay.LoadFSPallet(repoPath, version)
	return repo, errors.Wrap(err, "couldn't load repo from underlay")
}

// LoadFSPallets loads all FSPallets from the cache matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching repo directories to
// search for.
// The loaded FSPallet instances are fully initialized.
// All matching repos from the overlay cache will be included; all matching repos from the
// underlay cache will also be included, except for those repos which the overlay cache expected
// to have.
func (c *LayeredPalletCache) LoadFSPallets(searchPattern string) ([]*FSPallet, error) {
	loadedPallets, err := c.Overlay.LoadFSPallets(searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load repos from overlay")
	}

	underlayPallets, err := c.Underlay.LoadFSPallets(searchPattern)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load repos from underlay")
	}
	for _, repo := range underlayPallets {
		if c.Overlay.IncludesFSPallet(repo.Path(), repo.Version) {
			continue
		}
		loadedPallets = append(loadedPallets, repo)
	}

	sort.Slice(loadedPallets, func(i, j int) bool {
		return ComparePallets(loadedPallets[i].Pallet, loadedPallets[j].Pallet) == core.CompareLT
	})
	return loadedPallets, nil
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
func (c *PalletOverrideCache) SetVersions(palletPath string, versions map[string]struct{}) {
	if _, ok := c.palletVersionSets[palletPath]; !ok {
		c.palletVersionSets[palletPath] = make(map[string]struct{})
	}
	sortedVersions := make([]string, 0, len(versions))
	for version := range versions {
		sortedVersions = append(sortedVersions, version)
		c.palletVersionSets[palletPath][version] = struct{}{}
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
	_, ok := c.palletVersionSets[palletPath][version]
	return ok
}

// LoadFSPallet loads the FSPallet with the specified path, if the version matches any of versions
// for the pallet in the cache.
// The loaded FSPallet instance is fully initialized.
func (c *PalletOverrideCache) LoadFSPallet(palletPath string, version string) (*FSPallet, error) {
	pallet, ok := c.pallets[palletPath]
	if !ok {
		return nil, errors.Errorf("couldn't find a pallet with path %s", palletPath)
	}
	if _, ok = c.palletVersionSets[palletPath][version]; !ok {
		return nil, errors.Errorf("found pallet %s, but not with version %s", palletPath, version)
	}
	return pallet, nil
}

// LoadFSPallets loads all FSPallets matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching pallets to search
// for.
// The loaded FSPallet instances are fully initialized.
func (c *PalletOverrideCache) LoadFSPallets(searchPattern string) ([]*FSPallet, error) {
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
