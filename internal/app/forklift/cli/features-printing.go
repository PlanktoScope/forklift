package cli

import (
	"fmt"
	"io"
	"slices"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
)

func FprintPalletFeatures(indent int, out io.Writer, pallet *forklift.FSPallet) error {
	imps, err := pallet.LoadFeatures("**/*")
	if err != nil {
		return err
	}
	for _, imp := range imps {
		IndentedFprintf(indent, out, "%s\n", imp.Name)
	}
	return nil
}

func FprintFeatureInfo(
	indent int, out io.Writer,
	pallet *forklift.FSPallet, cache forklift.PathedPalletCache, featureName string,
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
	if err = FprintFeature(indent, out, resolved, cache); err != nil {
		return errors.Wrapf(
			err, "couldn't print feature %s resolved as import group %s", featureName, imp.Name,
		)
	}
	return nil
}

func FprintFeature(
	indent int, out io.Writer, imp *forklift.ResolvedImport, loader forklift.FSPalletLoader,
) error {
	IndentedFprintf(indent, out, "Feature %s:\n", imp.Name)
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

	if err := fprintModifiers(indent, out, imp.Def.Modifiers, imp.Pallet, loader); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(out)
	IndentedFprintln(indent, out, "Files grouped for import:")
	if err := fprintFeatureEvaluation(indent+1, out, imp, loader); err != nil {
		return err
	}

	return nil
}

func fprintFeatureEvaluation(
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
		BulletedFprintln(indent, out, target)
	}

	return nil
}
