package pallets

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// FSPkg

// LoadFSPkg loads a FSPkg from the specified directory path in the provided base filesystem.
// The base path should correspond to the location of the base filesystem. In the loaded FSPkg's
// embedded [Pkg], the Pallet package path (not to be confused with the path in the provided base
// filesystem) is not initialized, nor is the package subdirectory initialized.
func LoadFSPkg(baseFS fs.FS, baseFSPath, pkgFSPath string) (p FSPkg, err error) {
	p = FSPkg{
		FSPath: filepath.Join(baseFSPath, pkgFSPath),
	}
	if p.FS, err = fs.Sub(baseFS, pkgFSPath); err != nil {
		return FSPkg{}, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", pkgFSPath, baseFSPath,
		)
	}
	if p.Pkg.Config, err = LoadPkgConfig(p.FS, p.FSPath, PkgSpecFile); err != nil {
		return FSPkg{}, errors.Wrapf(err, "couldn't load package config")
	}
	return p, nil
}

// PkgConfig

// LoadPkgConfig loads a PkgConfig from the specified file path in the provided base filesystem.
// The base path should correspond to the location of the base filesystem.
func LoadPkgConfig(baseFS fs.FS, baseFSPath, filePath string) (PkgConfig, error) {
	bytes, err := fs.ReadFile(baseFS, filePath)
	if err != nil {
		return PkgConfig{}, errors.Wrapf(
			err, "couldn't read package config file %s/%s", baseFSPath, filePath,
		)
	}
	config := PkgConfig{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return PkgConfig{}, errors.Wrap(err, "couldn't parse package config")
	}
	return config, nil
}

// PkgHostSpec

// ResourceAttachmentSource returns the source path for resources under the PkgHostSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgHostSpec instance.
func (s PkgHostSpec) ResourceAttachmentSource(parentSource []string) []string {
	return append(parentSource, "host specification")
}

// PkgDeplSpec

// ResourceAttachmentSource returns the source path for resources under the PkgDeplSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgDeplSpec instance.
func (s PkgDeplSpec) ResourceAttachmentSource(parentSource []string) []string {
	return append(parentSource, "deployment specification")
}

// DefinesStack determines whether the PkgDeplSpec instance defines a Docker stack to be deployed.
func (s PkgDeplSpec) DefinesStack() bool {
	return s.DefinitionFile != ""
}

// PkgFeatureSpec

// ResourceAttachmentSource returns the source path for resources under the PkgFeatureSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgFeatureSpec instance.
func (s PkgFeatureSpec) ResourceAttachmentSource(parentSource []string, featureName string) []string {
	return append(parentSource, fmt.Sprintf("feature %s", featureName))
}
