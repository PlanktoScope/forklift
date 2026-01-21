package cache

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/forklift-run/forklift/internal/app/forklift"
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

// del-dl

func delDlAction(c *cli.Context) error {
	workspace, err := forklift.LoadWorkspace(c.String("workspace"))
	if err != nil {
		return err
	}

	cache, err := workspace.GetDownloadCache()
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Clearing downloads cache...")
	if err = cache.Remove(); err != nil {
		return errors.Wrap(err, "couldn't clear downloads cache")
	}
	return nil
}
