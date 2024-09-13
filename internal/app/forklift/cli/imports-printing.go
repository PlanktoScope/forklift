package cli

import (
	"fmt"
	"path"
	"slices"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
)

func PrintPalletImports(indent int, pallet *forklift.FSPallet) error {
	imps, err := pallet.LoadImports("**/*")
	if err != nil {
		return err
	}
	for _, imp := range imps {
		IndentedPrintf(indent, "%s\n", imp.Name)
	}
	return nil
}

func PrintImportInfo(
	indent int, pallet *forklift.FSPallet, cache forklift.PathedPalletCache, importName string,
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
	if err = PrintResolvedImport(indent, resolved); err != nil {
		return errors.Wrapf(err, "couldn't print resolved import group %s", imp.Name)
	}
	return nil
}

func PrintResolvedImport(indent int, imp *forklift.ResolvedImport) error {
	IndentedPrint(indent, "Import group")
	if imp.Import.Def.Disabled {
		fmt.Print(" (disabled!)")
	}
	fmt.Printf(" %s:\n", imp.Name)
	indent++

	deprecations := imp.CheckDeprecations()
	if len(deprecations) > 0 {
		IndentedPrintln(indent, "Deprecation warnings:")
		for _, deprecation := range deprecations {
			BulletedPrintln(indent+1, deprecation)
		}
	}

	IndentedPrintf(indent, "Import source: %s\n", imp.Pallet.Path())

	if err := printModifiers(indent, imp.Def.Modifiers, imp.Pallet); err != nil {
		return err
	}

	fmt.Println()
	IndentedPrintln(indent, "Imported files:")
	if err := printImportEvaluation(indent+1, imp); err != nil {
		return err
	}

	return nil
}

func printModifiers(indent int, modifiers []forklift.ImportModifier, plt *forklift.FSPallet) error {
	IndentedPrint(indent, "Sequential definition:")
	if len(modifiers) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++
	for i, modifier := range modifiers {
		switch modifier.Type {
		case forklift.ImportModifierTypeAdd:
			printAddModifier(indent, i, modifier)
		case forklift.ImportModifierTypeRemove:
			printRemoveModifier(indent, i, modifier)
		case forklift.ImportModifierTypeAddFeature:
			if err := printAddFeatureModifier(indent, i, modifier, plt); err != nil {
				return err
			}
		case forklift.ImportModifierTypeRemoveFeature:
			if err := printRemoveFeatureModifier(indent, i, modifier, plt); err != nil {
				return err
			}
		default:
			BulletedPrintf(indent, "[%d] Unknown modifier type %s: %+v\n", i, modifier.Type, modifier)
		}
	}
	return nil
}

func printAddModifier(indent, index int, modifier forklift.ImportModifier) {
	BulletedPrintf(indent, "[%d] Add files to group", index)
	if modifier.Description == "" {
		fmt.Println()
	} else {
		fmt.Printf(": %s\n", modifier.Description)
	}
	indent++
	indent++ // Because we're nesting a bulleted list in another bulleted list
	for _, filter := range modifier.OnlyMatchingAny {
		if modifier.Source == modifier.Target {
			BulletedPrintf(indent, "Add: %s\n", path.Join(modifier.Target, filter))
			continue
		}
		BulletedPrintf(indent, "From source: %s\n", path.Join(modifier.Source, filter))
		IndentedPrintf(indent+1, "Add as:      %s\n", path.Join(modifier.Target, filter))
	}
}

func printRemoveModifier(indent, index int, modifier forklift.ImportModifier) {
	BulletedPrintf(indent, "[%d] Remove files from group", index)
	if modifier.Description == "" {
		fmt.Println()
	} else {
		fmt.Printf(": %s\n", modifier.Description)
	}
	indent++
	indent++ // Because we're nesting a bulleted list in another bulleted list
	for _, filter := range modifier.OnlyMatchingAny {
		BulletedPrintf(indent, "Remove: %s\n", path.Join(modifier.Target, filter))
	}
}

func printAddFeatureModifier(
	indent, index int, modifier forklift.ImportModifier, plt *forklift.FSPallet,
) error {
	BulletedPrintf(indent, "[%d] Add feature-flagged files to group", index)
	if modifier.Description == "" {
		fmt.Println()
	} else {
		fmt.Printf(": %s\n", modifier.Description)
	}
	return errors.Wrap(
		printReferencedFeature(indent+1, modifier.Source, plt), "couldn't load feature in modifier",
	)
}

func printReferencedFeature(indent int, name string, plt *forklift.FSPallet) error {
	IndentedPrintf(indent, "Feature %s", name)
	feature, err := plt.LoadFeature(name)
	if err != nil {
		return errors.Wrapf(err, "couldn't load feature %s", name)
	}

	if feature.Def.Description != "" {
		fmt.Printf(": %s\n", feature.Def.Description)
	} else {
		fmt.Println(" (no description)")
	}

	resolved := &forklift.ResolvedImport{
		Import: feature,
		Pallet: plt,
	}
	deprecations := resolved.CheckDeprecations()
	if len(deprecations) > 0 {
		IndentedPrintln(indent, "Deprecation notices:")
		for _, deprecation := range deprecations {
			BulletedPrintln(indent+1, deprecation)
		}
	}
	return nil
}

func printRemoveFeatureModifier(
	indent, index int, modifier forklift.ImportModifier, plt *forklift.FSPallet,
) error {
	BulletedPrintf(indent, "[%d] Remove feature-flagged files from group", index)
	if modifier.Description == "" {
		fmt.Println()
	} else {
		fmt.Printf(": %s\n", modifier.Description)
	}
	return errors.Wrap(
		printReferencedFeature(indent+1, modifier.Source, plt), "couldn't load feature in modifier",
	)
}

func printImportEvaluation(indent int, imp *forklift.ResolvedImport) error {
	importMappings, err := imp.Evaluate()
	if err != nil {
		return errors.Wrapf(err, "couldn't evaluate import group")
	}

	targets := make([]string, 0, len(importMappings))
	for target := range importMappings {
		targets = append(targets, target)
	}
	slices.Sort(targets)
	for _, target := range targets {
		BulletedPrintf(indent, "As:          %s\n", target)
		IndentedPrintf(indent+1, "From source: %s\n", importMappings[target])
	}

	return nil
}
