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
