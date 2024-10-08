package cli

import (
	"fmt"
	"path"
	"slices"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/pkg/core"
)

func PrintPkg(indent int, cache forklift.PathedRepoCache, pkg *core.FSPkg) {
	IndentedPrintf(indent, "Package: %s\n", pkg.Path())
	indent++

	printPkgRepo(indent, cache, pkg)
	if core.CoversPath(cache, pkg.FS.Path()) {
		IndentedPrintf(indent, "Path in cache: %s\n", core.GetSubdirPath(cache, pkg.FS.Path()))
	} else {
		IndentedPrintf(indent, "Absolute path (replacing any cached copy): %s\n", pkg.FS.Path())
	}

	PrintPkgSpec(indent, pkg.Def.Package)
	fmt.Println()
	PrintDeplSpec(indent, pkg.Def.Deployment)
	fmt.Println()
	PrintFeatureSpecs(indent, pkg.Def.Features)
}

func printPkgRepo(indent int, cache forklift.PathedRepoCache, pkg *core.FSPkg) {
	IndentedPrintf(indent, "Provided by repo: %s\n", pkg.Repo.Path())
	indent++

	if core.CoversPath(cache, pkg.FS.Path()) {
		IndentedPrintf(indent, "Version: %s\n", pkg.Repo.Version)
	} else {
		IndentedPrintf(
			indent, "Absolute path (replacing any cached copy): %s\n", pkg.Repo.FS.Path(),
		)
	}

	IndentedPrintf(indent, "Description: %s\n", pkg.Repo.Def.Repo.Description)
}

func PrintPkgSpec(indent int, spec core.PkgSpec) {
	IndentedPrintf(indent, "Description: %s\n", spec.Description)

	IndentedPrint(indent, "Maintainers:")
	if len(spec.Maintainers) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for _, maintainer := range spec.Maintainers {
		printMaintainer(indent+1, maintainer)
	}

	if spec.License != "" {
		IndentedPrintf(indent, "License: %s\n", spec.License)
	} else {
		IndentedPrintf(indent, "License: (custom license)\n")
	}

	IndentedPrint(indent, "Sources:")
	if len(spec.Sources) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for _, source := range spec.Sources {
		BulletedPrintln(indent+1, source)
	}
}

func printMaintainer(indent int, maintainer core.PkgMaintainer) {
	if maintainer.Email != "" {
		BulletedPrintf(indent, "%s <%s>\n", maintainer.Name, maintainer.Email)
	} else {
		BulletedPrintln(indent, maintainer.Name)
	}
}

func PrintDeplSpec(indent int, spec core.PkgDeplSpec) {
	IndentedPrintf(indent, "Deployment:\n")
	indent++

	IndentedPrintf(indent, "Compose files:")
	if len(spec.ComposeFiles) == 0 {
		fmt.Printf(" (none)")
	}
	fmt.Println()
	for _, file := range spec.ComposeFiles {
		BulletedPrintln(indent+1, file)
	}

	printFileExports(indent, spec.Provides.FileExports)
}

func printFileExports(indent int, fileExports []core.FileExportRes) {
	IndentedPrint(indent, "File exports:")
	if len(fileExports) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++
	for _, fileExport := range fileExports {
		switch fileExport.SourceType {
		case core.FileExportSourceTypeLocal:
			printFileExportLocal(indent, fileExport)
		case core.FileExportSourceTypeHTTP:
			printFileExportHTTP(indent, fileExport)
		case core.FileExportSourceTypeHTTPArchive:
			printFileExportHTTPArchive(indent, fileExport)
		case core.FileExportSourceTypeOCIImage:
			printFileExportOCIImage(indent, fileExport)
		default:
			BulletedPrintf(indent, "Unknown source type %s: %+v\n", fileExport.SourceType, fileExport)
		}
	}
}

func printFileExportLocal(indent int, fileExport core.FileExportRes) {
	BulletedPrintf(indent, "Export from the package's local directory")
	indent++
	if fileExport.Description == "" {
		fmt.Println()
	} else {
		fmt.Println(":")
		IndentedPrintln(indent+1, fileExport.Description)
	}
	if fileExport.Source == fileExport.Target {
		IndentedPrintf(indent, "Export: %s\n", fileExport.Target)
		return
	}
	IndentedPrintf(indent, "From file: %s\n", fileExport.Source)
	IndentedPrintf(indent, "Export as: %s\n", fileExport.Target)
}

func printFileExportHTTP(indent int, fileExport core.FileExportRes) {
	BulletedPrintf(indent, "Export from an HTTP download")
	indent++
	if fileExport.Description == "" {
		fmt.Println()
	} else {
		fmt.Println(":")
		IndentedPrintln(indent+1, fileExport.Description)
	}
	IndentedPrintf(indent, "From file: %s\n", fileExport.URL)
	IndentedPrintf(indent, "Export as: %s\n", fileExport.Target)
}

func printFileExportHTTPArchive(indent int, fileExport core.FileExportRes) {
	BulletedPrintf(indent, "Export from an HTTP archive download")
	indent++
	if fileExport.Description == "" {
		fmt.Println()
	} else {
		fmt.Println(":")
		IndentedPrintln(indent+1, fileExport.Description)
	}
	IndentedPrintf(indent, "From file: [%s]/%s\n", fileExport.URL, fileExport.Source)
	IndentedPrintf(indent, "Export as: %s\n", fileExport.Target)
}

func printFileExportOCIImage(indent int, fileExport core.FileExportRes) {
	BulletedPrintf(indent, "Export from a Docker/OCI image")
	indent++
	if fileExport.Description == "" {
		fmt.Println()
	} else {
		fmt.Println(":")
		IndentedPrintln(indent+1, fileExport.Description)
	}
	IndentedPrintf(indent, "From file: [%s]/%s\n", fileExport.URL, fileExport.Source)
	IndentedPrintf(indent, "Export as: %s\n", fileExport.Target)
}

func PrintFeatureSpecs(indent int, features map[string]core.PkgFeatureSpec) {
	IndentedPrint(indent, "Optional features:")
	names := make([]string, 0, len(features))
	for name := range features {
		names = append(names, name)
	}
	slices.Sort(names)
	if len(names) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, name := range names {
		PrintFeatureSpec(indent, name, features[name])
	}
}

func PrintFeatureSpec(indent int, name string, spec core.PkgFeatureSpec) {
	IndentedPrintf(indent, "%s:\n", name)
	indent++

	IndentedPrintf(indent, "Description: %s\n", spec.Description)

	IndentedPrintf(indent, "Compose files:")
	if len(spec.ComposeFiles) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for _, file := range spec.ComposeFiles {
		BulletedPrintln(indent+1, file)
	}

	printFileExports(indent, spec.Provides.FileExports)
}

// Pallet packages

func PrintPalletPkgs(indent int, pallet *forklift.FSPallet, loader forklift.FSPkgLoader) error {
	reqs, err := pallet.LoadFSRepoReqs("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't identify repos in pallet %s", pallet.FS.Path())
	}

	// List packages provided by required repos
	pkgs := make([]*core.FSPkg, 0)
	for _, req := range reqs {
		repoCachePath := req.GetCachePath()
		loaded, err := loader.LoadFSPkgs(path.Join(repoCachePath, "**"))
		if err != nil {
			return errors.Wrapf(err, "couldn't load packages from repo cached at %s", repoCachePath)
		}
		pkgs = append(pkgs, loaded...)
	}

	// List local packages provided by the pallet itself
	loaded, err := pallet.LoadFSPkgs("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't load local packages defined by pallet at %s", pallet.Path())
	}
	for _, pkg := range loaded {
		pkg.Repo.Def.Repo.Path = "/"
		pkg.RepoPath = "/"
	}
	pkgs = append(pkgs, loaded...)

	slices.SortFunc(pkgs, func(a, b *core.FSPkg) int {
		return core.ComparePkgs(a.Pkg, b.Pkg)
	})
	for _, pkg := range pkgs {
		IndentedPrintf(indent, "%s\n", pkg.Path())
	}
	return nil
}

func PrintPkgLocation(
	pallet *forklift.FSPallet, cache forklift.PathedRepoCache, pkgPath string,
) error {
	pkg, _, err := forklift.LoadRequiredFSPkg(pallet, cache, pkgPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't look up information about package %s in pallet %s", pkgPath, pallet.FS.Path(),
		)
	}
	fsys, ok := pkg.FS.(*forklift.MergeFS)
	if !ok {
		fmt.Println(pkg.FS.Path())
		return nil
	}

	resolved, err := fsys.Resolve("forklift-package.yml")
	if err != nil {
		return errors.Wrapf(err, "couldn't resolve the location of package %s", pkgPath)
	}
	fmt.Println(path.Dir(resolved))
	return nil
}

func PrintPkgInfo(
	indent int, pallet *forklift.FSPallet, cache forklift.PathedRepoCache, pkgPath string,
) error {
	pkg, _, err := forklift.LoadRequiredFSPkg(pallet, cache, pkgPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't look up information about package %s in pallet %s", pkgPath, pallet.FS.Path(),
		)
	}
	PrintPkg(indent, cache, pkg)
	return nil
}
