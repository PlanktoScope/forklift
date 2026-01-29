package forklift

import (
	"path"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	ffs "github.com/forklift-run/forklift/pkg/fs"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
)

func ListPalletFiles(pallet *fplt.FSPallet, pattern string) ([]string, error) {
	if pattern == "" {
		pattern = "**"
	}

	paths, err := doublestar.Glob(pallet.FS, pattern, doublestar.WithFilesOnly())
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't list files matching pattern %s in %s", pattern, pallet.FS.Path(),
		)
	}

	if len(paths) == 0 {
		if pattern != "**" {
			pattern = path.Join(pattern, "**")
		}
		subPaths, err := doublestar.Glob(pallet.FS, pattern, doublestar.WithFilesOnly())
		if err != nil {
			return paths, errors.Wrapf(
				err, "couldn't list files matching pattern %s in %s", pattern, pallet.FS.Path(),
			)
		}
		paths = append(paths, subPaths...)
	}

	return paths, nil
}

func GetFileLocation(pallet *fplt.FSPallet, filePath string) (string, error) {
	fsys, ok := pallet.FS.(*ffs.MergeFS)
	if !ok {
		return path.Join(pallet.FS.Path(), filePath), nil
	}

	resolved, err := fsys.Resolve(filePath)
	if err != nil {
		return "", errors.Wrapf(
			err, "couldn't resolve the location of file %s in %s", filePath, pallet.FS.Path(),
		)
	}
	return resolved, nil
}
