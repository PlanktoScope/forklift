package cli

import (
	"fmt"
	"os"
	"sort"

	dct "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Print

func PrintEnvDepls(
	indent int, envPath, cachePath string, replacementRepos map[string]*pallets.FSRepo,
) error {
	depls, err := forklift.ListDepls(os.DirFS(envPath), os.DirFS(cachePath), replacementRepos)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify Pallet package deployments specified by environment %s", envPath,
		)
	}
	for _, depl := range depls {
		IndentedPrintf(indent, "%s\n", depl.Name)
	}
	return nil
}

func PrintDeplInfo(
	indent int, envPath, cachePath string, replacementRepos map[string]*pallets.FSRepo,
	deplName string,
) error {
	cacheFS := os.DirFS(cachePath)
	depl, err := forklift.LoadDepl(os.DirFS(envPath), cacheFS, replacementRepos, deplName)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find package deployment specification %s in environment %s", deplName, envPath,
		)
	}
	printDepl(indent, depl)
	indent++

	if depl.Pkg.Config.Deployment.DefinesStack() {
		fmt.Println()
		IndentedPrintln(indent, "Deploys with Docker stack:")
		stackConfig, err := loadStackDefinition(depl.Pkg.FSPkg)
		if err != nil {
			return err
		}
		printDockerStackConfig(indent+1, *stackConfig)
	}

	// TODO: print the state of the Docker stack associated with deplName - or maybe that should be
	// a `forklift depl show-d deplName` command instead?
	return nil
}

func printDepl(indent int, depl forklift.Depl) {
	IndentedPrintf(indent, "Pallet package deployment: %s\n", depl.Name)
	indent++

	printDeplPkg(indent, depl)

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

func printDeplPkg(indent int, depl forklift.Depl) {
	IndentedPrintf(indent, "Deploys Pallet package: %s\n", depl.Config.Package)
	indent++

	IndentedPrintf(indent, "Description: %s\n", depl.Pkg.Config.Package.Description)
	printVersionedPkgRepo(indent, depl.Pkg)
}

func printFeatures(indent int, features map[string]pallets.PkgFeatureSpec) {
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

func printDockerStackConfig(indent int, stackConfig dct.Config) {
	printDockerStackServices(indent, stackConfig.Services)
	// TODO: also print networks, volumes, etc.
}

func printDockerStackServices(indent int, services []dct.ServiceConfig) {
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
