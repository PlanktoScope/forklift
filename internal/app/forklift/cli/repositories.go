package cli

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/core"
)

// Print

func PrintCachedRepo(indent int, cache core.Pather, repo *core.FSRepo) error {
	IndentedPrintf(indent, "Cached repo: %s\n", repo.Path())
	indent++

	IndentedPrintf(indent, "Forklift version: %s\n", repo.Def.ForkliftVersion)
	fmt.Println()

	IndentedPrintf(indent, "Version: %s\n", repo.Version)
	IndentedPrintf(indent, "Path in cache: %s\n", core.GetSubdirPath(cache, repo.FS.Path()))
	IndentedPrintf(indent, "Description: %s\n", repo.Def.Repo.Description)

	readme, err := repo.LoadReadme()
	if err != nil {
		return errors.Wrapf(
			err, "couldn't load readme file for repo %s@%s from cache", repo.Path(), repo.Version,
		)
	}
	IndentedPrintln(indent, "Readme:")
	const widthLimit = 100
	PrintReadme(indent+1, readme, widthLimit)
	return nil
}
