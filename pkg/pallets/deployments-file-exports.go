package pallets

import (
	"cmp"
	"io/fs"
	"path"
	"slices"

	"github.com/pkg/errors"

	ffs "github.com/forklift-run/forklift/pkg/fs"
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

type InvalidFileExport struct {
	// Source path of the file export
	Source string
	// Target path of the file export
	Target string
	// Specific problem with the file export
	Err error
}

// CheckFileExports checks the validity of the source paths of all file exports from the package
// deployment.
// A non-nil error is only returned if file exports could not be checked.
func (d *ResolvedDepl) CheckFileExports() ([]InvalidFileExport, error) {
	exports, err := d.GetFileExports()
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't determine file exports for deployment %s", d.Name)
	}
	invalidFileExports := make([]InvalidFileExport, 0)
	for _, export := range exports {
		switch export.SourceType {
		default:
			// TODO: should we also check file exports from files in the cache of downloaded files?
			continue
		case fpkg.FileExportSourceTypeLocal, "":
		}
		sourcePath := cmp.Or(export.Source, export.Target)
		if err := checkFileOrSymlink(d.Pkg.FS, sourcePath); err != nil {
			invalidFileExports = append(
				invalidFileExports,
				InvalidFileExport{
					Source: sourcePath,
					Target: export.Target,
					Err:    err,
				},
			)
		}
	}
	if len(invalidFileExports) == 0 {
		return nil, nil
	}
	return invalidFileExports, nil
}

func checkFileOrSymlink(fsys ffs.PathedFS, file string) error {
	if _, err := fs.Stat(fsys, file); err == nil {
		return nil
	}
	// fs.Stat will return an error if the sourcePath exists but is a symlink pointing to a
	// nonexistent location. Really we want fs.Lstat (which is not implemented yet); until fs.Lstat
	// is implemented, when we get an error when we'll just check if a DirEntry exists for the path
	// (and if so, we'll assume the file is valid).
	dirEntries, err := fs.ReadDir(fsys, path.Dir(file))
	if err != nil {
		return err
	}
	for _, dirEntry := range dirEntries {
		if dirEntry.Name() == path.Base(file) {
			return nil
		}
	}
	return errors.Errorf(
		"couldn't find %s in %s", path.Base(file), path.Join(fsys.Path(), path.Dir(file)),
	)
}

// Checking

// CheckFileExports produces a map of all invalid file exports among all provided ResolvedDepl
// instances.
// A non-nil error is only returned if file exports could not be checked.
func CheckFileExports(
	depls []*ResolvedDepl,
) (invalidFileExports map[string][]InvalidFileExport, err error) {
	invalidFileExports = make(map[string][]InvalidFileExport)
	for _, depl := range depls {
		if invalidFileExports[depl.Name], err = depl.CheckFileExports(); err != nil {
			return nil, errors.Wrapf(err, "couldn't check file exports for deployment %s", depl.Name)
		}
		if len(invalidFileExports[depl.Name]) == 0 {
			delete(invalidFileExports, depl.Name)
		}
	}
	if len(invalidFileExports) == 0 {
		return nil, nil
	}
	return invalidFileExports, nil
}
