package cli

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/core"
)

func PrintCachedRepo(indent int, cache core.Pather, repo *core.FSRepo, printHeader bool) error {
	if printHeader {
		IndentedPrintf(indent, "Cached repo: %s\n", repo.Path())
		indent++
	}

	IndentedPrintf(indent, "Forklift version: %s\n", repo.Def.ForkliftVersion)
	fmt.Println()

	IndentedPrintf(indent, "Version: %s\n", repo.Version)
	if core.CoversPath(cache, repo.FS.Path()) {
		IndentedPrintf(indent, "Path in cache: %s\n", core.GetSubdirPath(cache, repo.FS.Path()))
	} else {
		// Note: this is used when the repo is replaced by an overlay from outside the cache
		IndentedPrintf(indent, "Absolute path (replacing any cached copy): %s\n", repo.FS.Path())
	}
	IndentedPrintf(indent, "Description: %s\n", repo.Def.Repo.Description)

	if err := printReadme(indent, repo); err != nil {
		return errors.Wrapf(
			err, "couldn't preview readme file for repo %s@%s from cache", repo.Path(), repo.Version,
		)
	}
	return nil
}

type readmeLoader interface {
	LoadReadme() ([]byte, error)
}

func printReadme(indent int, loader readmeLoader) error {
	readme, err := loader.LoadReadme()
	if err != nil {
		return errors.Wrapf(err, "couldn't load readme file")
	}
	const widthLimit = 100
	const lengthLimit = 10
	IndentedPrintf(indent, "Readme (first %d lines):\n", lengthLimit)
	PrintMarkdown(indent+1, readme, widthLimit, lengthLimit)
	return nil
}
