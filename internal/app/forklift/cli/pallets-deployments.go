package cli

import (
	"fmt"
	"path"
	"sort"

	dct "github.com/compose-spec/compose-go/types"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/pkg/core"
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
			err, "couldn't find package deployment specification %s in pallet %s",
			deplName, pallet.FS.Path(),
		)
	}
	resolved, err := forklift.ResolveDepl(pallet, cache, depl)
	if err != nil {
		return errors.Wrapf(err, "couldn't resolve package deployment %s", depl.Name)
	}
	printDepl(indent, cache, resolved)
	indent++

	definesApp, err := resolved.DefinesApp()
	if err != nil {
		return errors.Wrapf(
			err, "couldn't determine whether package deployment %s defines a Compose app", depl.Name,
		)
	}
	if definesApp {
		fmt.Println()
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
	}

	// TODO: print the state of the Docker Compose app associated with deplName - or maybe that should
	// be a `forklift depl show-d deplName` command instead?
	return nil
}

func printDepl(indent int, cache forklift.PathedRepoCache, depl *forklift.ResolvedDepl) {
	IndentedPrint(indent, "Package deployment")
	if depl.Depl.Def.Disabled {
		fmt.Print(" (disabled!)")
	}
	fmt.Printf(": %s\n", depl.Name)
	indent++

	printDeplPkg(indent, cache, depl)

	enabledFeatures, err := depl.EnabledFeatures()
	if err != nil {
		IndentedPrintf(indent, "Warning: couldn't determine enabled features: %s\n", err.Error())
	}
	IndentedPrint(indent, "Enabled features:")
	if len(enabledFeatures) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	printFeatures(indent+1, enabledFeatures)

	disabledFeatures := depl.DisabledFeatures()
	IndentedPrint(indent, "Disabled features:")
	if len(disabledFeatures) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	printFeatures(indent+1, disabledFeatures)
}

func printDeplPkg(indent int, cache forklift.PathedRepoCache, depl *forklift.ResolvedDepl) {
	IndentedPrintf(indent, "Deploys package: %s\n", depl.Def.Package)
	indent++

	IndentedPrintf(indent, "Description: %s\n", depl.Pkg.Def.Package.Description)
	printPkgRepo(indent, cache, depl.Pkg)
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
	// TODO: also print networks, volumes, etc.
}

func printDockerAppServices(indent int, services []dct.ServiceConfig) {
	if len(services) == 0 {
		return
	}
	IndentedPrint(indent, "Services:")
	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})
	if len(services) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, service := range services {
		IndentedPrintf(indent, "%s: %s\n", service.Name, service.Image)
	}
}
