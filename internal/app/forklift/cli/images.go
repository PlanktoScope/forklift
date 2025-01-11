package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/clients/cli"
	"github.com/PlanktoScope/forklift/internal/clients/docker"
	"github.com/PlanktoScope/forklift/pkg/structures"
)

// Download

func DownloadImagesForStoreApply(
	indent int, store *forklift.FSStageStore, platform, toolVersion, bundleMinVersion string,
	parallel, ignoreToolVersion bool,
) error {
	next, hasNext := store.GetNext()
	current, hasCurrent := store.GetCurrent()
	indent++

	if hasCurrent && current != next {
		bundle, err := store.LoadFSBundle(current)
		if err != nil {
			return errors.Wrapf(err, "couldn't load staged pallet bundle %d", current)
		}
		if err = CheckBundleShallowCompat(
			bundle, toolVersion, bundleMinVersion, ignoreToolVersion,
		); err != nil {
			return err
		}
		IndentedFprintln(
			indent, os.Stderr,
			"Downloading Docker container images specified by the last successfully-applied staged "+
				"pallet bundle, in case the next to be applied fails to be applied...",
		)
		if err := DownloadImages(indent, bundle, bundle, platform, false, parallel); err != nil {
			return err
		}
	}
	if hasNext {
		bundle, err := store.LoadFSBundle(next)
		if err != nil {
			return errors.Wrapf(err, "couldn't load staged pallet bundle %d", next)
		}
		if err = CheckBundleShallowCompat(
			bundle, toolVersion, bundleMinVersion, ignoreToolVersion,
		); err != nil {
			return err
		}
		IndentedFprintln(
			indent, os.Stderr,
			"Downloading Docker container images specified by the next staged pallet bundle to be "+
				"applied...",
		)
		if err := DownloadImages(indent, bundle, bundle, platform, false, parallel); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr)
	}
	return nil
}

func DownloadImages(
	indent int, deplsLoader ResolvedDeplsLoader, pkgLoader forklift.FSPkgLoader,
	platform string, includeDisabled, parallel bool,
) error {
	orderedImages, err := ListRequiredImages(deplsLoader, pkgLoader, includeDisabled)
	if err != nil {
		return errors.Wrap(err, "couldn't determine images required by package deployments")
	}
	if len(orderedImages) == 0 {
		// When there are no images to download, don't cause an error if we can't initialize the
		// Docker API client!
		return nil
	}

	dc, err := docker.NewClient()
	if err != nil {
		return errors.Wrap(err, "couldn't make Docker API client")
	}

	if parallel {
		return downloadImagesParallel(indent, orderedImages, platform, dc)
	}
	return downloadImagesSerial(indent, orderedImages, platform, dc)
}

func ListRequiredImages(
	deplsLoader ResolvedDeplsLoader, pkgLoader forklift.FSPkgLoader, includeDisabled bool,
) ([]string, error) {
	depls, err := deplsLoader.LoadDepls("**/*")
	if err != nil {
		return nil, err
	}
	if !includeDisabled {
		depls = forklift.FilterDeplsForEnabled(depls)
	}
	resolved, err := forklift.ResolveDepls(deplsLoader, pkgLoader, depls)
	if err != nil {
		return nil, err
	}

	orderedImages := make([]string, 0, len(resolved))
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

		appDef, err := loadAppDefinition(depl)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't load Compose app definition")
		}
		for _, service := range appDef.Services {
			if !images.Has(service.Image) {
				images.Add(service.Image)
				orderedImages = append(orderedImages, service.Image)
			}
		}
	}
	return orderedImages, nil
}

func downloadImagesParallel(indent int, images []string, platform string, dc *docker.Client) error {
	eg, egctx := errgroup.WithContext(context.Background())
	for _, image := range images {
		eg.Go(func() error {
			IndentedFprintf(indent, os.Stderr, "Downloading %s...\n", image)
			pulled, err := dc.PullImage(egctx, image, platform, docker.NewOutStream(io.Discard))
			if err != nil {
				return errors.Wrapf(err, "couldn't download %s", image)
			}
			IndentedFprintf(
				indent, os.Stderr, "Downloaded %s from %s\n", pulled.Reference(), pulled.RepoInfo().Name,
			)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

func downloadImagesSerial(indent int, images []string, platform string, dc *docker.Client) error {
	for _, image := range images {
		IndentedFprintf(indent, os.Stderr, "Downloading %s...\n", image)
		pulled, err := dc.PullImage(
			context.Background(), image, platform,
			docker.NewOutStream(cli.NewIndentedWriter(indent+1, os.Stdout)),
		)
		if err != nil {
			return errors.Wrapf(err, "couldn't download %s", image)
		}
		IndentedFprintf(
			indent+1, os.Stderr, "Downloaded %s from %s\n", pulled.Reference(), pulled.RepoInfo().Name,
		)
	}
	return nil
}
