package cli

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/forklift-run/forklift/internal/app/forklift"
	"github.com/forklift-run/forklift/pkg/structures"
)

// Add

func AddDepl(
	indent int, pallet *forklift.FSPallet, pkgLoader forklift.FSPkgLoader,
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
	depl := forklift.Depl{
		Name: deplName,
		Decl: forklift.DeplDecl{
			Package:  pkgPath,
			Features: features,
			Disabled: disabled,
		},
	}

	if err := checkDepl(pallet, pkgLoader, depl); err != nil {
		if !force {
			return errors.Wrap(
				err, "package deployment has invalid settings; to skip this check, enable the --force flag",
			)
		}
		IndentedFprintf(
			indent, os.Stderr, "Warning: package deployment has invalid settings: %s", err.Error(),
		)
	}

	if err := writeDepl(pallet, depl); err != nil {
		return errors.Wrapf(err, "couldn't save deployment %s", deplName)
	}
	return nil
}

func checkDepl(
	pallet *forklift.FSPallet, pkgLoader forklift.FSPkgLoader, depl forklift.Depl,
) error {
	pkg, _, err := forklift.LoadRequiredFSPkg(pallet, pkgLoader, depl.Decl.Package)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't resolve package path %s to a package using the pallet's repo requirements",
			depl.Decl.Package,
		)
	}

	allowedFeatures := pkg.Decl.Features
	unrecognizedFeatures := make([]string, 0, len(depl.Decl.Features))
	for _, name := range depl.Decl.Features {
		if _, ok := allowedFeatures[name]; !ok {
			unrecognizedFeatures = append(unrecognizedFeatures, name)
		}
	}
	if len(unrecognizedFeatures) > 0 {
		return errors.Errorf("unrecognized feature flags: %+v", unrecognizedFeatures)
	}
	return nil
}

func writeDepl(pallet *forklift.FSPallet, depl forklift.Depl) error {
	deplsFS, err := pallet.GetDeplsFS()
	if err != nil {
		return err
	}
	deplPath := path.Join(deplsFS.Path(), fmt.Sprintf("%s.deploy.yml", depl.Name))
	buf := bytes.Buffer{}
	encoder := yaml.NewEncoder(&buf)
	const yamlIndent = 2
	encoder.SetIndent(yamlIndent)
	if err = encoder.Encode(depl.Decl); err != nil {
		return errors.Wrapf(err, "couldn't marshal package deployment for %s", deplPath)
	}
	if err := forklift.EnsureExists(filepath.FromSlash(path.Dir(deplPath))); err != nil {
		return errors.Wrapf(
			err, "couldn't make directory %s", filepath.FromSlash(path.Dir(deplPath)),
		)
	}
	const perm = 0o644 // owner rw, group r, public r
	if err := os.WriteFile(filepath.FromSlash(deplPath), buf.Bytes(), perm); err != nil {
		return errors.Wrapf(
			err, "couldn't save deployment declaration to %s", filepath.FromSlash(deplPath),
		)
	}
	return nil
}

// Remove

func RemoveDepls(indent int, pallet *forklift.FSPallet, deplNames []string) error {
	IndentedFprintf(indent, os.Stderr, "Removing package deployments from %s...\n", pallet.FS.Path())
	for _, deplName := range deplNames {
		deplsFS, err := pallet.GetDeplsFS()
		if err != nil {
			return err
		}
		deplPath := path.Join(deplsFS.Path(), fmt.Sprintf("%s.deploy.yml", deplName))
		if err = os.RemoveAll(deplPath); err != nil {
			return errors.Wrapf(
				err, "couldn't remove package deployment %s, at %s", deplName, deplPath,
			)
		}
	}
	// TODO: maybe it'd be better to remove everything we can remove and then report errors at the
	// end?
	return nil
}

// Set Package

func SetDeplPkg(
	indent int, pallet *forklift.FSPallet, pkgLoader forklift.FSPkgLoader,
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
	if err := checkDepl(pallet, pkgLoader, depl); err != nil {
		if !force {
			return errors.Wrap(
				err, "package deployment has invalid settings; to skip this check, enable the --force flag",
			)
		}
		IndentedFprintf(
			indent, os.Stderr, "Warning: package deployment has invalid settings: %s", err.Error(),
		)
	}

	if err := writeDepl(pallet, depl); err != nil {
		return errors.Wrapf(err, "couldn't save updated deployment declaration %s", depl.Name)
	}
	return nil
}

// Add Feature

func AddDeplFeat(
	indent int, pallet *forklift.FSPallet, pkgLoader forklift.FSPkgLoader,
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
	resolved, err := forklift.ResolveDepl(pallet, pkgLoader, depl)
	if err != nil {
		return errors.Wrapf(err, "couldn't resolve package deployment %s", depl.Name)
	}

	existingFeatures := make(structures.Set[string])
	for _, name := range depl.Decl.Features {
		existingFeatures.Add(name)
	}
	allowedFeatures := resolved.Pkg.Decl.Features
	unrecognizedFeatures := make([]string, 0, len(features))
	newFeatures := make([]string, 0, len(features))
	for _, name := range features {
		if _, ok := allowedFeatures[name]; !ok {
			unrecognizedFeatures = append(unrecognizedFeatures, name)
		}
		if existingFeatures.Has(name) {
			continue
		}
		newFeatures = append(newFeatures, name)
		existingFeatures.Add(name) // suppress duplicates in the input features list
	}
	if len(unrecognizedFeatures) > 0 {
		err := errors.Errorf(
			"feature flags %+v are allowed by package %s; to skip this check, enable the --force flag",
			unrecognizedFeatures, depl.Decl.Package,
		)
		if !force {
			return err
		}
		IndentedFprintf(indent, os.Stderr, "Warning: %s", err.Error())
	}

	depl.Decl.Features = append(depl.Decl.Features, newFeatures...)
	if err := writeDepl(pallet, depl); err != nil {
		return errors.Wrapf(err, "couldn't save updated deployment declaration %s", depl.Name)
	}
	return nil
}

// Remove Feature

func RemoveDeplFeat(
	indent int, pallet *forklift.FSPallet, deplName string, features []string,
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

	removedFeatures := make(structures.Set[string])
	for _, name := range features {
		removedFeatures.Add(name)
	}
	newFeatures := make([]string, 0, len(depl.Decl.Features))
	for _, name := range depl.Decl.Features {
		if removedFeatures.Has(name) {
			continue
		}
		newFeatures = append(newFeatures, name)
	}

	depl.Decl.Features = newFeatures
	if err := writeDepl(pallet, depl); err != nil {
		return errors.Wrapf(err, "couldn't save updated deployment declaration %s", depl.Name)
	}
	return nil
}

// Set Disabled

func SetDeplDisabled(indent int, pallet *forklift.FSPallet, deplName string, disabled bool) error {
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
	if err := writeDepl(pallet, depl); err != nil {
		return errors.Wrapf(err, "couldn't save updated deployment declaration %s", depl.Name)
	}
	return nil
}
