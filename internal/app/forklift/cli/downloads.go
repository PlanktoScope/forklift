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
	"github.com/PlanktoScope/forklift/pkg/structures"
)

func GetDlCache(wpath string, ensureCache bool) (*forklift.FSDownloadCache, error) {
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
	dlCache *forklift.FSDownloadCache, includeDisabled, parallel bool,
) error {
	downloads, err := listRequiredDownloads(deplsLoader, pkgLoader, includeDisabled)
	if err != nil {
		return errors.Wrap(err, "couldn't determine images required by package deployments")
	}
	if len(downloads) == 0 {
		return nil
	}

	newDownloads := make([]string, 0, len(downloads))
	for _, url := range downloads {
		ok, err := dlCache.HasFile(url)
		if err != nil {
			return errors.Wrapf(
				err, "couldn't determine whether the cache of downloaded files includes %s", url,
			)
		}
		if ok {
			IndentedPrintf(indent, "Skipping already-cached download: %s\n", url)
			continue
		}
		newDownloads = append(newDownloads, url)
	}

	if parallel {
		return downloadHTTPFilesParallel(indent, newDownloads, dlCache, http.DefaultClient)
	}
	return downloadHTTPFilesSerial(indent, newDownloads, dlCache, http.DefaultClient)
}

func listRequiredDownloads(
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

	orderedDownloads := make([]string, 0, len(resolved))
	downloads := make(structures.Set[string])
	for _, depl := range resolved {
		urls, err := depl.GetHTTPFileDownloadURLs()
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't determine file downloads for export by deployment %s", depl.Name,
			)
		}
		for _, url := range urls {
			if !downloads.Has(url) {
				downloads.Add(url)
				orderedDownloads = append(orderedDownloads, url)
			}
		}
	}
	slices.Sort(orderedDownloads)
	return orderedDownloads, nil
}

func downloadHTTPFilesParallel(
	indent int, urls []string, cache *forklift.FSDownloadCache, hc *http.Client,
) error {
	eg, egctx := errgroup.WithContext(context.Background())
	for _, url := range urls {
		eg.Go(func() error {
			IndentedPrintf(indent, "Downloading %s to cache...\n", url)
			outputPath, err := cache.GetFilePath(url)
			if err != nil {
				return errors.Wrapf(err, "couldn't determine path to cache download for %s", url)
			}
			if err = downloadFile(egctx, url, outputPath, hc); err != nil {
				return errors.Wrapf(err, "couldn't download %s", url)
			}
			IndentedPrintf(indent, "Downloaded %s\n", url)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

func downloadHTTPFilesSerial(
	indent int, urls []string, cache *forklift.FSDownloadCache, hc *http.Client,
) error {
	for _, url := range urls {
		IndentedPrintf(indent, "Downloading %s to cache...\n", url)
		outputPath, err := cache.GetFilePath(url)
		if err != nil {
			return errors.Wrapf(err, "couldn't determine path to cache download for %s", url)
		}
		if err = downloadFile(context.Background(), url, outputPath, hc); err != nil {
			return errors.Wrapf(err, "couldn't download %s", url)
		}
		IndentedPrintf(indent, "Downloaded %s\n", url)
	}
	return nil
}

func downloadFile(ctx context.Context, url, outputPath string, hc *http.Client) error {
	if err := forklift.EnsureExists(filepath.FromSlash(path.Dir(outputPath))); err != nil {
		return err
	}
	tmpPath := outputPath + ".tmp"
	file, err := os.Create(filepath.FromSlash(tmpPath))
	if err != nil {
		return errors.Wrapf(err, "couldn't create temporary download file at %s", tmpPath)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// FIXME: handle this error better
			fmt.Printf("Error: couldn't close temporary download file %s\n", tmpPath)
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
			fmt.Printf("Error: couldn't close http response for %s\n", url)
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
