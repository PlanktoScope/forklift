package cli

import (
	"fmt"
	"io/fs"
	"path"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
)

func PrintPalletFiles(indent int, pallet *forklift.FSPallet, pattern string) error {
	if pattern == "" {
		pattern = "**"
	}

	paths, err := doublestar.Glob(pallet.FS, pattern, doublestar.WithFilesOnly())
	if err != nil {
		return errors.Wrapf(
			err, "couldn't list files matching pattern %s in %s", pattern, pallet.FS.Path(),
		)
	}

	if len(paths) == 0 {
		if pattern != "**" {
			pattern = path.Join(pattern, "**")
		}
		subPaths, err := doublestar.Glob(pallet.FS, pattern, doublestar.WithFilesOnly())
		if err != nil {
			return errors.Wrapf(
				err, "couldn't list files matching pattern %s in %s", pattern, pallet.FS.Path(),
			)
		}
		paths = append(paths, subPaths...)
	}

	for _, p := range paths {
		IndentedPrintln(indent, p)
	}
	return nil
}

func PrintFileLocation(
	pallet *forklift.FSPallet, cache forklift.PathedRepoCache, filePath string,
) error {
	fsys, ok := pallet.FS.(*forklift.MergeFS)
	if !ok {
		fmt.Println(path.Join(pallet.FS.Path(), filePath))
		return nil
	}

	resolved, err := fsys.Resolve(filePath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't resolve the location of file %s in %s", filePath, pallet.FS.Path(),
		)
	}
	fmt.Println(resolved)
	return nil
}

func PrintFile(
	pallet *forklift.FSPallet, cache forklift.PathedRepoCache, filePath string,
) error {
	data, err := fs.ReadFile(pallet.FS, filePath)
	if err != nil {
		return errors.Wrapf(err, "couldn't read file %s in %s", filePath, pallet.FS.Path())
	}

	fmt.Print(string(data))
	return nil
}
