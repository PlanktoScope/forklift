package pallets

import (
	"cmp"
	"slices"

	"github.com/pkg/errors"

	fpkg "github.com/forklift-run/forklift/pkg/packaging"
)

// ResolvedDepl: File Downloads

// GetHTTPFileDownloadURLs returns a list of the HTTP(s) URLs of files to be downloaded for export
// by the package deployment, with all URLs sorted alphabetically.
func (d *ResolvedDepl) GetHTTPFileDownloadURLs() ([]string, error) {
	downloadURLs := make([]string, 0, len(d.Pkg.Decl.Deployment.Provides.FileExports))
	for _, export := range d.Pkg.Decl.Deployment.Provides.FileExports {
		switch export.SourceType {
		default:
			continue
		case fpkg.FileExportSourceTypeHTTP, fpkg.FileExportSourceTypeHTTPArchive:
		}
		downloadURLs = append(downloadURLs, export.URL)
	}
	enabledFeatures, err := d.EnabledFeatures()
	if err != nil {
		return downloadURLs, errors.Wrapf(
			err, "couldn't determine files to download for export from deployment %s", d.Name,
		)
	}
	for _, name := range sortKeys(enabledFeatures) {
		for _, export := range enabledFeatures[name].Provides.FileExports {
			switch export.SourceType {
			default:
				continue
			case fpkg.FileExportSourceTypeHTTP, fpkg.FileExportSourceTypeHTTPArchive:
			}
			downloadURLs = append(downloadURLs, export.URL)
		}
	}
	slices.Sort(downloadURLs)
	return downloadURLs, nil
}

// GetOCIImageDownloadNames returns a list of the image names of OCI container images to be
// downloaded for export by the package deployment, with all names sorted alphabetically.
func (d *ResolvedDepl) GetOCIImageDownloadNames() ([]string, error) {
	imageNames := make([]string, 0, len(d.Pkg.Decl.Deployment.Provides.FileExports))
	for _, export := range d.Pkg.Decl.Deployment.Provides.FileExports {
		if export.SourceType != fpkg.FileExportSourceTypeOCIImage {
			continue
		}
		imageNames = append(imageNames, export.URL)
	}
	enabledFeatures, err := d.EnabledFeatures()
	if err != nil {
		return imageNames, errors.Wrapf(
			err, "couldn't determine oci images to download for export from deployment %s", d.Name,
		)
	}
	for _, name := range sortKeys(enabledFeatures) {
		for _, export := range enabledFeatures[name].Provides.FileExports {
			if export.SourceType != fpkg.FileExportSourceTypeOCIImage {
				continue
			}
			imageNames = append(imageNames, export.URL)
		}
	}
	slices.Sort(imageNames)
	return imageNames, nil
}

// ResolvedDepl: File Exports

// GetFileExportTargets returns a list of the target paths of the files to be exported by the
// package deployment, with all target file paths sorted alphabetically.
func (d *ResolvedDepl) GetFileExportTargets() ([]string, error) {
	exportTargets := make([]string, 0, len(d.Pkg.Decl.Deployment.Provides.FileExports))
	for _, export := range d.Pkg.Decl.Deployment.Provides.FileExports {
		exportTargets = append(exportTargets, export.Target)
	}
	enabledFeatures, err := d.EnabledFeatures()
	if err != nil {
		return exportTargets, errors.Wrapf(
			err, "couldn't determine exported file targets of deployment %s", d.Name,
		)
	}
	for _, name := range sortKeys(enabledFeatures) {
		for _, export := range enabledFeatures[name].Provides.FileExports {
			exportTargets = append(exportTargets, export.Target)
		}
	}
	slices.Sort(exportTargets)
	return exportTargets, nil
}

// GetFileExports returns a list of file exports to be exported by the package deployment, with
// file export objects sorted alphabetically by their target file paths, and (if multiple source
// files are specified for a target path) preserving precedence of feature flags over the
// deployment section, and preserving precedence among feature flags by alphabetical ordering of
// feature flags.
func (d *ResolvedDepl) GetFileExports() ([]fpkg.FileExportRes, error) {
	exports := append([]fpkg.FileExportRes{}, d.Pkg.Decl.Deployment.Provides.FileExports...)
	enabledFeatures, err := d.EnabledFeatures()
	if err != nil {
		return exports, errors.Wrapf(
			err, "couldn't determine exported file targets of deployment %s", d.Name,
		)
	}
	for _, name := range sortKeys(enabledFeatures) {
		exports = append(exports, enabledFeatures[name].Provides.FileExports...)
	}
	slices.SortStableFunc(exports, func(a, b fpkg.FileExportRes) int {
		return cmp.Compare(a.Target, b.Target)
	})
	return exports, nil
}
