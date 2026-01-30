package pallets

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ffs "github.com/forklift-run/forklift/exp/fs"
	"github.com/forklift-run/forklift/exp/structures"
)

const (
	// DeplsDirName is the directory in a pallet which contains deployment declarations.
	DeplsDirName = "deployments"
	// DeplsFileExt is the file extension for deployment declaration files.
	DeplDeclFileExt = ".deploy.yml"
)

// A Depl is a package deployment, a complete declaration of how a package is to be deployed on a
// Docker host.
type Depl struct {
	// Name is the name of the package deployment.
	Name string
	// Decl is the Forklift package deployment definition for the deployment.
	Decl DeplDecl
}

// A DeplDecl defines a package deployment.
type DeplDecl struct {
	// Package is the package path of the package to deploy.
	Package string `yaml:"package"`
	// Features is a list of features from the package which should be enabled in the deployment.
	Features FeatureFlags `yaml:"features,omitempty"`
	// Disabled represents whether the deployment should be ignored.
	Disabled bool `yaml:"disabled,omitempty"`
}

type FeatureFlags []string

// Depl

// FilterDeplsForEnabled filters a slice of Depls to only include those which are not disabled.
func FilterDeplsForEnabled(depls []Depl) []Depl {
	filtered := make([]Depl, 0, len(depls))
	for _, depl := range depls {
		if depl.Decl.Disabled {
			continue
		}
		filtered = append(filtered, depl)
	}
	return filtered
}

// loadDepl loads the Depl from a file path in the provided base filesystem, assuming the file path
// is the specified name of the deployment followed by the deployment declaration file extension.
func loadDepl(fsys ffs.PathedFS, name string) (depl Depl, err error) {
	depl.Name = name
	if depl.Decl, err = loadDeplDecl(fsys, name+DeplDeclFileExt); err != nil {
		return Depl{}, errors.Wrapf(err, "couldn't load deployment declaration")
	}
	return depl, nil
}

// loadDepls loads all package deployment declarations from the provided base filesystem matching
// the specified search pattern.
// The search pattern should not include the file extension for deployment declaration files - the
// file extension will be appended to the search pattern by LoadDepls.
func loadDepls(fsys ffs.PathedFS, searchPattern string) ([]Depl, error) {
	searchPattern += DeplDeclFileExt
	deplDeclFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for package deployment declarations matching %s/%s",
			fsys.Path(), searchPattern,
		)
	}

	depls := make([]Depl, 0, len(deplDeclFiles))
	for _, deplDeclFilePath := range deplDeclFiles {
		if !strings.HasSuffix(deplDeclFilePath, DeplDeclFileExt) {
			continue
		}

		deplName := strings.TrimSuffix(deplDeclFilePath, DeplDeclFileExt)
		depl, err := loadDepl(fsys, deplName)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't load package deployment declaration from %s", deplDeclFilePath,
			)
		}
		depls = append(depls, depl)
	}
	return depls, nil
}

// ResAttachmentSource returns the source path for resources under the Depl instance.
// The resulting slice is useful for constructing [res.Attached] instances.
func (d *Depl) ResAttachmentSource() []string {
	return []string{
		fmt.Sprintf("deployment %s", d.Name),
	}
}

// DeplDecl

// loadDeplDecl loads a DeplDecl from the specified file path in the provided base filesystem.
func loadDeplDecl(fsys ffs.PathedFS, filePath string) (DeplDecl, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return DeplDecl{}, errors.Wrapf(
			err, "couldn't read deployment declaration file %s/%s", fsys.Path(), filePath,
		)
	}
	declaration := DeplDecl{}
	if err = yaml.Unmarshal(bytes, &declaration); err != nil {
		return DeplDecl{}, errors.Wrap(err, "couldn't parse deployment declaration")
	}
	return declaration, nil
}

// FeatureFlags

// With returns a copy of f with any additional flags found in the provided featureFlags. Any
// elements of featureFlags which aren't among the allowed will added to both result and
// unrecognized.
func (f FeatureFlags) With(
	featureFlags FeatureFlags, allowed FeatureFlags,
) (result FeatureFlags, unrecognized FeatureFlags) {
	existing := make(structures.Set[string])
	existing.Add(f...)
	allowedSet := make(structures.Set[string])
	allowedSet.Add(allowed...)

	result = slices.Clone(f)
	unrecognized = make([]string, 0, len(featureFlags))
	for _, featureFlag := range featureFlags {
		if !allowedSet.Has(featureFlag) {
			unrecognized = append(unrecognized, featureFlag)
		}
		if existing.Has(featureFlag) {
			continue
		}
		result = append(result, featureFlag)
		existing.Add(featureFlag) // don't add duplicates to the preexisting list
	}
	return result, unrecognized
}

// Without returns a copy of f excluding any flags found in the provided featureFlags.
func (f FeatureFlags) Without(featureFlags FeatureFlags) (result FeatureFlags) {
	exclude := make(structures.Set[string])
	exclude.Add(featureFlags...)

	result = make([]string, 0, len(f))
	for _, featureFlag := range f {
		if exclude.Has(featureFlag) {
			continue
		}
		result = append(result, featureFlag)
	}
	return result
}
