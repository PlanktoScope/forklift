package forklift

import (
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/pkg/core"
)

// ResolvedImport

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

// ResolveImport loads the pallet from the [FSPalletLoader] instance based on the requirements in
// the provided file import group and the pallet.
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

// Evaluate returns a list of target file paths and a mapping between target file paths and source
// file paths relative to the attached pallet's FS member. Directories are excluded from this
// mapping.
func (i *ResolvedImport) Evaluate() (map[string]string, error) {
	pathMappings := make(map[string]string) // target -> source
	for _, modifier := range i.Def.Modifiers {
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
			if err := applyAddFeatureModifier(modifier, i.Pallet, pathMappings); err != nil {
				return pathMappings, err
			}
		case ImportModifierTypeRemoveFeature:
			if err := applyRemoveFeatureModifier(modifier, i.Pallet, pathMappings); err != nil {
				return pathMappings, err
			}
		}
	}
	return pathMappings, nil
}

func applyAddModifier(
	modifier ImportModifier, pallet *FSPallet, pathMappings map[string]string,
) error {
	for _, matcher := range modifier.OnlyMatchingAny {
		sourcePattern := modifier.Source
		if matcher != "" {
			sourcePattern = path.Join(sourcePattern, matcher)
		}
		sourcePattern = strings.TrimPrefix(sourcePattern, "/")
		sourceFiles, err := globWithChildren(pallet.FS, sourcePattern, doublestar.WithFilesOnly())
		if err != nil {
			return err
		}
		for _, sourcePath := range sourceFiles {
			targetPath := path.Join("/", modifier.Target, strings.TrimPrefix(
				path.Join("/", sourcePath), path.Join("/", modifier.Source),
			))
			pathMappings[targetPath] = path.Join("/", sourcePath)
		}
	}
	return nil
}

func globWithChildren(
	fsys core.PathedFS, pattern string, opts ...doublestar.GlobOption,
) ([]string, error) {
	fileMatches, err := doublestar.Glob(fsys, pattern, opts...)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for files in %s matching pattern %s", fsys.Path(), pattern,
		)
	}
	pattern = path.Join(pattern, "**")
	childMatches, err := doublestar.Glob(fsys, pattern, opts...)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for files in %s matching pattern %s", fsys.Path(), pattern,
		)
	}
	return slices.Concat(fileMatches, childMatches), nil
}

func applyRemoveModifier(modifier ImportModifier, pathMappings map[string]string) error {
	for _, matcher := range modifier.OnlyMatchingAny {
		if matcher == "" {
			matcher = "**"
		} else {
			matcher = path.Join(matcher, "**")
		}
		targetPattern := path.Join(modifier.Target, matcher)
		for target := range pathMappings {
			matched, err := matchWithChildren(targetPattern, target)
			if err != nil {
				return err
			}
			if !matched {
				continue
			}
			delete(pathMappings, target)
		}
	}
	return nil
}

func matchWithChildren(pattern, name string) (bool, error) {
	baseMatches, err := doublestar.Match(pattern, name)
	if err != nil {
		return false, errors.Wrapf(
			err, "couldn't check whether %s matches pattern %s", name, pattern,
		)
	}
	if baseMatches {
		return true, nil
	}
	pattern = path.Join(pattern, "**")
	childMatches, err := doublestar.Match(pattern, name)
	if err != nil {
		return false, errors.Wrapf(
			err, "couldn't check whether %s matches pattern %s", name, pattern,
		)
	}
	return childMatches, nil
}

func applyAddFeatureModifier(
	modifier ImportModifier, pallet *FSPallet, pathMappings map[string]string,
) error {
	feature, err := pallet.LoadFeature(modifier.Source)
	if err != nil {
		return errors.Wrapf(err, "couldn't load feature %s", modifier.Source)
	}
	resolved := &ResolvedImport{
		Import: feature,
		Pallet: pallet,
	}
	featureMappings, err := resolved.Evaluate()
	if err != nil {
		return errors.Wrapf(
			err, "couldn't evaluate feature %s to determine file imports to add", modifier.Source,
		)
	}
	maps.Insert(pathMappings, maps.All(featureMappings))
	return nil
}

func applyRemoveFeatureModifier(
	modifier ImportModifier, pallet *FSPallet, pathMappings map[string]string,
) error {
	feature, err := pallet.LoadFeature(modifier.Source)
	if err != nil {
		return errors.Wrapf(err, "couldn't load feature %s", modifier.Source)
	}
	resolved := &ResolvedImport{
		Import: feature,
		Pallet: pallet,
	}
	featureMappings, err := resolved.Evaluate()
	if err != nil {
		return errors.Wrapf(
			err, "couldn't evaluate feature %s to determine file imports to add", modifier.Source,
		)
	}
	maps.DeleteFunc(pathMappings, func(target, source string) bool {
		_, ok := featureMappings[target]
		return ok
	})
	return nil
}

// CheckDeprecations returns a list of [error]s for any directly-referenced or
// transitively-referenced features which are deprecated.
func (i *ResolvedImport) CheckDeprecations() []error {
	if i.Def.Deprecated != "" {
		return []error{errors.New(i.Def.Deprecated)}
	}

	errs := make([]error, 0)
	for _, modifier := range i.Def.Modifiers {
		switch modifier.Type {
		default:
			continue
		case ImportModifierTypeAddFeature, ImportModifierTypeRemove:
			errs = append(errs, modifier.CheckDeprecations(i.Pallet)...)
		}
	}
	return errs
}

// TODO: add a method to check whether any import modifiers don't match any files, so that we can
// issue a warning when that happens!

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
func loadImport(fsys core.PathedFS, name, fileExt string) (imp Import, err error) {
	imp.Name = name
	if imp.Def, err = loadImportDef(fsys, name+fileExt); err != nil {
		return Import{}, errors.Wrapf(err, "couldn't load import group")
	}
	// TODO: if the import is deprecated, print a warning with the deprecation message
	return imp, nil
}

// loadImports loads all file import groups from the provided base filesystem matching
// the specified search pattern.
// The search pattern should not include the file extension for import group files - the
// file extension will be appended to the search pattern by LoadImports.
func loadImports(fsys core.PathedFS, searchPattern, fileExt string) ([]Import, error) {
	searchPattern += fileExt
	impDefFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for import groups matching %s/%s", fsys.Path(), searchPattern,
		)
	}

	imps := make([]Import, 0, len(impDefFiles))
	for _, impDefFilePath := range impDefFiles {
		if !strings.HasSuffix(impDefFilePath, fileExt) {
			continue
		}

		impName := strings.TrimSuffix(impDefFilePath, fileExt)
		imp, err := loadImport(fsys, impName, fileExt)
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

	return declaration.AddDefaults(), nil
}

func (d ImportDef) AddDefaults() ImportDef {
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

func (d ImportDef) RemoveDefaults() ImportDef {
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

// ImportModifier

// CheckDeprecations returns a list of [error]s for any directly-referenced or
// transitively-referenced features in the specified pallet which are deprecated.
func (m ImportModifier) CheckDeprecations(pallet *FSPallet) []error {
	feature, err := pallet.LoadFeature(m.Source)
	if err != nil {
		return []error{errors.Wrapf(err, "couldn't load referenced feature %s", m.Source)}
	}
	if deprecation := feature.Def.Deprecated; deprecation != "" {
		return []error{errors.Errorf("feature %s is deprecated: %s", feature.Name, deprecation)}
	}

	resolved := &ResolvedImport{
		Import: feature,
		Pallet: pallet,
	}
	deprecations := resolved.CheckDeprecations()
	wrapped := make([]error, 0, len(deprecations))
	for _, deprecation := range deprecations {
		wrapped = append(wrapped, errors.Wrapf(
			deprecation, "referenced by feature %s", feature.Name,
		))
	}
	return wrapped
}
