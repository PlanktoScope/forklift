package forklift

import (
	"slices"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/exp/caching"
	ffs "github.com/forklift-run/forklift/exp/fs"
	fplt "github.com/forklift-run/forklift/exp/pallets"
	"github.com/forklift-run/forklift/exp/structures"
	fws "github.com/forklift-run/forklift/exp/workspaces"
)

func GetDownloadCache(wpath string, ensureCache bool) (*caching.FSDownloadCache, error) {
	workspace, err := fws.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	cache, err := workspace.GetDownloadCache()
	if err != nil {
		return nil, err
	}

	if ensureCache && !cache.Exists() {
		if err = ffs.EnsureExists(cache.FS.Path()); err != nil {
			return nil, err
		}
	}
	return cache, nil
}

// Download

type ResolvedDeplsLoader interface {
	fplt.PkgReqLoader
	LoadDepls(searchPattern string) ([]fplt.Depl, error)
}

func ListRequiredDownloads(
	deplsLoader ResolvedDeplsLoader, pkgLoader fplt.FSPkgLoader, includeDisabled bool,
) (http, oci []string, err error) {
	depls, err := deplsLoader.LoadDepls("**/*")
	if err != nil {
		return nil, nil, err
	}
	if !includeDisabled {
		depls = fplt.FilterDeplsForEnabled(depls)
	}
	resolved, err := fplt.ResolveDepls(deplsLoader, pkgLoader, depls)
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
