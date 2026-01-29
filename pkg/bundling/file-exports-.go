package bundling

import (
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/pkg/caching"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	fpkg "github.com/forklift-run/forklift/pkg/packaging"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
)

func exportLocalFile(
	resolved *fplt.ResolvedDepl,
	export fpkg.FileExportRes,
	exportPath string,
) error {
	if err := ffs.CopyFSFile(
		resolved.Pkg.FS, strings.TrimPrefix(export.Source, "/"), filepath.FromSlash(exportPath),
		export.Permissions,
	); err != nil {
		return errors.Wrapf(err, "couldn't export file from %s to %s", export.Source, exportPath)
	}
	return nil
}

func exportHTTPFile(
	export fpkg.FileExportRes,
	exportPath string,
	dlCache *caching.FSDownloadCache,
) error {
	sourcePath, err := dlCache.GetFilePath(export.URL)
	if err != nil {
		return errors.Wrapf(err, "couldn't determine cache path for HTTP download %s", export.URL)
	}
	if err := ffs.CopyFSFile(
		dlCache.FS, strings.TrimPrefix(strings.TrimPrefix(sourcePath, dlCache.FS.Path()), "/"),
		filepath.FromSlash(exportPath),
		export.Permissions,
	); err != nil {
		return errors.Wrapf(err, "couldn't export file from %s to %s", sourcePath, exportPath)
	}
	return nil
}
