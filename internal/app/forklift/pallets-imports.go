package forklift

import (
	"io/fs"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/pkg/core"
)

// ResolvedImport

// ResolveImport loads the package from the [FSPkgLoader] instance based on the requirements in the
// provided deployment and the package requirement loader.
func ResolveImport(
	pallet *FSPallet, palletLoader FSPalletLoader, imp Import,
) (resolved *ResolvedImport, err error) {
	resolved = &ResolvedImport{
		Import: imp,
	}
	palletReqsFS, err := pallet.GetPalletReqsFS()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open directory for pallet requirements from pallet")
	}
	palletReq, err := LoadFSPalletReqContaining(palletReqsFS, imp.Name)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't find pallet requirement declaration for import group %s", imp.Name,
		)
	}
	if resolved.Pallet, _, err = LoadRequiredFSPallet(
		pallet, palletLoader, palletReq.RequiredPath,
	); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load pallet %s to resolve from import group %s",
			palletReq.RequiredPath, imp.Name,
		)
	}
	return resolved, nil
}

// ResolveImports loads the packages from the [FSPkgLoader] instance based on the requirements in the
// provided deployments and the package requirement loader.
func ResolveImports(
	pallet *FSPallet, palletLoader FSPalletLoader, imps []Import,
) (resolved []*ResolvedImport, err error) {
	resolvedImports := make([]*ResolvedImport, 0, len(imps))
	for _, imp := range imps {
		resolved, err := ResolveImport(pallet, palletLoader, imp)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't resolve import group %s", imp.Name)
		}
		resolvedImports = append(resolvedImports, resolved)
	}

	return resolvedImports, nil
}

// Import

// FilterImportsForEnabled filters a slice of Imports to only include those which are not disabled.
func FilterImportsForEnabled(imps []Import) []Import {
	filtered := make([]Import, 0, len(imps))
	for _, imp := range imps {
		if imp.Def.Disabled {
			continue
		}
		filtered = append(filtered, imp)
	}
	return filtered
}

// loadImport loads the Import from a file path in the provided base filesystem, assuming the file path
// is the specified name of the import followed by the import group file extension.
func loadImport(fsys core.PathedFS, name string) (imp Import, err error) {
	imp.Name = name
	if imp.Def, err = loadImportDef(fsys, name+ImportDefFileExt); err != nil {
		return Import{}, errors.Wrapf(err, "couldn't load import group")
	}
	return imp, nil
}

// loadImports loads all file import groups from the provided base filesystem matching
// the specified search pattern.
// The search pattern should not include the file extension for import group files - the
// file extension will be appended to the search pattern by LoadImports.
func loadImports(fsys core.PathedFS, searchPattern string) ([]Import, error) {
	searchPattern += ImportDefFileExt
	impDefFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for import groups matching %s/%s", fsys.Path(), searchPattern,
		)
	}

	imps := make([]Import, 0, len(impDefFiles))
	for _, impDefFilePath := range impDefFiles {
		if !strings.HasSuffix(impDefFilePath, ImportDefFileExt) {
			continue
		}

		impName := strings.TrimSuffix(impDefFilePath, ImportDefFileExt)
		imp, err := loadImport(fsys, impName)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load import group from %s", impDefFilePath)
		}
		imps = append(imps, imp)
	}
	return imps, nil
}

// ImportDef

// loadImportDef loads an ImportDef from the specified file path in the provided base filesystem.
func loadImportDef(fsys core.PathedFS, filePath string) (ImportDef, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return ImportDef{}, errors.Wrapf(
			err, "couldn't read import group file %s/%s", fsys.Path(), filePath,
		)
	}
	declaration := ImportDef{}
	if err = yaml.Unmarshal(bytes, &declaration); err != nil {
		return ImportDef{}, errors.Wrap(err, "couldn't parse import group")
	}

	// Normalize empty values with defaults:
	for i, modifier := range declaration.Modifiers {
		if modifier.Target == "" {
			modifier.Target = "/"
		}
		if modifier.Source == "" {
			modifier.Source = modifier.Target
		}
		if len(modifier.OnlyMatchingAny) == 0 {
			modifier.OnlyMatchingAny = []string{""}
		}
		declaration.Modifiers[i] = modifier
	}

	return declaration, nil
}

// TODO: add a method to validate the import definition
