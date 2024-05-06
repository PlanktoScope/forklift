package cache

import (
	"fmt"
	"io/fs"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
)

// ls-dl

func lsDlAction(c *cli.Context) error {
	workspace, err := forklift.LoadWorkspace(c.String("workspace"))
	if err != nil {
		return err
	}

	cache, err := workspace.GetDownloadCache()
	if err != nil {
		return err
	}
	if !cache.Exists() {
		return errors.New(
			"you first need to cache any downloads specified by your pallet with " +
				"`forklift plt cache-dl`",
		)
	}

	// TODO: add a --pattern cli flag for the pattern
	if err = doublestar.GlobWalk(cache.FS, "**", func(path string, d fs.DirEntry) error {
		if d.IsDir() {
			return nil
		}
		fmt.Println(path)
		return nil
	}); err != nil {
		return errors.Wrapf(err, "couldn't list files in download cache %s", cache.FS.Path())
	}
	return nil
}
