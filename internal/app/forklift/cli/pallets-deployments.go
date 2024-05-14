package cli

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"

	dct "github.com/compose-spec/compose-go/v2/types"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/pkg/core"
	"github.com/PlanktoScope/forklift/pkg/structures"
)

// Print

func PrintPalletDepls(indent int, pallet *forklift.FSPallet, loader forklift.FSPkgLoader) error {
	depls, err := pallet.LoadDepls("**/*")
	if err != nil {
		return err
	}
	for _, depl := range depls {
		IndentedPrintf(indent, "%s\n", depl.Name)
	}
	return nil
}

func PrintDeplInfo(
	indent int, pallet *forklift.FSPallet, cache forklift.PathedRepoCache, deplName string,
) error {
	depl, err := pallet.LoadDepl(deplName)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find package deployment declaration %s in pallet %s",
			deplName, pallet.FS.Path(),
		)
	}
	resolved, err := forklift.ResolveDepl(pallet, cache, depl)
	if err != nil {
		return errors.Wrapf(err, "couldn't resolve package deployment %s", depl.Name)
	}
	if err = PrintResolvedDepl(indent, cache, resolved); err != nil {
		return errors.Wrapf(err, "couldn't print resolved package deployment %s", depl.Name)
	}
	return nil
}

func PrintResolvedDepl(
	indent int, cache forklift.PathedRepoCache, resolved *forklift.ResolvedDepl,
) error {
	if err := printDepl(indent, cache, resolved); err != nil {
		return err
	}
	indent++

	definesApp, err := resolved.DefinesApp()
	if err != nil {
		return errors.Wrap(err, "couldn't determine whether package deployment defines a Compose app")
	}
	if !definesApp {
		return nil
	}

	appDef, err := loadAppDefinition(resolved)
	if err != nil {
		return errors.Wrap(err, "couldn't load Compose app definition")
	}
	IndentedPrintf(indent, "Deploys as Docker Compose app %s:\n", appDef.Name)
	indent++

	if err = printDockerAppDefFiles(indent, resolved); err != nil {
		return err
	}
	printDockerAppDef(indent, appDef)

	// TODO: print the state of the Docker Compose app associated with deplName - or maybe that should
	// be a `forklift depl show-d deplName` command instead?
	return nil
}

func printDepl(indent int, cache forklift.PathedRepoCache, depl *forklift.ResolvedDepl) error {
	IndentedPrint(indent, "Package deployment")
	if depl.Depl.Def.Disabled {
		fmt.Print(" (disabled!)")
	}
	fmt.Printf(": %s\n", depl.Name)
	indent++

	printDeplPkg(indent, cache, depl)

	IndentedPrint(indent, "Enabled features:")
	if len(depl.Def.Features) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	enabledFeatures, err := depl.EnabledFeatures()
	if err != nil {
		return errors.Wrap(err, "couldn't determine enabled features")
	}
	printFeatures(indent+1, enabledFeatures)

	disabledFeatures := depl.DisabledFeatures()
	IndentedPrint(indent, "Disabled features:")
	if len(disabledFeatures) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	printFeatures(indent+1, disabledFeatures)

	fileExportTargets, err := depl.GetFileExportTargets()
	if err != nil {
		return errors.Wrap(err, "couldn't determine export file targets")
	}
	IndentedPrint(indent, "File export targets:")
	if len(fileExportTargets) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for _, fileExport := range fileExportTargets {
		BulletedPrintln(indent+1, fileExport)
	}
	return nil
}

func printDeplPkg(indent int, cache forklift.PathedRepoCache, depl *forklift.ResolvedDepl) {
	IndentedPrintf(indent, "Deploys package: %s\n", depl.Def.Package)
	indent++

	IndentedPrintf(indent, "Description: %s\n", depl.Pkg.Def.Package.Description)
	if depl.Pkg.Repo.Def.Repo != (core.RepoSpec{}) {
		printPkgRepo(indent, cache, depl.Pkg)
	}
}

func printFeatures(indent int, features map[string]core.PkgFeatureSpec) {
	orderedNames := make([]string, 0, len(features))
	for name := range features {
		orderedNames = append(orderedNames, name)
	}
	sort.Strings(orderedNames)
	for _, name := range orderedNames {
		if description := features[name].Description; description != "" {
			IndentedPrintf(indent, "%s: %s\n", name, description)
			continue
		}
		IndentedPrintf(indent, "%s\n", name)
	}
}

func printDockerAppDefFiles(indent int, depl *forklift.ResolvedDepl) error {
	composeFiles, err := depl.GetComposeFilenames()
	if err != nil {
		return errors.Wrap(err, "couldn't determine Compose files for deployment")
	}

	IndentedPrintf(indent, "Compose files: ")
	if len(composeFiles) == 0 {
		fmt.Printf("(none)")
		return nil
	}
	fmt.Println()
	for _, file := range composeFiles {
		BulletedPrintln(indent+1, path.Join(depl.Pkg.Path(), file))
	}
	return nil
}

func printDockerAppDef(indent int, appDef *dct.Project) {
	printDockerAppServices(indent, appDef.Services)
	printDockerAppNetworks(indent, appDef.Networks)
	printDockerAppVolumes(indent, appDef.Volumes)
}

func printDockerAppServices(indent int, services dct.Services) {
	if len(services) == 0 {
		return
	}
	IndentedPrintln(indent, "Services:")
	sortedServices := make([]dct.ServiceConfig, 0, len(services))
	for _, service := range services {
		sortedServices = append(sortedServices, service)
	}
	sort.Slice(sortedServices, func(i, j int) bool {
		return sortedServices[i].Name < sortedServices[j].Name
	})
	indent++

	for _, service := range sortedServices {
		IndentedPrintf(indent, "%s: %s\n", service.Name, service.Image)
	}
}

func printDockerAppNetworks(indent int, networks dct.Networks) {
	if len(networks) == 0 {
		return
	}
	networkNames := make([]string, 0, len(networks))
	for name, network := range networks {
		if name == "default" && network.Name == "none" {
			// Ignore the default network if its creation is suppressed by the Compose file
			continue
		}
		networkNames = append(networkNames, name)
	}
	IndentedPrint(indent, "Networks:")
	sort.Slice(networkNames, func(i, j int) bool {
		return networks[networkNames[i]].Name < networks[networkNames[j]].Name
	})
	if len(networkNames) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, name := range networkNames {
		BulletedPrintln(indent, networks[name].Name)
	}
}

func printDockerAppVolumes(indent int, volumes dct.Volumes) {
	if len(volumes) == 0 {
		return
	}
	volumeNames := make([]string, 0, len(volumes))
	for name := range volumes {
		volumeNames = append(volumeNames, name)
	}
	IndentedPrint(indent, "Volumes:")
	sort.Slice(volumeNames, func(i, j int) bool {
		return volumeNames[i] < volumeNames[j]
	})
	if len(volumeNames) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, name := range volumeNames {
		BulletedPrintln(indent, name)
	}
}

func PrintDeplPkgPath(
	indent int, pallet *forklift.FSPallet, cache forklift.PathedRepoCache, deplName string,
	allowDisabled bool,
) error {
	depl, err := pallet.LoadDepl(deplName)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find package deployment declaration %s in pallet %s",
			deplName, pallet.FS.Path(),
		)
	}
	resolved, err := forklift.ResolveDepl(pallet, cache, depl)
	if err != nil {
		return errors.Wrapf(err, "couldn't resolve package deployment %s", depl.Name)
	}
	if resolved.Def.Disabled && !allowDisabled {
		return errors.Errorf("package deployment %s is not enabled!", depl.Name)
	}
	fmt.Println(resolved.Pkg.FS.Path())
	return nil
}

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
	IndentedPrintf(
		indent, "Adding %spackage deployment %s for %s%s...\n",
		disabledString, deplName, pkgPath, featuresString,
	)
	depl := forklift.Depl{
		Name: deplName,
		Def: forklift.DeplDef{
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
		IndentedPrintf(indent, "Warning: package deployment has invalid settings: %s", err.Error())
	}

	if err := writeDepl(pallet, depl); err != nil {
		return errors.Wrapf(err, "couldn't save deployment %s", deplName)
	}
	return nil
}

func checkDepl(
	pallet *forklift.FSPallet, pkgLoader forklift.FSPkgLoader, depl forklift.Depl,
) error {
	pkg, _, err := forklift.LoadRequiredFSPkg(pallet, pkgLoader, depl.Def.Package)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't resolve package path %s to a package using the pallet's repo requirements",
			depl.Def.Package,
		)
	}

	allowedFeatures := pkg.Def.Features
	unrecognizedFeatures := make([]string, 0, len(depl.Def.Features))
	for _, name := range depl.Def.Features {
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
	if err = encoder.Encode(depl.Def); err != nil {
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
	fmt.Printf("Removing package deployments from %s...\n", pallet.FS.Path())
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
	IndentedPrintf(
		indent, "Setting package deployment %s to deploy package %s...\n", deplName, pkgPath,
	)
	depl, err := pallet.LoadDepl(deplName)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find package deployment declaration %s in pallet %s",
			deplName, pallet.FS.Path(),
		)
	}

	depl.Def.Package = pkgPath
	// We want to check both the package path and the feature flags for validity, since changing the
	// package path could change the allowed feature flags:
	if err := checkDepl(pallet, pkgLoader, depl); err != nil {
		if !force {
			return errors.Wrap(
				err, "package deployment has invalid settings; to skip this check, enable the --force flag",
			)
		}
		IndentedPrintf(indent, "Warning: package deployment has invalid settings: %s", err.Error())
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
	IndentedPrintf(indent, "Enabling features %+v in package deployment %s...\n", features, deplName)
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
	for _, name := range depl.Def.Features {
		existingFeatures.Add(name)
	}
	allowedFeatures := resolved.Pkg.Def.Features
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
			unrecognizedFeatures, depl.Def.Package,
		)
		if !force {
			return err
		}
		IndentedPrintf(indent, "Warning: %s", err.Error())
	}

	depl.Def.Features = append(depl.Def.Features, newFeatures...)
	if err := writeDepl(pallet, depl); err != nil {
		return errors.Wrapf(err, "couldn't save updated deployment declaration %s", depl.Name)
	}
	return nil
}

// Remove Feature

func RemoveDeplFeat(
	indent int, pallet *forklift.FSPallet, deplName string, features []string,
) error {
	IndentedPrintf(indent, "Disabling features %+v in package deployment %s...\n", features, deplName)
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
	newFeatures := make([]string, 0, len(depl.Def.Features))
	for _, name := range depl.Def.Features {
		if removedFeatures.Has(name) {
			continue
		}
		newFeatures = append(newFeatures, name)
	}

	depl.Def.Features = newFeatures
	if err := writeDepl(pallet, depl); err != nil {
		return errors.Wrapf(err, "couldn't save updated deployment declaration %s", depl.Name)
	}
	return nil
}

// Set Disabled

func SetDeplDisabled(indent int, pallet *forklift.FSPallet, deplName string, disabled bool) error {
	if disabled {
		IndentedPrintf(indent, "Disabling package deployment %s...\n", deplName)
	} else {
		IndentedPrintf(indent, "Enabling package deployment %s...\n", deplName)
	}
	depl, err := pallet.LoadDepl(deplName)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find package deployment declaration %s in pallet %s",
			deplName, pallet.FS.Path(),
		)
	}

	depl.Def.Disabled = disabled
	if err := writeDepl(pallet, depl); err != nil {
		return errors.Wrapf(err, "couldn't save updated deployment declaration %s", depl.Name)
	}
	return nil
}
