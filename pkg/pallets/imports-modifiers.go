package forklift

import (
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	"github.com/pkg/errors"
)

// An ImportModifier defines an operation for transforming a set of files for importing into a
// different set of files for importing.
type ImportModifier struct {
	// Description is a short description of the import modifier to be shown to users.
	Description string `yaml:"description,omitempty"`
	// Type is either `add` (for adding one or more files to the set of files to import), `remove`
	// (for removing one or more files from the set of files to import), `add-feature` (for adding
	// files specified by a feature flag to the set of files to import), or `remove-feature` (for
	// removing one or more files specified by a feature flag from the set of files to import).
	Type string `yaml:"type,omitempty"`
	// Source is the path in the required pallet of the file/directory to be imported, for an `add`
	// modifier; or the name of a feature flag, for an `add-feature` or `remove-feature` modifier. If
	// omitted, the source path will be inferred from the Target path.
	Source string `yaml:"source,omitempty"`
	// Target is the path which the file/directory will be imported as, for an `add` modifier; or the
	// path of the file/directory which will be removed from the set of files to import, for a
	// `remove` modifier.
	Target string `yaml:"target,omitempty"`
	// OnlyMatchingAny is, if the source is a directory, a list of glob patterns (relative to the
	// source path) of files which will be added/removed (depending on modifier type). Any file which
	// matches none of patterns provided in this field will be ignored for the add/remove modifier. If
	// omitted, no files in the source directory will be ignored.
	OnlyMatchingAny []string `yaml:"only-matching-any,omitempty"`
}

const (
	ImportModifierTypeAdd           = "add"
	ImportModifierTypeRemove        = "remove"
	ImportModifierTypeAddFeature    = "add-feature"
	ImportModifierTypeRemoveFeature = "remove-feature"
)

// CheckDeprecations returns a list of [error]s for any directly-referenced or
// transitively-referenced features in the specified pallet which are deprecated.
func (m ImportModifier) CheckDeprecations(
	pallet *FSPallet, loader FSPalletLoader,
) (deprecations []error, err error) {
	feature, err := pallet.LoadFeature(m.Source, loader)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't load referenced feature %s", m.Source)
	}
	if deprecation := feature.Decl.Deprecated; deprecation != "" {
		return []error{errors.Errorf("feature %s is deprecated: %s", feature.Name, deprecation)}, nil
	}

	resolved, err := ResolveImport(pallet, loader, feature)
	if err != nil {
		return deprecations, errors.Wrapf(err, "couldn't resolve feature %s", feature.Name)
	}
	deprecations, err = resolved.CheckDeprecations(loader)
	if err != nil {
		return deprecations, err
	}
	wrapped := make([]error, 0, len(deprecations))
	for _, deprecation := range deprecations {
		wrapped = append(wrapped, errors.Wrapf(deprecation, "referenced by feature %s", feature.Name))
	}
	return wrapped, nil
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
	fsys ffs.PathedFS, pattern string, opts ...doublestar.GlobOption,
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
	modifier ImportModifier, pallet *FSPallet, pathMappings map[string]string, loader FSPalletLoader,
) error {
	feature, err := pallet.LoadFeature(modifier.Source, loader)
	if err != nil {
		return errors.Wrapf(err, "couldn't load feature %s", modifier.Source)
	}
	resolved := &ResolvedImport{
		Import: feature,
		Pallet: pallet,
	}
	featureMappings, err := resolved.Evaluate(loader)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't evaluate feature %s to determine file imports to add", modifier.Source,
		)
	}
	maps.Insert(pathMappings, maps.All(featureMappings))
	return nil
}

func applyRemoveFeatureModifier(
	modifier ImportModifier, pallet *FSPallet, pathMappings map[string]string, loader FSPalletLoader,
) error {
	feature, err := pallet.LoadFeature(modifier.Source, loader)
	if err != nil {
		return errors.Wrapf(err, "couldn't load feature %s", modifier.Source)
	}
	resolved := &ResolvedImport{
		Import: feature,
		Pallet: pallet,
	}
	featureMappings, err := resolved.Evaluate(loader)
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
