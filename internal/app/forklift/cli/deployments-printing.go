package cli

import (
	"fmt"
	"path"
	"sort"

	dct "github.com/compose-spec/compose-go/v2/types"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/pkg/core"
)

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

func PrintDeplPkgLocation(
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
