package pallets

import (
	"io/fs"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ffs "github.com/forklift-run/forklift/exp/fs"
)

const (
	// ImportDeclFileExt is the file extension for import group files.
	ImportDeclFileExt = ".imports.yml"
)

// A ImportDecl defines a file import group.
type ImportDecl struct {
	// Description is a short description of the import group to be shown to users.
	Description string `yaml:"description,omitempty"`
	// Modifiers is a list of modifiers evaluated in the provided order to build up a set of files to
	// import.
	Modifiers []ImportModifier `yaml:"modifiers"`
	// Disabled represents whether the import should be ignored.
	Disabled bool `yaml:"disabled,omitempty"`
	// Deprecated is a deprecation notice which, if specified as a non-empty string, causes warnings
	// to be issued whenever the file import group is used via a feature flag.
	Deprecated string `yaml:"deprecated,omitempty"`
}

// ImportDecl

// loadImportDecl loads an ImportDecl from the specified file path in the provided base filesystem.
func loadImportDecl(fsys ffs.PathedFS, filePath string) (ImportDecl, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return ImportDecl{}, errors.Wrapf(
			err, "couldn't read import group file %s/%s", fsys.Path(), filePath,
		)
	}
	declaration := ImportDecl{}
	if err = yaml.Unmarshal(bytes, &declaration); err != nil {
		return ImportDecl{}, errors.Wrap(err, "couldn't parse import group")
	}

	return declaration.AddDefaults(), nil
}

func (d ImportDecl) AddDefaults() ImportDecl {
	updatedModifiers := make([]ImportModifier, 0, len(d.Modifiers))
	for _, modifier := range d.Modifiers {
		if modifier.Type == "" {
			modifier.Type = ImportModifierTypeAdd
		}
		if modifier.Target == "" {
			modifier.Target = "/"
		}
		if modifier.Source == "" {
			modifier.Source = modifier.Target
		}
		if len(modifier.OnlyMatchingAny) == 0 {
			modifier.OnlyMatchingAny = []string{""}
		}
		updatedModifiers = append(updatedModifiers, modifier)
	}
	d.Modifiers = updatedModifiers
	return d
}

func (d ImportDecl) RemoveDefaults() ImportDecl {
	// TODO: use this method when saving import definitions!
	updatedModifiers := make([]ImportModifier, 0, len(d.Modifiers))
	for _, modifier := range d.Modifiers {
		if modifier.Type == ImportModifierTypeAdd {
			modifier.Type = ""
		}
		if modifier.Target == "/" {
			modifier.Target = ""
		}
		if modifier.Source == modifier.Target {
			modifier.Source = ""
		}
		if len(modifier.OnlyMatchingAny) == 1 && modifier.OnlyMatchingAny[0] == "" {
			modifier.OnlyMatchingAny = nil
		}
		updatedModifiers = append(updatedModifiers, modifier)
	}
	d.Modifiers = updatedModifiers
	return d
}
