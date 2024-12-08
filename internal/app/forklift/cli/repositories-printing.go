package cli

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/core"
)

func FprintCachedRepo(
	indent int, out io.Writer, cache core.Pather, repo *core.FSRepo, printHeader bool,
) error {
	if printHeader {
		IndentedFprintf(indent, out, "Cached repo: %s\n", repo.Path())
		indent++
	}

	IndentedFprintf(indent, out, "Forklift version: %s\n", repo.Def.ForkliftVersion)
	_, _ = fmt.Fprintln(out)

	IndentedFprintf(indent, out, "Version: %s\n", repo.Version)
	if core.CoversPath(cache, repo.FS.Path()) {
		IndentedFprintf(indent, out, "Path in cache: %s\n", core.GetSubdirPath(cache, repo.FS.Path()))
	} else {
		// Note: this is used when the repo is replaced by an overlay from outside the cache
		IndentedFprintf(indent, out, "Absolute path (replacing any cached copy): %s\n", repo.FS.Path())
	}
	IndentedFprintf(indent, out, "Description: %s\n", repo.Def.Repo.Description)

	if err := fprintReadme(indent, out, repo); err != nil {
		return errors.Wrapf(
			err, "couldn't preview readme file for repo %s@%s from cache", repo.Path(), repo.Version,
		)
	}

	_, _ = fmt.Fprintln(out)
	if err := fprintRepoPkgs(indent, out, repo); err != nil {
		return errors.Wrapf(err, "couldn't list packages provided by repo %s", repo.Path())
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

func fprintRepoPkgs(indent int, out io.Writer, repo *core.FSRepo) error {
	IndentedFprint(indent, out, "Packages:")

	pkgs, err := repo.LoadFSPkgs("**")
	if err != nil {
		return errors.Wrapf(err, "couldn't load packages from repo %s", repo.Path())
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
		BulletedFprintf(indent, out, "%s: ", pkg.Path())

		names := make([]string, 0, len(pkg.Def.Features))
		for name := range pkg.Def.Features {
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
