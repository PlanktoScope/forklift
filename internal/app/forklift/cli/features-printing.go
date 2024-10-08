package cli

import (
	"fmt"
	"slices"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
)

func PrintPalletFeatures(indent int, pallet *forklift.FSPallet) error {
	imps, err := pallet.LoadFeatures("**/*")
	if err != nil {
		return err
	}
	for _, imp := range imps {
		IndentedPrintf(indent, "%s\n", imp.Name)
	}
	return nil
}

func PrintFeatureInfo(
	indent int, pallet *forklift.FSPallet, cache forklift.PathedPalletCache, featureName string,
) error {
	imp, err := pallet.LoadFeature(featureName, cache)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find feature declaration %s in pallet %s", featureName, pallet.FS.Path(),
		)
	}
	resolved := &forklift.ResolvedImport{
		Import: imp,
		Pallet: pallet,
	}
	resolved.Pallet, err = forklift.MergeFSPallet(resolved.Pallet, cache, nil)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't print merge pallet referenced by feature %s resolved as import group %s",
			featureName, imp.Name,
		)
	}
	if err = PrintFeature(indent, resolved, cache); err != nil {
		return errors.Wrapf(
			err, "couldn't print feature %s resolved as import group %s", featureName, imp.Name,
		)
	}
	return nil
}

func PrintFeature(indent int, imp *forklift.ResolvedImport, loader forklift.FSPalletLoader) error {
	IndentedPrintf(indent, "Feature %s:\n", imp.Name)
	indent++
	deprecations, err := imp.CheckDeprecations(loader)
	if err != nil {
		return errors.Wrapf(err, "couldn't check deprecations for import %s", imp.Name)
	}
	if len(deprecations) > 0 {
		IndentedPrintln(indent, "Deprecation warnings:")
		for _, deprecation := range deprecations {
			BulletedPrintln(indent+1, deprecation)
		}
	}

	if err := printModifiers(indent, imp.Def.Modifiers, imp.Pallet, loader); err != nil {
		return err
	}

	fmt.Println()
	IndentedPrintln(indent, "Files grouped for import:")
	if err := printFeatureEvaluation(indent+1, imp, loader); err != nil {
		return err
	}

	return nil
}

func printFeatureEvaluation(
	indent int, imp *forklift.ResolvedImport, loader forklift.FSPalletLoader,
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
		BulletedPrintln(indent, target)
	}

	return nil
}
