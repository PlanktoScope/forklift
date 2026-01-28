package packaging

import (
	"fmt"
	"io/fs"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ffs "github.com/forklift-run/forklift/pkg/fs"
)

// PkgDeclFile is the name of the file defining each package.
const PkgDeclFile = "forklift-package.yml"

// A PkgDecl defines a package.
type PkgDecl struct {
	// Package defines the basic metadata for the package.
	Package PkgSpec `yaml:"package,omitempty"`
	// Host contains information about the Docker host independent of any deployment of the package.
	Host PkgHostSpec `yaml:"host,omitempty"`
	// Deployment contains information about any deployment of the package.
	Deployment PkgDeplSpec `yaml:"deployment,omitempty"`
	// Features contains optional features which can be enabled or disabled.
	Features map[string]PkgFeatureSpec `yaml:"features,omitempty"`
}

// PkgSpec defines the basic metadata for a package.
type PkgSpec struct {
	// Description is a short description of the package to be shown to users.
	Description string `yaml:"description"`
	// Maintainers is a list of people who maintain the package.
	Maintainers []PkgMaintainer `yaml:"maintainers,omitempty"`
	// License is an SPDX 2.1 license expression specifying the licensing terms of the software
	// provided by the package.
	License string `yaml:"license"`
	// LicenseFile is the name of a license file describing the licensing terms of the software
	// provided by the package.
	LicenseFile string `yaml:"license-file,omitempty"`
	// Sources is a list of URLs providing the source code of the software provided by the package.
	Sources []string `yaml:"sources,omitempty"`
}

// PkgMaintainer describes a maintainer of a package.
type PkgMaintainer struct {
	// Name is the maintainer's name.
	Name string `yaml:"name,omitempty"`
	// Email is an email address for contacting the maintainer.
	Email string `yaml:"email,omitempty"`
}

// PkgHostSpec contains information about the Docker host independent of any deployment of the
// package.
type PkgHostSpec struct {
	// Tags is a list of strings associated with the host.
	Tags []string `yaml:"tags,omitempty"`
	// Provides describes resources ambiently provided by the Docker host.
	Provides ProvidedRes `yaml:"provides,omitempty"`
}

// PkgDeplSpec contains information about any deployment of the package.
type PkgDeplSpec struct {
	// ComposeFiles is a list of the names of Docker Compose files specifying the Docker Compose
	// application which will be deployed as part of a package deployment.
	ComposeFiles []string `yaml:"compose-files,omitempty"`
	// Tags is a list of strings associated with the deployment.
	Tags []string `yaml:"tags,omitempty"`
	// Requires describes resource requirements which must be met for a deployment of the package to
	// succeed.
	Requires RequiredRes `yaml:"requires,omitempty"`
	// Provides describes resources provided by a successful deployment of the package.
	Provides ProvidedRes `yaml:"provides,omitempty"`
}

// PkgFeatureSpec defines an optional feature of the package.
type PkgFeatureSpec struct {
	// Description is a short description of the feature to be shown to users.
	Description string `yaml:"description"`
	// ComposeFiles is a list of the names of Docker Compose files specifying the Docker Compose
	// application which will be merged together with any other Compose files as part of a package
	// deployment which enables the feature.
	ComposeFiles []string `yaml:"compose-files,omitempty"`
	// Tags is a list of strings associated with the feature.
	Tags []string `yaml:"tags,omitempty"`
	// Provides describes resource requirements which must be met for a deployment of the package to
	// succeed, if the feature is enabled.
	Requires RequiredRes `yaml:"requires,omitempty"`
	// Provides describes resources provided by a successful deployment of the package, if the feature
	// is enabled.
	Provides ProvidedRes `yaml:"provides,omitempty"`
}

// PkgDecl

// LoadPkgDecl loads a PkgDecl from the specified file path in the provided base filesystem.
func LoadPkgDecl(fsys ffs.PathedFS, filePath string) (PkgDecl, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return PkgDecl{}, errors.Wrapf(
			err, "couldn't read package declaration file %s/%s", fsys.Path(), filePath,
		)
	}
	declaration := PkgDecl{}
	if err = yaml.Unmarshal(bytes, &declaration); err != nil {
		return PkgDecl{}, errors.Wrap(err, "couldn't parse package declaration")
	}

	return declaration.AddDefaults(), nil
}

// AddDefaults makes a copy with empty values replaced by default values.
func (d PkgDecl) AddDefaults() PkgDecl {
	d.Host = d.Host.AddDefaults()
	d.Deployment = d.Deployment.AddDefaults()
	updatedFeatures := make(map[string]PkgFeatureSpec)
	for name, feature := range d.Features {
		updatedFeatures[name] = feature.AddDefaults()
	}
	d.Features = updatedFeatures
	return d
}

// PkgHostSpec

// ResAttachmentSource returns the source path for resources under the PkgHostSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgHostSpec instance.
// The resulting slice is useful for constructing [res.Attached] instances.
func (s PkgHostSpec) ResAttachmentSource(parentSource []string) []string {
	return append(parentSource, "host specification")
}

// AddDefaults makes a copy with empty values replaced by default values.
func (s PkgHostSpec) AddDefaults() PkgHostSpec {
	s.Provides = s.Provides.AddDefaults()
	return s
}

// PkgDeplSpec

// ResAttachmentSource returns the source path for resources under the PkgDeplSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgDeplSpec instance.
// The resulting slice is useful for constructing [res.Attached] instances.
func (s PkgDeplSpec) ResAttachmentSource(parentSource []string) []string {
	return append(parentSource, "deployment specification")
}

// AddDefaults makes a copy with empty values replaced by default values.
func (s PkgDeplSpec) AddDefaults() PkgDeplSpec {
	s.Provides = s.Provides.AddDefaults()
	return s
}

// PkgFeatureSpec

// ResAttachmentSource returns the source path for resources under the PkgFeatureSpec instance,
// adding a string to the provided list of source elements which describes the source of the
// PkgFeatureSpec instance.
// The resulting slice is useful for constructing [res.Attached] instances.
func (s PkgFeatureSpec) ResAttachmentSource(parentSource []string, featureName string) []string {
	return append(parentSource, fmt.Sprintf("feature %s", featureName))
}

// AddDefaults makes a copy with empty values replaced by default values.
func (s PkgFeatureSpec) AddDefaults() PkgFeatureSpec {
	s.Provides = s.Provides.AddDefaults()
	return s
}
