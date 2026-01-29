package pallets

import (
	"path"
	"strings"

	"github.com/pkg/errors"

	ffs "github.com/forklift-run/forklift/pkg/fs"
)

const (
	// FeaturesDirName is the directory in a pallet containing declarations of file import groups
	// which can be referenced by name in file import groups.
	FeaturesDirName = "features"
	// FeatureDeclFileExt is the file extension for import group files.
	FeatureDeclFileExt = ".feature.yml"
)

// FSPallet: Features

// GetFeaturesFS returns the [fs.FS] in the pallet which contains pallet feature flag declarations.
func (p *FSPallet) GetFeaturesFS() (ffs.PathedFS, error) {
	return p.FS.Sub(FeaturesDirName)
}

// LoadFeature loads the Import declared by the specified feature flag name. The feature name is
// assumed to be either a path relative to the root of the pallet's filesystem, beginning with a
// "/", or (if the provided pallet loader is non-nil) a fully-qualified path in the form
// "github.com/repo-owner/repo-name/feature-subdir-path"
// (e.g. "github.com/PlanktoScope/pallet-standard/features/all").
func (p *FSPallet) LoadFeature(name string, loader FSPalletLoader) (imp Import, err error) {
	featuresFS, err := p.GetFeaturesFS()
	if err != nil {
		return Import{}, errors.Wrap(err, "couldn't open directory for feature declarations in pallet")
	}
	if imp, err = loadImport(featuresFS, name, FeatureDeclFileExt); err != nil {
		reqsFS, err := p.GetPalletReqsFS()
		if err != nil {
			return Import{}, errors.Wrap(
				err, "couldn't open directory for pallet requirements from pallet",
			)
		}
		req, err := LoadFSPalletReqContaining(reqsFS, name)
		if err != nil {
			return Import{}, errors.Wrapf(
				err, "couldn't find pallet requirement declaration for feature %s", name,
			)
		}
		if loader == nil {
			return Import{}, errors.Errorf("no pallet loader provided for loading feature %s", name)
		}
		loaded, err := loadRequiredFSPallet(p, loader, req.RequiredPath)
		if err != nil {
			return Import{}, errors.Wrapf(
				err, "couldn't load pallet %s providing feature %s", req.RequiredPath, name,
			)
		}
		feature, err := loaded.LoadFeature(
			strings.TrimPrefix(name, path.Join(loaded.Path(), FeaturesDirName)+"/"), nil,
		)
		feature.Name = name
		return feature, errors.Wrapf(err, "couldn't load import group for feature %s", name)
	}
	return imp, nil
}

// LoadFeatures loads all Imports from the pallet matching the specified search pattern.
// The search pattern should be a [doublestar] pattern, such as `**`, matching the feature paths to
// search for (but excluding the file extension for feature declaration files).
func (p *FSPallet) LoadFeatures(searchPattern string) ([]Import, error) {
	featuresFS, err := p.GetFeaturesFS()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open directory for feature declarations in pallet")
	}
	return loadImports(featuresFS, searchPattern, FeatureDeclFileExt)
}
