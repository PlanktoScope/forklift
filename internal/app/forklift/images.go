package forklift

import (
	"maps"
	"slices"

	"github.com/pkg/errors"

	fplt "github.com/forklift-run/forklift/exp/pallets"
	"github.com/forklift-run/forklift/exp/structures"
)

func ListRequiredImages(
	deplsLoader ResolvedDeplsLoader, pkgLoader fplt.FSPkgLoader, includeDisabled bool,
) ([]string, error) {
	depls, err := deplsLoader.LoadDepls("**/*")
	if err != nil {
		return nil, err
	}
	if !includeDisabled {
		depls = fplt.FilterDeplsForEnabled(depls)
	}
	resolved, err := fplt.ResolveDepls(deplsLoader, pkgLoader, depls)
	if err != nil {
		return nil, err
	}

	images := make(structures.Set[string])
	for _, depl := range resolved {
		definesApp, err := depl.DefinesComposeApp()
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't determine whether package deployment %s defines a Compose app", depl.Name,
			)
		}
		if !definesApp {
			continue
		}

		appDef, err := depl.LoadComposeAppDefinition(false)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't load Compose app definition")
		}
		for _, service := range appDef.Services {
			images.Add(service.Image)
		}
	}
	return slices.Sorted(maps.Keys(images)), nil
}
