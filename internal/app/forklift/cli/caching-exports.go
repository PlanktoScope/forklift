package cli

import (
	"context"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/forklift-run/forklift/exp/caching"
	fplt "github.com/forklift-run/forklift/exp/pallets"
	"github.com/forklift-run/forklift/internal/app/forklift"
)

// Download

func DownloadExportFiles(
	indent int, deplsLoader ResolvedDeplsLoader, pkgLoader fplt.FSPkgLoader,
	dlCache *caching.FSDownloadCache,
	platform string, includeDisabled, parallel bool,
) error {
	httpDownloads, ociDownloads, err := forklift.ListRequiredDownloads(
		deplsLoader, pkgLoader, includeDisabled,
	)
	if err != nil {
		return errors.Wrap(err, "couldn't determine file downloads required by package deployments")
	}
	if len(httpDownloads)+len(ociDownloads) == 0 {
		return nil
	}

	IndentedFprintln(indent, os.Stderr, "Downloading files for export...")
	indent++
	newHTTP := make([]string, 0, len(httpDownloads))
	for _, url := range httpDownloads {
		ok, err := dlCache.HasFile(url)
		if err != nil {
			return errors.Wrapf(
				err, "couldn't determine whether the cache of downloaded files includes %s", url,
			)
		}
		if ok {
			IndentedFprintf(indent, os.Stderr, "Skipped already-cached file download: %s\n", url)
			continue
		}
		newHTTP = append(newHTTP, url)
	}

	newOCI := make([]string, 0, len(ociDownloads))
	for _, url := range ociDownloads {
		ok, err := dlCache.HasOCIImage(url)
		if err != nil {
			return errors.Wrapf(
				err, "couldn't determine whether the cache of downloaded OCI images includes %s", url,
			)
		}
		if ok {
			IndentedFprintf(indent, os.Stderr, "Skipped already-cached OCI image download: %s\n", url)
			continue
		}
		newOCI = append(newOCI, url)
	}

	if parallel {
		return downloadParallel(indent, newHTTP, newOCI, platform, dlCache, http.DefaultClient)
	}
	return downloadSerial(indent, newHTTP, newOCI, platform, dlCache, http.DefaultClient)
}

func downloadParallel(
	indent int, httpURLs, ociImageNames []string, platform string, cache *caching.FSDownloadCache,
	hc *http.Client,
) error {
	eg, egctx := errgroup.WithContext(context.Background())
	for _, url := range httpURLs {
		eg.Go(func() error {
			IndentedFprintf(indent, os.Stderr, "Downloading file %s...\n", url)
			outputPath, err := cache.GetFilePath(url)
			if err != nil {
				return errors.Wrapf(err, "couldn't determine path to cache download for %s", url)
			}
			if err = forklift.DownloadFile(egctx, url, outputPath, hc); err != nil {
				return errors.Wrapf(err, "couldn't download %s", url)
			}
			IndentedFprintf(indent, os.Stderr, "Downloaded %s\n", url)
			return nil
		})
	}
	for _, imageName := range ociImageNames {
		eg.Go(func() error {
			IndentedFprintf(indent, os.Stderr, "Downloading OCI container image %s...\n", imageName)
			outputPath, err := cache.GetOCIImagePath(imageName)
			if err != nil {
				return errors.Wrapf(err, "couldn't determine path to cache download for %s", imageName)
			}
			if err = forklift.DownloadOCIImage(egctx, imageName, outputPath, platform); err != nil {
				return errors.Wrapf(err, "couldn't download %s", imageName)
			}
			IndentedFprintf(indent, os.Stderr, "Downloaded %s\n", imageName)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

func downloadSerial(
	indent int, httpURLs, ociImageNames []string, platform string, cache *caching.FSDownloadCache,
	hc *http.Client,
) error {
	for _, url := range httpURLs {
		IndentedFprintf(indent, os.Stderr, "Downloading file %s to cache...\n", url)
		outputPath, err := cache.GetFilePath(url)
		if err != nil {
			return errors.Wrapf(err, "couldn't determine path to cache download for %s", url)
		}
		if err = forklift.DownloadFile(context.Background(), url, outputPath, hc); err != nil {
			return errors.Wrapf(err, "couldn't download %s", url)
		}
		IndentedFprintf(indent, os.Stderr, "Downloaded %s\n", url)
	}
	for _, imageName := range ociImageNames {
		IndentedFprintf(
			indent, os.Stderr, "Downloading OCI container image %s to cache...\n", imageName,
		)
		outputPath, err := cache.GetFilePath(imageName)
		if err != nil {
			return errors.Wrapf(err, "couldn't determine path to cache download for %s", imageName)
		}
		if err = forklift.DownloadOCIImage(
			context.Background(), imageName, outputPath, platform,
		); err != nil {
			return errors.Wrapf(err, "couldn't download %s", imageName)
		}
		IndentedFprintf(indent, os.Stderr, "Downloaded %s\n", imageName)
	}
	return nil
}
