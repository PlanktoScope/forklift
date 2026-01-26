package cli

import (
	"fmt"
	"io"
	"path"
	"sort"

	dct "github.com/compose-spec/compose-go/v2/types"
	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/internal/app/forklift"
	"github.com/forklift-run/forklift/pkg/core"
)

func FprintPalletDepls(indent int, out io.Writer, pallet *forklift.FSPallet) error {
	depls, err := pallet.LoadDepls("**/*")
	if err != nil {
		return err
	}
	for _, depl := range depls {
		IndentedFprintf(indent, out, "%s\n", depl.Name)
	}
	return nil
}

func FprintDeplInfo(
	indent int, out io.Writer,
	pallet *forklift.FSPallet, cache forklift.PathedPalletCache, deplName string,
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
	if err = FprintResolvedDepl(indent, out, cache, resolved); err != nil {
		return errors.Wrapf(err, "couldn't print resolved package deployment %s", depl.Name)
	}
	return nil
}

func FprintResolvedDepl(
	indent int, out io.Writer, cache forklift.PathedPalletCache, resolved *forklift.ResolvedDepl,
) error {
	if err := fprintDepl(indent, out, cache, resolved); err != nil {
		return err
	}
	indent++

	definesApp, err := resolved.DefinesComposeApp()
	if err != nil {
		return errors.Wrap(err, "couldn't determine whether package deployment defines a Compose app")
	}
	if !definesApp {
		return nil
	}

	appDef, err := resolved.LoadComposeAppDefinition(false)
	if err != nil {
		return errors.Wrap(err, "couldn't load Compose app definition")
	}
	IndentedFprintf(indent, out, "Deploys as Docker Compose app %s:\n", appDef.Name)
	indent++

	if err = fprintDockerAppDefFiles(indent, out, resolved); err != nil {
		return err
	}
	fprintDockerAppDef(indent, out, appDef)

	// TODO: print the state of the Docker Compose app associated with deplName - or maybe that should
	// be a `forklift depl show-d deplName` command instead?
	return nil
}

func fprintDepl(
	indent int, out io.Writer, cache forklift.PathedPalletCache, depl *forklift.ResolvedDepl,
) error {
	IndentedFprint(indent, out, "Package deployment")
	if depl.Depl.Decl.Disabled {
		_, _ = fmt.Fprint(out, " (disabled!)")
	}
	_, _ = fmt.Fprintf(out, ": %s\n", depl.Name)
	indent++

	fprintDeplPkg(indent, out, cache, depl)

	IndentedFprint(indent, out, "Enabled features:")
	if len(depl.Decl.Features) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	enabledFeatures, err := depl.EnabledFeatures()
	if err != nil {
		return errors.Wrap(err, "couldn't determine enabled features")
	}
	fprintFeatures(indent+1, out, enabledFeatures)

	disabledFeatures := depl.DisabledFeatures()
	IndentedFprint(indent, out, "Disabled features:")
	if len(disabledFeatures) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	fprintFeatures(indent+1, out, disabledFeatures)

	fileExportTargets, err := depl.GetFileExportTargets()
	if err != nil {
		return errors.Wrap(err, "couldn't determine export file targets")
	}
	IndentedFprint(indent, out, "File export targets:")
	if len(fileExportTargets) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	for _, fileExport := range fileExportTargets {
		BulletedFprintln(indent+1, out, fileExport)
	}
	return nil
}

func fprintDeplPkg(
	indent int, out io.Writer, cache forklift.PathedPalletCache, depl *forklift.ResolvedDepl,
) {
	IndentedFprintf(indent, out, "Deploys package: %s\n", depl.Decl.Package)
	indent++

	IndentedFprintf(indent, out, "Description: %s\n", depl.Pkg.Decl.Package.Description)
	if depl.Pkg.PkgTree.Decl.PkgTree != (core.PkgTreeSpec{}) {
		fprintPkgPallet(indent, out, cache, depl.Pkg)
	}
}

func fprintFeatures(indent int, out io.Writer, features map[string]core.PkgFeatureSpec) {
	orderedNames := make([]string, 0, len(features))
	for name := range features {
		orderedNames = append(orderedNames, name)
	}
	sort.Strings(orderedNames)
	for _, name := range orderedNames {
		if description := features[name].Description; description != "" {
			IndentedFprintf(indent, out, "%s: %s\n", name, description)
			continue
		}
		IndentedFprintf(indent, out, "%s\n", name)
	}
}

func fprintDockerAppDefFiles(indent int, out io.Writer, depl *forklift.ResolvedDepl) error {
	composeFiles, err := depl.GetComposeFilenames()
	if err != nil {
		return errors.Wrap(err, "couldn't determine Compose files for deployment")
	}

	IndentedFprint(indent, out, "Compose files: ")
	if len(composeFiles) == 0 {
		_, _ = fmt.Fprint(out, "(none)")
		return nil
	}
	_, _ = fmt.Fprintln(out)
	for _, file := range composeFiles {
		BulletedFprintln(indent+1, out, path.Join(depl.Pkg.Path(), file))
	}
	return nil
}

func fprintDockerAppDef(indent int, out io.Writer, appDef *dct.Project) {
	fprintDockerAppServices(indent, out, appDef.Services)
	fprintDockerAppNetworks(indent, out, appDef.Networks)
	fprintDockerAppVolumes(indent, out, appDef.Volumes)
}

func fprintDockerAppServices(indent int, out io.Writer, services dct.Services) {
	if len(services) == 0 {
		return
	}
	IndentedFprintln(indent, out, "Services:")
	sortedServices := make([]dct.ServiceConfig, 0, len(services))
	for _, service := range services {
		sortedServices = append(sortedServices, service)
	}
	sort.Slice(sortedServices, func(i, j int) bool {
		return sortedServices[i].Name < sortedServices[j].Name
	})
	indent++

	for _, service := range sortedServices {
		IndentedFprintf(indent, out, "%s: %s\n", service.Name, service.Image)
	}
}

func fprintDockerAppNetworks(indent int, out io.Writer, networks dct.Networks) {
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
	IndentedFprint(indent, out, "Networks:")
	sort.Slice(networkNames, func(i, j int) bool {
		return networks[networkNames[i]].Name < networks[networkNames[j]].Name
	})
	if len(networkNames) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	indent++

	for _, name := range networkNames {
		BulletedFprintln(indent, out, networks[name].Name)
	}
}

func fprintDockerAppVolumes(indent int, out io.Writer, volumes dct.Volumes) {
	if len(volumes) == 0 {
		return
	}
	volumeNames := make([]string, 0, len(volumes))
	for name := range volumes {
		volumeNames = append(volumeNames, name)
	}
	IndentedFprint(indent, out, "Volumes:")
	sort.Slice(volumeNames, func(i, j int) bool {
		return volumeNames[i] < volumeNames[j]
	})
	if len(volumeNames) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	indent++

	for _, name := range volumeNames {
		BulletedFprintln(indent, out, name)
	}
}

func FprintDeplPkgLocation(
	indent int, out io.Writer,
	pallet *forklift.FSPallet, cache forklift.PathedPalletCache, deplName string, allowDisabled bool,
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
	if resolved.Decl.Disabled && !allowDisabled {
		return errors.Errorf("package deployment %s is not enabled!", depl.Name)
	}
	_, _ = fmt.Fprintln(out, resolved.Pkg.FS.Path())
	return nil
}
