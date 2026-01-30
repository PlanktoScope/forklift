package cli

import (
	"fmt"
	"io"
	"path"
	"slices"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/internal/app/forklift"
	"github.com/forklift-run/forklift/pkg/caching"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	fpkg "github.com/forklift-run/forklift/pkg/packaging"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
)

func FprintPkg(indent int, out io.Writer, cache caching.PathedPalletCache, pkg *fpkg.FSPkg) {
	IndentedFprintf(indent, out, "Package: %s\n", pkg.Path())
	indent++

	fprintPkgPallet(indent, out, cache, pkg)
	if ffs.CoversPath(cache, pkg.FS.Path()) {
		IndentedFprintf(indent, out, "Path in cache: %s\n", ffs.GetSubdirPath(cache, pkg.FS.Path()))
	} else {
		IndentedFprintf(indent, out, "Absolute path (replacing any cached copy): %s\n", pkg.FS.Path())
	}

	fmt.Println()
	FprintPkgSpec(indent, out, pkg.Decl.Package)
	_, _ = fmt.Fprintln(out)
	FprintDeplSpec(indent, out, pkg.Decl.Deployment)
	_, _ = fmt.Fprintln(out)
	FprintFeatureSpecs(indent, out, pkg.Decl.Features)
}

func fprintPkgPallet(indent int, out io.Writer, cache caching.PathedPalletCache, pkg *fpkg.FSPkg) {
	IndentedFprintf(indent, out, "Provided by pallet: %s\n", pkg.FSPkgTree.Path())
	indent++

	if ffs.CoversPath(cache, pkg.FS.Path()) {
		IndentedFprintf(indent, out, "Version: %s\n", pkg.FSPkgTree.Version)
	} else {
		IndentedFprintf(
			indent, out, "Absolute path (replacing any cached copy): %s\n", pkg.FSPkgTree.FS.Path(),
		)
	}
}

func FprintPkgSpec(indent int, out io.Writer, spec fpkg.PkgSpec) {
	IndentedFprintf(indent, out, "Description: %s\n", spec.Description)

	IndentedFprint(indent, out, "Maintainers:")
	if len(spec.Maintainers) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	for _, maintainer := range spec.Maintainers {
		fprintMaintainer(indent+1, out, maintainer)
	}

	if spec.License != "" {
		IndentedFprintf(indent, out, "License: %s\n", spec.License)
	} else {
		IndentedFprint(indent, out, "License: (custom license)\n")
	}

	IndentedFprint(indent, out, "Sources:")
	if len(spec.Sources) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	for _, source := range spec.Sources {
		BulletedFprintln(indent+1, out, source)
	}
}

func fprintMaintainer(indent int, out io.Writer, maintainer fpkg.PkgMaintainer) {
	if maintainer.Email != "" {
		BulletedFprintf(indent, out, "%s <%s>\n", maintainer.Name, maintainer.Email)
	} else {
		BulletedFprintln(indent, out, maintainer.Name)
	}
}

func FprintDeplSpec(indent int, out io.Writer, spec fpkg.PkgDeplSpec) {
	IndentedFprint(indent, out, "Deployment:\n")
	indent++

	IndentedFprint(indent, out, "Compose files:")
	if len(spec.ComposeFiles) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	for _, file := range spec.ComposeFiles {
		BulletedFprintln(indent+1, out, file)
	}

	fprintFileExports(indent, out, spec.Provides.FileExports)
}

func fprintFileExports(indent int, out io.Writer, fileExports []fpkg.FileExportRes) {
	IndentedFprint(indent, out, "File exports:")
	if len(fileExports) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	indent++
	for _, fileExport := range fileExports {
		switch fileExport.SourceType {
		case fpkg.FileExportSourceTypeLocal:
			fprintFileExportLocal(indent, out, fileExport)
		case fpkg.FileExportSourceTypeHTTP:
			fprintFileExportHTTP(indent, out, fileExport)
		case fpkg.FileExportSourceTypeHTTPArchive:
			fprintFileExportHTTPArchive(indent, out, fileExport)
		case fpkg.FileExportSourceTypeOCIImage:
			fprintFileExportOCIImage(indent, out, fileExport)
		default:
			BulletedFprintf(
				indent, out, "Unknown source type %s: %+v\n", fileExport.SourceType, fileExport,
			)
		}
	}
}

func fprintFileExportLocal(indent int, out io.Writer, fileExport fpkg.FileExportRes) {
	BulletedFprint(indent, out, "Export from the package's local directory")
	indent++
	if fileExport.Description == "" {
		_, _ = fmt.Fprintln(out)
	} else {
		_, _ = fmt.Fprintln(out, ":")
		IndentedFprintln(indent+1, out, fileExport.Description)
	}
	if fileExport.Source == fileExport.Target {
		IndentedFprintf(indent, out, "Export: %s\n", fileExport.Target)
		return
	}
	IndentedFprintf(indent, out, "From file: %s\n", fileExport.Source)
	IndentedFprintf(indent, out, "Export as: %s\n", fileExport.Target)
}

func fprintFileExportHTTP(indent int, out io.Writer, fileExport fpkg.FileExportRes) {
	BulletedFprint(indent, out, "Export from an HTTP download")
	indent++
	if fileExport.Description == "" {
		_, _ = fmt.Fprintln(out)
	} else {
		_, _ = fmt.Fprintln(out, ":")
		IndentedFprintln(indent+1, out, fileExport.Description)
	}
	IndentedFprintf(indent, out, "From file: %s\n", fileExport.URL)
	IndentedFprintf(indent, out, "Export as: %s\n", fileExport.Target)
}

func fprintFileExportHTTPArchive(indent int, out io.Writer, fileExport fpkg.FileExportRes) {
	BulletedFprint(indent, out, "Export from an HTTP archive download")
	indent++
	if fileExport.Description == "" {
		_, _ = fmt.Fprintln(out)
	} else {
		_, _ = fmt.Fprintln(out, ":")
		IndentedFprintln(indent+1, out, fileExport.Description)
	}
	IndentedFprintf(indent, out, "From file: [%s]/%s\n", fileExport.URL, fileExport.Source)
	IndentedFprintf(indent, out, "Export as: %s\n", fileExport.Target)
}

func fprintFileExportOCIImage(indent int, out io.Writer, fileExport fpkg.FileExportRes) {
	BulletedFprint(indent, out, "Export from a Docker/OCI image")
	indent++
	if fileExport.Description == "" {
		_, _ = fmt.Fprintln(out)
	} else {
		_, _ = fmt.Fprintln(out, ":")
		IndentedFprintln(indent+1, out, fileExport.Description)
	}
	IndentedFprintf(indent, out, "From file: [%s]/%s\n", fileExport.URL, fileExport.Source)
	IndentedFprintf(indent, out, "Export as: %s\n", fileExport.Target)
}

func FprintFeatureSpecs(indent int, out io.Writer, features map[string]fpkg.PkgFeatureSpec) {
	IndentedFprint(indent, out, "Optional features:")
	names := make([]string, 0, len(features))
	for name := range features {
		names = append(names, name)
	}
	slices.Sort(names)
	if len(names) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	indent++

	for _, name := range names {
		FprintFeatureSpec(indent, out, name, features[name])
	}
}

func FprintFeatureSpec(indent int, out io.Writer, name string, spec fpkg.PkgFeatureSpec) {
	IndentedFprintf(indent, out, "%s:\n", name)
	indent++

	IndentedFprintf(indent, out, "Description: %s\n", spec.Description)

	IndentedFprint(indent, out, "Compose files:")
	if len(spec.ComposeFiles) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	for _, file := range spec.ComposeFiles {
		BulletedFprintln(indent+1, out, file)
	}

	fprintFileExports(indent, out, spec.Provides.FileExports)
}

// Pallet packages

func FprintPalletPkgs(
	indent int, out io.Writer, pallet *fplt.FSPallet, loader fplt.FSPkgLoader,
) error {
	reqs, err := pallet.LoadFSPalletReqs("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't identify pkg trees in pallet %s", pallet.FS.Path())
	}

	// List packages provided by required pkg trees
	pkgs := make([]*fpkg.FSPkg, 0)
	for _, req := range reqs {
		pkgTreeCachePath := req.GetQueryPath()
		loaded, err := loader.LoadFSPkgs(path.Join(pkgTreeCachePath, "**"))
		if err != nil {
			return errors.Wrapf(
				err, "couldn't load packages from pkg tree cached at %s", pkgTreeCachePath,
			)
		}
		pkgs = append(pkgs, loaded...)
	}

	// List local packages provided by the pallet itself
	loaded, err := pallet.LoadFSPkgs("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't load local packages defined by pallet at %s", pallet.Path())
	}
	for _, pkg := range loaded {
		pkg.ParentPath = "/"
	}
	pkgs = append(pkgs, loaded...)

	slices.SortFunc(pkgs, fpkg.CompareFSPkgs)
	for _, pkg := range pkgs {
		IndentedFprintf(indent, out, "%s\n", pkg.Path())
	}
	return nil
}

func FprintPkgLocation(
	out io.Writer, pallet *fplt.FSPallet, cache caching.PathedPalletCache, pkgPath string,
) error {
	overlayCache, err := forklift.MakeOverlayCache(pallet, cache)
	if err != nil {
		return err
	}
	pkg, _, err := fplt.LoadRequiredFSPkg(pallet, overlayCache, pkgPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't look up information about package %s in pallet %s", pkgPath, pallet.FS.Path(),
		)
	}
	fsys, ok := pkg.FS.(*ffs.MergeFS)
	if !ok {
		_, _ = fmt.Fprintln(out, pkg.FS.Path())
		return nil
	}

	resolved, err := fsys.Resolve("forklift-package.yml")
	if err != nil {
		return errors.Wrapf(err, "couldn't resolve the location of package %s", pkgPath)
	}
	_, _ = fmt.Fprintln(out, path.Dir(resolved))
	return nil
}

func FprintPkgInfo(
	indent int, out io.Writer,
	pallet *fplt.FSPallet, cache caching.PathedPalletCache, pkgPath string,
) error {
	overlayCache, err := forklift.MakeOverlayCache(pallet, cache)
	if err != nil {
		return err
	}
	pkg, _, err := fplt.LoadRequiredFSPkg(pallet, overlayCache, pkgPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't look up information about package %s in pallet %s", pkgPath, pallet.FS.Path(),
		)
	}
	FprintPkg(indent, out, cache, pkg)
	return nil
}
