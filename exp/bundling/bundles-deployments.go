package bundling

import (
	"path"
	"path/filepath"
	"slices"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	ffs "github.com/forklift-run/forklift/exp/fs"
	fplt "github.com/forklift-run/forklift/exp/pallets"
	"github.com/forklift-run/forklift/exp/structures"
)

// FSBundle: Deployments

func (b *FSBundle) AddResolvedDepl(depl *fplt.ResolvedDepl) (err error) {
	b.Manifest.Deploys[depl.Name] = depl.Depl.Decl
	downloads := BundleDeplDownloads{}
	if downloads.HTTPFile, err = depl.GetHTTPFileDownloadURLs(); err != nil {
		return errors.Wrapf(
			err, "couldn't determine HTTP file downloads for deployment %s", depl.Depl.Name,
		)
	}
	if downloads.OCIImage, err = depl.GetOCIImageDownloadNames(); err != nil {
		return errors.Wrapf(
			err, "couldn't determine OCI image downloads for deployment %s", depl.Depl.Name,
		)
	}
	b.Manifest.Downloads[depl.Name] = downloads

	if err = ffs.CopyFS(depl.Pkg.FS, filepath.FromSlash(
		path.Join(b.getPackagesPath(), depl.Decl.Package),
	)); err != nil {
		return errors.Wrapf(
			err, "couldn't bundle files from package %s for deployment %s from %s",
			depl.Pkg.Path(), depl.Depl.Name, depl.Pkg.FS.Path(),
		)
	}

	exports := BundleDeplExports{}
	if exports.File, err = depl.GetFileExportTargets(); err != nil {
		return errors.Wrapf(err, "couldn't determine file exports of deployment %s", depl.Depl.Name)
	}
	definesComposeApp, err := depl.DefinesComposeApp()
	if err != nil {
		return errors.Wrapf(
			err, "couldn't check deployment %s for a Compose app", depl.Depl.Name,
		)
	}
	if definesComposeApp {
		exports.ComposeApp, err = b.makeComposeAppSummary(depl)
		if err != nil {
			return errors.Wrap(err, "couldn't make summary of Compose app definition")
		}
	}
	b.Manifest.Exports[depl.Name] = exports

	allOCIImages := make(structures.Set[string])
	allOCIImages.Add(downloads.OCIImage...)
	allOCIImages.Add(exports.ComposeApp.Images...)
	downloads.OCIImage = slices.Sorted(allOCIImages.All())
	b.Manifest.Downloads[depl.Name] = downloads

	if downloads.Empty() {
		delete(b.Manifest.Downloads, depl.Name)
	}
	if exports.Empty() {
		delete(b.Manifest.Exports, depl.Name)
	}
	return nil
}

func (b *FSBundle) makeComposeAppSummary(depl *fplt.ResolvedDepl) (BundleDeplComposeApp, error) {
	bundlePkg, err := b.LoadFSPkg(depl.Decl.Package, "")
	if err != nil {
		return BundleDeplComposeApp{}, errors.Wrapf(
			err, "couldn't load bundled package %s", depl.Pkg.Path(),
		)
	}
	depl = &fplt.ResolvedDepl{
		Depl:   depl.Depl,
		PkgReq: depl.PkgReq,
		Pkg:    bundlePkg,
	}

	return makeComposeAppSummary(depl, b.FS.Path())
}

func (b *FSBundle) LoadDepl(name string) (fplt.Depl, error) {
	depl, ok := b.Manifest.Deploys[name]
	if !ok {
		return fplt.Depl{}, errors.Errorf("bundle does not contain package deployment %s", name)
	}
	return fplt.Depl{
		Name: name,
		Decl: depl,
	}, nil
}

func (b *FSBundle) LoadDepls(searchPattern string) ([]fplt.Depl, error) {
	deplNames := make([]string, 0, len(b.Manifest.Deploys))
	for deplName := range b.Manifest.Deploys {
		match, err := doublestar.Match(searchPattern, deplName)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't search for package deployment configs matching %s", searchPattern,
			)
		}
		if !match {
			continue
		}
		deplNames = append(deplNames, deplName)
	}
	slices.Sort(deplNames)
	depls := make([]fplt.Depl, 0, len(deplNames))
	for _, deplName := range deplNames {
		depl, err := b.LoadDepl(deplName)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load package deployment %s from bundle", deplName)
		}
		depls = append(depls, depl)
	}
	return depls, nil
}

func (b *FSBundle) LoadResolvedDepl(name string) (depl *fplt.ResolvedDepl, err error) {
	resolved := &fplt.ResolvedDepl{
		Depl: fplt.Depl{
			Name: name,
			Decl: b.Manifest.Deploys[name],
		},
	}
	pkgPath := b.Manifest.Deploys[name].Package
	if resolved.PkgReq, err = b.LoadPkgReq(pkgPath); err != nil {
		return depl, err
	}
	if resolved.Pkg, err = b.LoadFSPkg(pkgPath, ""); err != nil {
		return depl, errors.Wrapf(err, "couldn't load package deployment %s from bundle", pkgPath)
	}
	return resolved, nil
}
