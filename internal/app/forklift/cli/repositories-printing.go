package cli

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/pkg/core"
	ffs "github.com/forklift-run/forklift/pkg/fs"
)

func FprintCachedPkgTree(
	indent int, out io.Writer, cache ffs.Pather, pkgTree *core.FSPkgTree, printHeader bool,
) error {
	if printHeader {
		IndentedFprintf(indent, out, "Cached pkg tree: %s\n", pkgTree.Path())
		indent++
	}

	IndentedFprintf(indent, out, "Forklift version: %s\n", pkgTree.Decl.ForkliftVersion)
	_, _ = fmt.Fprintln(out)

	IndentedFprintf(indent, out, "Version: %s\n", pkgTree.Version)
	if ffs.CoversPath(cache, pkgTree.FS.Path()) {
		IndentedFprintf(indent, out, "Path in cache: %s\n", ffs.GetSubdirPath(cache, pkgTree.FS.Path()))
	} else {
		// Note: this is used when the pkg tree is replaced by an overlay from outside the cache
		IndentedFprintf(indent, out, "Absolute path (replacing any cached copy): %s\n", pkgTree.FS.Path())
	}
	IndentedFprintf(indent, out, "Description: %s\n", pkgTree.Decl.PkgTree.Description)

	if err := fprintReadme(indent, out, pkgTree); err != nil {
		return errors.Wrapf(
			err,
			"couldn't preview readme file for pkg tree %s@%s from cache",
			pkgTree.Path(),
			pkgTree.Version,
		)
	}

	_, _ = fmt.Fprintln(out)
	if err := fprintPkgTreePkgs(indent, out, pkgTree); err != nil {
		return errors.Wrapf(err, "couldn't list packages provided by pkg tree %s", pkgTree.Path())
	}
	return nil
}

type readmeLoader interface {
	LoadReadme() ([]byte, error)
}

func fprintReadme(indent int, out io.Writer, loader readmeLoader) error {
	readme, err := loader.LoadReadme()
	if err != nil {
		return errors.Wrapf(err, "couldn't load readme file")
	}
	const widthLimit = 100
	const lengthLimit = 10
	IndentedFprintf(indent, out, "Readme (first %d lines):\n", lengthLimit)
	PrintMarkdown(indent+1, readme, widthLimit, lengthLimit)
	return nil
}

func fprintPkgTreePkgs(indent int, out io.Writer, pkgTree *core.FSPkgTree) error {
	IndentedFprint(indent, out, "Packages:")

	pkgs, err := pkgTree.LoadFSPkgs("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't load packages from pkg tree %s", pkgTree.Path())
	}
	slices.SortFunc(pkgs, func(a, b *core.FSPkg) int {
		return core.ComparePkgs(a.Pkg, b.Pkg)
	})

	if len(pkgs) == 0 {
		_, _ = fmt.Fprint(out, " (none)")
	}
	_, _ = fmt.Fprintln(out)
	indent += 1
	for _, pkg := range pkgs {
		IndentedFprintf(indent, out, "...%s: ", strings.TrimPrefix(pkg.Path(), pkgTree.Path()))

		names := make([]string, 0, len(pkg.Decl.Features))
		for name := range pkg.Decl.Features {
			names = append(names, name)
		}
		slices.Sort(names)

		if len(names) == 0 {
			_, _ = fmt.Fprintln(out, "(no optional features)")
			continue
		}
		_, _ = fmt.Fprintf(out, "[%s]\n", strings.Join(names, ", "))
	}
	return nil
}
