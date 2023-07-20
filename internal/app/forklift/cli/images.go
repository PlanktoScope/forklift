package cli

import (
	"context"
	"fmt"
	"os"

	dct "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/clients/docker"
)

func DownloadImages(
	indent int, envPath, cachePath string, replacementRepos map[string]forklift.ExternalRepo,
) error {
	orderedImages, err := listRequiredImages(indent, envPath, cachePath, replacementRepos)
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
	indent int, envPath, cachePath string, replacementRepos map[string]forklift.ExternalRepo,
) ([]string, error) {
	cacheFS := os.DirFS(cachePath)
	depls, err := forklift.ListDepls(os.DirFS(envPath), cacheFS, replacementRepos)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't identify Pallet package deployments specified by environment %s", envPath,
		)
	}
	orderedImages := make([]string, 0, len(depls))
	images := make(map[string]struct{})
	for _, depl := range depls {
		IndentedPrintf(
			indent, "Checking Docker container images used by package deployment %s...\n", depl.Name,
		)
		if !depl.Pkg.Cached.Config.Deployment.DefinesStack() {
			continue
		}

		stackConfig, err := loadStackDefinition(depl.Pkg.Cached)
		if err != nil {
			return nil, err
		}
		for _, service := range stackConfig.Services {
			BulletedPrintf(indent+1, "%s: %s\n", service.Name, service.Image)
			if _, ok := images[service.Image]; !ok {
				images[service.Image] = struct{}{}
				orderedImages = append(orderedImages, service.Image)
			}
		}
	}
	return orderedImages, nil
}

func loadStackDefinition(pkg forklift.CachedPkg) (*dct.Config, error) {
	definitionFile := pkg.Config.Deployment.DefinitionFile
	stackConfig, err := docker.LoadStackDefinition(pkg.FS, definitionFile)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load Docker stack definition from %s/%s", pkg.FSPath, definitionFile,
		)
	}
	return stackConfig, nil
}
