package pallets

import (
	"io/fs"
	"path"

	"github.com/pkg/errors"
)

// A ResolvedImport is a file import group with a loaded pallet.
type ResolvedImport struct {
	// Import is the declared file import group.
	Import
	// Pallet is the pallet which files will be imported from
	Pallet *FSPallet
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

// ResolveImport loads the import from a pallet loaded from the [FSPalletLoader] instance based on
// the requirements in the provided file import group and the pallet.
func ResolveImport(
	pallet *FSPallet, palletLoader FSPalletLoader, imp Import,
) (resolved *ResolvedImport, err error) {
	resolved = &ResolvedImport{
		Import: imp,
	}
	if _, err = fs.Stat(pallet.FS, path.Join(FeaturesDirName, imp.Name+FeatureDeclFileExt)); err == nil {
		// Attach the import to the current pallet
		resolved.Pallet = pallet
		return resolved, nil
	}

	// Attach the import to a required pallet
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
	if resolved.Pallet, err = loadRequiredFSPallet(
		pallet, palletLoader, palletReq.RequiredPath,
	); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load pallet %s to resolve from import group %s",
			palletReq.RequiredPath, imp.Name,
		)
	}
	return resolved, nil
}

// Evaluate returns a list of target file paths and a mapping between target file paths and source
// file paths relative to the attached pallet's FS member. Directories are excluded from this
// mapping.
func (i *ResolvedImport) Evaluate(loader FSPalletLoader) (map[string]string, error) {
	pathMappings := make(map[string]string) // target -> source
	for _, modifier := range i.Decl.Modifiers {
		switch modifier.Type {
		default:
			return pathMappings, errors.Errorf("unknown modifier type: %s", modifier.Type)
		case ImportModifierTypeAdd:
			if err := applyAddModifier(modifier, i.Pallet, pathMappings); err != nil {
				return pathMappings, err
			}
		case ImportModifierTypeRemove:
			if err := applyRemoveModifier(modifier, pathMappings); err != nil {
				return pathMappings, err
			}
		case ImportModifierTypeAddFeature:
			if err := applyAddFeatureModifier(modifier, i.Pallet, pathMappings, loader); err != nil {
				return pathMappings, err
			}
		case ImportModifierTypeRemoveFeature:
			if err := applyRemoveFeatureModifier(modifier, i.Pallet, pathMappings, loader); err != nil {
				return pathMappings, err
			}
		}
	}
	return pathMappings, nil
}

// CheckDeprecations returns a list of [error]s for any directly-referenced or
// transitively-referenced features which are deprecated.
func (i *ResolvedImport) CheckDeprecations(
	loader FSPalletLoader,
) (deprecations []error, err error) {
	if i.Decl.Deprecated != "" {
		return []error{errors.New(i.Decl.Deprecated)}, nil
	}

	for _, modifier := range i.Decl.Modifiers {
		switch modifier.Type {
		default:
			continue
		case ImportModifierTypeAddFeature, ImportModifierTypeRemove:
			checked, err := modifier.CheckDeprecations(i.Pallet, loader)
			if err != nil {
				return deprecations, err
			}
			deprecations = append(deprecations, checked...)
		}
	}
	return deprecations, nil
}

// TODO: add a method to check whether any import modifiers don't match any files, so that we can
// issue a warning when that happens!
