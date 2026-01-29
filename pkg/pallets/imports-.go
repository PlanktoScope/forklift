package pallets

import (
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	ffs "github.com/forklift-run/forklift/pkg/fs"
)

// An Import is an import group, a declaration of a group of files to import from a required pallet.
type Import struct {
	// Name is the name of the file import group.
	Name string
	// Decl is the file import definition for the file import group.
	Decl ImportDecl
}

// FilterImportsForEnabled filters a slice of Imports to only include those which are not disabled.
func FilterImportsForEnabled(imps []Import) []Import {
	filtered := make([]Import, 0, len(imps))
	for _, imp := range imps {
		if imp.Decl.Disabled {
			continue
		}
		filtered = append(filtered, imp)
	}
	return filtered
}

// loadImport loads the Import from a file path in the provided base filesystem, assuming the file path
// is the specified name of the import followed by the import group file extension.
func loadImport(fsys ffs.PathedFS, name, fileExt string) (imp Import, err error) {
	imp.Name = name
	if imp.Decl, err = loadImportDecl(fsys, name+fileExt); err != nil {
		return Import{}, errors.Wrapf(err, "couldn't load import group")
	}
	// TODO: if the import is deprecated, print a warning with the deprecation message
	return imp, nil
}

// loadImports loads all file import groups from the provided base filesystem matching
// the specified search pattern.
// The search pattern should not include the file extension for import group files - the
// file extension will be appended to the search pattern by LoadImports.
func loadImports(fsys ffs.PathedFS, searchPattern, fileExt string) ([]Import, error) {
	searchPattern += fileExt
	impDeclFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for import groups matching %s/%s", fsys.Path(), searchPattern,
		)
	}

	imps := make([]Import, 0, len(impDeclFiles))
	for _, impDeclFilePath := range impDeclFiles {
		if !strings.HasSuffix(impDeclFilePath, fileExt) {
			continue
		}

		impName := strings.TrimSuffix(impDeclFilePath, fileExt)
		imp, err := loadImport(fsys, impName, fileExt)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load import group from %s", impDeclFilePath)
		}
		imps = append(imps, imp)
	}
	return imps, nil
}
