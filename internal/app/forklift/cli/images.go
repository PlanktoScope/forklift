package cli

import (
	"context"
	"fmt"
	"os"
	"path"

	dct "github.com/compose-spec/compose-go/types"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/clients/docker"
	"github.com/PlanktoScope/forklift/pkg/core"
)

// Download

func DownloadImages(indent int, pallet *forklift.FSPallet, loader forklift.FSPkgLoader) error {
	orderedImages, err := listRequiredImages(indent, pallet, loader)
	if err != nil {
		return errors.Wrap(err, "couldn't determine images required by package deployments")
	}

	dc, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}
	for _, image := range orderedImages {
		fmt.Println()
		IndentedPrintf(indent, "Downloading %s...\n", image)
		pulled, err := dc.PullImage(context.Background(), image, docker.NewOutStream(os.Stdout))
		if err != nil {
			return errors.Wrapf(err, "couldn't download %s", image)
		}
		IndentedPrintf(indent, "Downloaded %s from %s\n", pulled.Reference(), pulled.RepoInfo().Name)
	}
	return nil
}

func listRequiredImages(
	indent int, pallet *forklift.FSPallet, loader forklift.FSPkgLoader,
) ([]string, error) {
	depls, err := pallet.LoadDepls("**/*")
	if err != nil {
		return nil, err
	}
	resolved, err := forklift.ResolveDepls(pallet, loader, depls)
	if err != nil {
		return nil, err
	}

	orderedImages := make([]string, 0, len(resolved))
	images := make(map[string]struct{})
	for _, depl := range resolved {
		IndentedPrintf(
			indent, "Checking Docker container images used by package deployment %s...\n", depl.Name,
		)
		if !depl.Pkg.Def.Deployment.DefinesApp() {
			continue
		}

		appDef, err := loadAppDefinition(depl.Pkg)
		if err != nil {
			return nil, err
		}
		for _, service := range appDef.Services {
			BulletedPrintf(indent+1, "%s: %s\n", service.Name, service.Image)
			if _, ok := images[service.Image]; !ok {
				images[service.Image] = struct{}{}
				orderedImages = append(orderedImages, service.Image)
			}
		}
	}
	return orderedImages, nil
}

func loadAppDefinition(pkg *core.FSPkg) (*dct.Project, error) {
	appDef, err := docker.LoadAppDefinition(
		pkg.FS, path.Base(pkg.Path()), pkg.Def.Deployment.DefinitionFiles, nil,
	)
	// TODO: also load the docker compose files for all features
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load Docker Compose app definition for a basic deployment of %s",
			pkg.FS.Path(),
		)
	}
	return appDef, nil
}
