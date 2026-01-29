package bundling

import (
	"cmp"
	"slices"
	"strings"

	dct "github.com/compose-spec/compose-go/v2/types"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
	"github.com/forklift-run/forklift/pkg/structures"
	"github.com/pkg/errors"
)

func makeComposeAppSummary(
	depl *fplt.ResolvedDepl, bundleRoot string,
) (BundleDeplComposeApp, error) {
	appDef, err := depl.LoadComposeAppDefinition(true)
	if err != nil {
		return BundleDeplComposeApp{}, errors.Wrap(err, "couldn't load Compose app definition")
	}

	services := make(structures.Set[string])
	images := make(structures.Set[string])
	for _, service := range appDef.Services {
		services.Add(service.Name)
		images.Add(service.Image)
	}

	createdBindMounts, requiredBindMounts := makeComposeAppBindMountSummaries(appDef, bundleRoot)
	createdVolumes, requiredVolumes := makeComposeAppVolumeSummaries(appDef)
	createdNetworks, requiredNetworks := makeComposeAppNetworkSummaries(appDef)

	app := BundleDeplComposeApp{
		Name:               appDef.Name,
		Services:           slices.Sorted(services.All()),
		Images:             slices.Sorted(images.All()),
		CreatedBindMounts:  slices.Sorted(createdBindMounts.All()),
		RequiredBindMounts: slices.Sorted(requiredBindMounts.All()),
		CreatedVolumes:     slices.Sorted(createdVolumes.All()),
		RequiredVolumes:    slices.Sorted(requiredVolumes.All()),
		CreatedNetworks:    slices.Sorted(createdNetworks.All()),
		RequiredNetworks:   slices.Sorted(requiredNetworks.All()),
	}
	return app, nil
}

func makeComposeAppBindMountSummaries(
	appDef *dct.Project, bundleRoot string,
) (created structures.Set[string], required structures.Set[string]) {
	created = make(structures.Set[string])
	required = make(structures.Set[string])
	for _, service := range appDef.Services {
		for _, volume := range service.Volumes {
			if volume.Type != "bind" {
				continue
			}
			// If the path on the host is declared as a relative path, then it's supposed to be a path
			// managed by Forklift, and its location will depend on where the bundle is. So we record it
			// relative to the path of the bundle.
			volume.Source = strings.TrimPrefix(volume.Source, bundleRoot+"/")
			if volume.Bind != nil && !volume.Bind.CreateHostPath {
				required.Add(volume.Source)
				continue
			}
			created.Add(volume.Source)
		}
	}

	return created.Difference(required), required
}

func makeComposeAppVolumeSummaries(
	appDef *dct.Project,
) (created structures.Set[string], required structures.Set[string]) {
	created = make(structures.Set[string])
	required = make(structures.Set[string])
	for volumeName, volume := range appDef.Volumes {
		if volume.External {
			required.Add(cmp.Or(volume.Name, volumeName))
			continue
		}
		created.Add(cmp.Or(volume.Name, volumeName))
	}
	return created, required
}

func makeComposeAppNetworkSummaries(
	appDef *dct.Project,
) (created structures.Set[string], required structures.Set[string]) {
	created = make(structures.Set[string])
	required = make(structures.Set[string])
	for networkName, network := range appDef.Networks {
		if network.External {
			if networkName == "default" && network.Name == "none" {
				// If the network is Docker's pre-made "none" network (which uses the null network driver),
				// we ignore it for brevity since the intention is to suppress creating a network for the
				// container.
				continue
			}
			required.Add(cmp.Or(network.Name, networkName))
			continue
		}
		created.Add(cmp.Or(network.Name, networkName))
	}
	return created, required
}
