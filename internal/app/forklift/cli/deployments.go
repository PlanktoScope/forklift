package cli

import (
	"fmt"
	"maps"
	"os"
	"slices"

	"github.com/pkg/errors"

	fplt "github.com/forklift-run/forklift/exp/pallets"
	"github.com/forklift-run/forklift/internal/app/forklift"
)

// Add

func AddDepl(
	indent int, pallet *fplt.FSPallet, pkgLoader fplt.FSPkgLoader,
	deplName, pkgPath string, features []string, disabled, force bool,
) error {
	disabledString := ""
	if disabled {
		disabledString = "disabled "
	}
	featuresString := ""
	if len(features) > 0 {
		featuresString = fmt.Sprintf(" (with feature flags: %+v)", features)
	}
	IndentedFprintf(
		indent, os.Stderr, "Adding %spackage deployment %s for %s%s...\n",
		disabledString, deplName, pkgPath, featuresString,
	)
	depl := fplt.Depl{
		Name: deplName,
		Decl: fplt.DeplDecl{
			Package:  pkgPath,
			Features: features,
			Disabled: disabled,
		},
	}

	if err := forklift.CheckDepl(pallet, pkgLoader, depl); err != nil {
		if !force {
			return errors.Wrap(
				err, "package deployment has invalid settings; to skip this check, enable the --force flag",
			)
		}
		IndentedFprintf(
			indent, os.Stderr, "Warning: package deployment has invalid settings: %s", err.Error(),
		)
	}

	if err := forklift.WriteDepl(pallet, depl); err != nil {
		return errors.Wrapf(err, "couldn't save deployment %s", deplName)
	}
	return nil
}

// Remove

func RemoveDepls(indent int, pallet *fplt.FSPallet, deplNames []string) error {
	IndentedFprintf(indent, os.Stderr, "Removing package deployments from %s...\n", pallet.FS.Path())
	for _, deplName := range deplNames {
		if err := pallet.RemoveDepl(deplName); err != nil {
			return err
		}
	}
	// TODO: maybe it'd be better to remove everything we can remove and then report errors at the
	// end?
	return nil
}

// Set Package

func SetDeplPkg(
	indent int, pallet *fplt.FSPallet, pkgLoader fplt.FSPkgLoader,
	deplName, pkgPath string, force bool,
) error {
	IndentedFprintf(
		indent, os.Stderr, "Setting package deployment %s to deploy package %s...\n", deplName, pkgPath,
	)
	depl, err := pallet.LoadDepl(deplName)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find package deployment declaration %s in pallet %s",
			deplName, pallet.FS.Path(),
		)
	}

	depl.Decl.Package = pkgPath
	// We want to check both the package path and the feature flags for validity, since changing the
	// package path could change the allowed feature flags:
	if err := forklift.CheckDepl(pallet, pkgLoader, depl); err != nil {
		if !force {
			return errors.Wrap(
				err, "package deployment has invalid settings; to skip this check, enable the --force flag",
			)
		}
		IndentedFprintf(
			indent, os.Stderr, "Warning: package deployment has invalid settings: %s", err.Error(),
		)
	}

	if err := forklift.WriteDepl(pallet, depl); err != nil {
		return errors.Wrapf(err, "couldn't save updated deployment declaration %s", depl.Name)
	}
	return nil
}

// Add Feature

func AddDeplFeat(
	indent int, pallet *fplt.FSPallet, pkgLoader fplt.FSPkgLoader,
	deplName string, features []string, force bool,
) error {
	IndentedFprintf(
		indent, os.Stderr, "Enabling features %+v in package deployment %s...\n", features, deplName,
	)
	depl, err := pallet.LoadDepl(deplName)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find package deployment declaration %s in pallet %s",
			deplName, pallet.FS.Path(),
		)
	}
	resolved, err := fplt.ResolveDepl(pallet, pkgLoader, depl)
	if err != nil {
		return errors.Wrapf(err, "couldn't resolve package deployment %s", depl.Name)
	}

	appended, unrecognized := depl.Decl.Features.With(
		features, slices.Collect(maps.Keys(resolved.Pkg.Decl.Features)),
	)
	if len(unrecognized) > 0 {
		err := errors.Errorf(
			"feature flags %+v aren't allowed by package %s; to skip this check, enable the --force flag",
			unrecognized, depl.Decl.Package,
		)
		if !force {
			return err
		}
		IndentedFprintf(indent, os.Stderr, "Warning: %s", err.Error())
	}

	depl.Decl.Features = appended
	if err := forklift.WriteDepl(pallet, depl); err != nil {
		return errors.Wrapf(err, "couldn't save updated deployment declaration %s", depl.Name)
	}
	return nil
}

// Remove Feature

func RemoveDeplFeat(
	indent int, pallet *fplt.FSPallet, deplName string, features []string,
) error {
	IndentedFprintf(
		indent, os.Stderr, "Disabling features %+v in package deployment %s...\n", features, deplName,
	)
	depl, err := pallet.LoadDepl(deplName)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find package deployment declaration %s in pallet %s",
			deplName, pallet.FS.Path(),
		)
	}

	depl.Decl.Features = depl.Decl.Features.Without(features)
	if err := forklift.WriteDepl(pallet, depl); err != nil {
		return errors.Wrapf(err, "couldn't save updated deployment declaration %s", depl.Name)
	}
	return nil
}

// Set Disabled

func SetDeplDisabled(indent int, pallet *fplt.FSPallet, deplName string, disabled bool) error {
	if disabled {
		IndentedFprintf(indent, os.Stderr, "Disabling package deployment %s...\n", deplName)
	} else {
		IndentedFprintf(indent, os.Stderr, "Enabling package deployment %s...\n", deplName)
	}
	depl, err := pallet.LoadDepl(deplName)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find package deployment declaration %s in pallet %s",
			deplName, pallet.FS.Path(),
		)
	}

	depl.Decl.Disabled = disabled
	if err := forklift.WriteDepl(pallet, depl); err != nil {
		return errors.Wrapf(err, "couldn't save updated deployment declaration %s", depl.Name)
	}
	return nil
}
