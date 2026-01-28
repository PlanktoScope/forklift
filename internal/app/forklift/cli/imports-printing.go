package cli

import (
	"fmt"
	"io"
	"path"
	"slices"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/internal/app/forklift"
)

func FprintPalletImports(indent int, out io.Writer, pallet *forklift.FSPallet) error {
	imps, err := pallet.LoadImports("**/*")
	if err != nil {
		return err
	}
	for _, imp := range imps {
		IndentedFprintf(indent, out, "%s\n", imp.Name)
	}
	return nil
}

func FprintImportInfo(
	indent int, out io.Writer,
	pallet *forklift.FSPallet, cache forklift.PathedPalletCache, importName string,
) error {
	imp, err := pallet.LoadImport(importName)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find import group declaration %s in pallet %s", importName, pallet.FS.Path(),
		)
	}
	resolved, err := forklift.ResolveImport(pallet, cache, imp)
	if err != nil {
		return errors.Wrapf(err, "couldn't resolve import group %s", imp.Name)
	}
	resolved.Pallet, err = forklift.MergeFSPallet(resolved.Pallet, cache, nil)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't print merge pallet referenced by resolved import group %s", imp.Name,
		)
	}
	if err = FprintResolvedImport(indent, out, resolved, cache); err != nil {
		return errors.Wrapf(err, "couldn't print resolved import group %s", imp.Name)
	}
	return nil
}

func FprintResolvedImport(
	indent int, out io.Writer, imp *forklift.ResolvedImport, loader forklift.FSPalletLoader,
) error {
	IndentedFprint(indent, out, "Import group")
	if imp.Import.Decl.Disabled {
		_, _ = fmt.Fprint(out, " (disabled!)")
	}
	_, _ = fmt.Fprintf(out, " %s:\n", imp.Name)
	indent++

	deprecations, err := imp.CheckDeprecations(loader)
	if err != nil {
		return errors.Wrapf(err, "couldn't check deprecations for import %s", imp.Name)
	}
	if len(deprecations) > 0 {
		IndentedFprintln(indent, out, "Deprecation warnings:")
		for _, deprecation := range deprecations {
			BulletedFprintln(indent+1, out, deprecation)
		}
	}

	IndentedFprintf(indent, out, "Import source: %s\n", imp.Pallet.Path())

	if err := fprintModifiers(indent, out, imp.Decl.Modifiers, imp.Pallet, loader); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(out)
	IndentedFprintln(indent, out, "Imported files:")
	if err := fprintImportEvaluation(indent+1, out, imp, loader); err != nil {
		return err
	}

	return nil
}

func fprintModifiers(
	indent int, out io.Writer,
	modifiers []forklift.ImportModifier, plt *forklift.FSPallet, loader forklift.FSPalletLoader,
) error {
	IndentedFprint(indent, out, "Sequential definition:")
	if len(modifiers) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	indent++
	for i, modifier := range modifiers {
		switch modifier.Type {
		case forklift.ImportModifierTypeAdd:
			fprintAddModifier(indent, out, i, modifier)
		case forklift.ImportModifierTypeRemove:
			fprintRemoveModifier(indent, out, i, modifier)
		case forklift.ImportModifierTypeAddFeature:
			if err := fprintAddFeatureModifier(indent, out, i, modifier, plt, loader); err != nil {
				return err
			}
		case forklift.ImportModifierTypeRemoveFeature:
			if err := fprintRemoveFeatureModifier(indent, out, i, modifier, plt, loader); err != nil {
				return err
			}
		default:
			BulletedFprintf(
				indent, out, "[%d] Unknown modifier type %s: %+v\n", i, modifier.Type, modifier,
			)
		}
	}
	return nil
}

func fprintAddModifier(indent int, out io.Writer, index int, modifier forklift.ImportModifier) {
	BulletedFprintf(indent, out, "[%d] Add files to group", index)
	if modifier.Description == "" {
		_, _ = fmt.Fprintln(out)
	} else {
		_, _ = fmt.Fprintf(out, ": %s\n", modifier.Description)
	}
	indent++
	indent++ // Because we're nesting a bulleted list in another bulleted list
	for _, filter := range modifier.OnlyMatchingAny {
		if modifier.Source == modifier.Target {
			BulletedFprintf(indent, out, "Add: %s\n", path.Join(modifier.Target, filter))
			continue
		}
		BulletedFprintf(indent, out, "From source: %s\n", path.Join(modifier.Source, filter))
		IndentedFprintf(indent+1, out, "Add as:      %s\n", path.Join(modifier.Target, filter))
	}
}

func fprintRemoveModifier(indent int, out io.Writer, index int, modifier forklift.ImportModifier) {
	BulletedFprintf(indent, out, "[%d] Remove files from group", index)
	if modifier.Description == "" {
		_, _ = fmt.Fprintln(out)
	} else {
		_, _ = fmt.Fprintf(out, ": %s\n", modifier.Description)
	}
	indent++
	indent++ // Because we're nesting a bulleted list in another bulleted list
	for _, filter := range modifier.OnlyMatchingAny {
		BulletedFprintf(indent, out, "Remove: %s\n", path.Join(modifier.Target, filter))
	}
}

func fprintAddFeatureModifier(
	indent int, out io.Writer, index int, modifier forklift.ImportModifier, plt *forklift.FSPallet,
	loader forklift.FSPalletLoader,
) error {
	BulletedFprintf(indent, out, "[%d] Add feature-flagged files to group", index)
	if modifier.Description == "" {
		_, _ = fmt.Fprintln(out)
	} else {
		_, _ = fmt.Fprintf(out, ": %s\n", modifier.Description)
	}
	return errors.Wrap(
		fprintReferencedFeature(indent+1, out, modifier.Source, plt, loader),
		"couldn't load feature in modifier",
	)
}

func fprintReferencedFeature(
	indent int, out io.Writer, name string, plt *forklift.FSPallet, loader forklift.FSPalletLoader,
) error {
	IndentedFprintf(indent, out, "Feature %s", name)
	feature, err := plt.LoadFeature(name, loader)
	if err != nil {
		return errors.Wrapf(err, "couldn't load feature %s", name)
	}

	if feature.Decl.Description != "" {
		_, _ = fmt.Fprintf(out, ": %s\n", feature.Decl.Description)
	} else {
		_, _ = fmt.Fprintln(out, " (no description)")
	}

	resolved := &forklift.ResolvedImport{
		Import: feature,
		Pallet: plt,
	}
	deprecations, err := resolved.CheckDeprecations(loader)
	if err != nil {
		return errors.Wrapf(err, "couldn't check deprecations for import %s", resolved.Name)
	}
	if len(deprecations) > 0 {
		IndentedFprintln(indent, out, "Deprecation notices:")
		for _, deprecation := range deprecations {
			BulletedFprintln(indent+1, out, deprecation)
		}
	}
	return nil
}

func fprintRemoveFeatureModifier(
	indent int, out io.Writer, index int, modifier forklift.ImportModifier, plt *forklift.FSPallet,
	loader forklift.FSPalletLoader,
) error {
	BulletedFprintf(indent, out, "[%d] Remove feature-flagged files from group", index)
	if modifier.Description == "" {
		_, _ = fmt.Fprintln(out)
	} else {
		_, _ = fmt.Fprintf(out, ": %s\n", modifier.Description)
	}
	return errors.Wrap(
		fprintReferencedFeature(indent+1, out, modifier.Source, plt, loader),
		"couldn't load feature in modifier",
	)
}

func fprintImportEvaluation(
	indent int, out io.Writer, imp *forklift.ResolvedImport, loader forklift.FSPalletLoader,
) error {
	importMappings, err := imp.Evaluate(loader)
	if err != nil {
		return errors.Wrapf(err, "couldn't evaluate import group")
	}

	targets := make([]string, 0, len(importMappings))
	for target := range importMappings {
		targets = append(targets, target)
	}
	slices.Sort(targets)
	for _, target := range targets {
		BulletedFprintf(indent, out, "As:          %s\n", target)
		IndentedFprintf(indent+1, out, "From source: %s\n", importMappings[target])
	}

	return nil
}
