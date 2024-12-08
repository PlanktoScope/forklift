package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/internal/clients/crane"
	"github.com/PlanktoScope/forklift/pkg/structures"
)

func GetDownloadCache(wpath string, ensureCache bool) (*forklift.FSDownloadCache, error) {
	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	cache, err := workspace.GetDownloadCache()
	if err != nil {
		return nil, err
	}

	if ensureCache && !cache.Exists() {
		if err = forklift.EnsureExists(cache.FS.Path()); err != nil {
			return nil, err
		}
	}
	return cache, nil
}

// Download

func DownloadExportFiles(
	indent int, deplsLoader ResolvedDeplsLoader, pkgLoader forklift.FSPkgLoader,
	dlCache *forklift.FSDownloadCache,
	platform string, includeDisabled, parallel bool,
) error {
	httpDownloads, ociDownloads, err := ListRequiredDownloads(deplsLoader, pkgLoader, includeDisabled)
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

func ListRequiredDownloads(
	deplsLoader ResolvedDeplsLoader, pkgLoader forklift.FSPkgLoader, includeDisabled bool,
) (http, oci []string, err error) {
	depls, err := deplsLoader.LoadDepls("**/*")
	if err != nil {
		return nil, nil, err
	}
	if !includeDisabled {
		depls = forklift.FilterDeplsForEnabled(depls)
	}
	resolved, err := forklift.ResolveDepls(deplsLoader, pkgLoader, depls)
	if err != nil {
		return nil, nil, err
	}

	http = make([]string, 0, len(resolved))
	oci = make([]string, 0, len(resolved))
	added := make(structures.Set[string])
	for _, depl := range resolved {
		httpURLs, err := depl.GetHTTPFileDownloadURLs()
		if err != nil {
			return nil, nil, errors.Wrapf(
				err, "couldn't determine http file downloads for export by deployment %s", depl.Name,
			)
		}
		for _, url := range httpURLs {
			if added.Has(url) {
				continue
			}
			added.Add(url)
			http = append(http, url)
		}
		ociImageNames, err := depl.GetOCIImageDownloadNames()
		if err != nil {
			return nil, nil, errors.Wrapf(
				err, "couldn't determine oci image downloads for export by deployment %s", depl.Name,
			)
		}
		for _, imageName := range ociImageNames {
			if added.Has(imageName) {
				continue
			}
			added.Add(imageName)
			oci = append(oci, imageName)
		}
	}
	slices.Sort(http)
	slices.Sort(oci)
	return http, oci, nil
}

func downloadParallel(
	indent int, httpURLs, ociImageNames []string, platform string, cache *forklift.FSDownloadCache,
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
			if err = downloadFile(egctx, url, outputPath, hc); err != nil {
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
			if err = downloadOCIImage(egctx, imageName, outputPath, platform); err != nil {
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
	indent int, httpURLs, ociImageNames []string, platform string, cache *forklift.FSDownloadCache,
	hc *http.Client,
) error {
	for _, url := range httpURLs {
		IndentedFprintf(indent, os.Stderr, "Downloading file %s to cache...\n", url)
		outputPath, err := cache.GetFilePath(url)
		if err != nil {
			return errors.Wrapf(err, "couldn't determine path to cache download for %s", url)
		}
		if err = downloadFile(context.Background(), url, outputPath, hc); err != nil {
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
		if err = downloadOCIImage(context.Background(), imageName, outputPath, platform); err != nil {
			return errors.Wrapf(err, "couldn't download %s", imageName)
		}
		IndentedFprintf(indent, os.Stderr, "Downloaded %s\n", imageName)
	}
	return nil
}

func downloadFile(ctx context.Context, url, outputPath string, hc *http.Client) error {
	if err := forklift.EnsureExists(filepath.FromSlash(path.Dir(outputPath))); err != nil {
		return err
	}
	tmpPath := outputPath + ".fkldownload"
	file, err := os.Create(filepath.FromSlash(tmpPath))
	if err != nil {
		return errors.Wrapf(err, "couldn't create temporary download file at %s", tmpPath)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// FIXME: handle this error better
			fmt.Fprintf(os.Stderr, "Error: couldn't close temporary download file %s\n", tmpPath)
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errors.Wrapf(err, "couldn't make http get request for %s", url)
	}
	res, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			// FIXME: handle this error better
			fmt.Fprintf(os.Stderr, "Error: couldn't close http response for %s\n", url)
		}
	}()

	_, err = io.Copy(file, res.Body)
	if err != nil {
		return errors.Wrapf(err, "couldn't download %s to %s", url, tmpPath)
	}

	if err = os.Rename(filepath.FromSlash(tmpPath), filepath.FromSlash(outputPath)); err != nil {
		return errors.Wrapf(
			err, "couldn't commit completed download from %s to %s", tmpPath, outputPath,
		)
	}
	return nil
}

func downloadOCIImage(ctx context.Context, imageName, outputPath, platform string) error {
	if err := forklift.EnsureExists(filepath.FromSlash(path.Dir(outputPath))); err != nil {
		return err
	}
	tmpPath := outputPath + ".fkldownload"
	file, err := os.Create(filepath.FromSlash(tmpPath))
	if err != nil {
		return errors.Wrapf(err, "couldn't create temporary download file at %s", tmpPath)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// FIXME: handle this error better
			fmt.Fprintf(os.Stderr, "Error: couldn't close temporary download file %s\n", tmpPath)
		}
	}()

	if err = crane.ExportOCIImage(ctx, imageName, file, platform); err != nil {
		return errors.Wrapf(err, "couldn't download and export image as a tarball: %s", imageName)
	}

	if err = os.Rename(filepath.FromSlash(tmpPath), filepath.FromSlash(outputPath)); err != nil {
		return errors.Wrapf(
			err, "couldn't commit completed download from %s to %s", tmpPath, outputPath,
		)
	}
	return nil
}
