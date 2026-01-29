package pallets

import (
	"io/fs"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ffs "github.com/forklift-run/forklift/pkg/fs"
)

// PalletDeclFile is the name of the file defining each Forklift pallet.
const PalletDeclFile = "forklift-pallet.yml"

// A PalletDecl defines a Forklift pallet.
type PalletDecl struct {
	// ForkliftVersion indicates that the pallet was written assuming the semantics of a given version
	// of Forklift. The version must be a valid Forklift version, and it sets the minimum version of
	// Forklift required to use the pallet. The Forklift tool refuses to use pallets declaring newer
	// Forklift versions for any operations beyond printing information. The Forklift version of the
	// pallet must be greater than or equal to the Forklift version of every required Forklift pallet.
	ForkliftVersion string `yaml:"forklift-version"`
	// Pallet defines the basic metadata for the pallet.
	Pallet PalletSpec `yaml:"pallet,omitempty"`
}

// PalletSpec defines the basic metadata for a Forklift pallet.
type PalletSpec struct {
	// Path is the pallet path, which acts as the canonical name for the pallet. It should just be the
	// path of the VCS repository for the pallet.
	Path string `yaml:"path"`
	// Description is a short description of the pallet to be shown to users.
	Description string `yaml:"description"`
	// ReadmeFile is the name of a readme file to be shown to users.
	ReadmeFile string `yaml:"readme-file"`
}

// PalletDecl

// loadPalletDecl loads a PalletDecl from the specified file path in the provided base filesystem.
func loadPalletDecl(fsys ffs.PathedFS, filePath string) (PalletDecl, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return PalletDecl{}, errors.Wrapf(
			err, "couldn't read pallet config file %s/%s", fsys.Path(), filePath,
		)
	}
	config := PalletDecl{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return PalletDecl{}, errors.Wrap(err, "couldn't parse pallet config")
	}
	return config, nil
}

// Check looks for errors in the construction of the pallet configuration.
func (d PalletDecl) Check() (errs []error) {
	return errsWrap(d.Pallet.Check(), "invalid pallet spec")
}

// PalletSpec

// Check looks for errors in the construction of the pallet spec.
func (s PalletSpec) Check() (errs []error) {
	if s.Path == "" {
		errs = append(errs, errors.Errorf("pallet spec is missing `path` parameter"))
	}
	return errs
}
