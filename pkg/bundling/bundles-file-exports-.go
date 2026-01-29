package bundling

import (
	"path"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/pkg/caching"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	fpkg "github.com/forklift-run/forklift/pkg/packaging"
)

// exportsDirName is the name of the directory containing exported files for all package
// deployments, collected together.
const exportsDirName = "exports"

// FSBundle: File Exports

func (b *FSBundle) getExportsPath() string {
	return path.Join(b.FS.Path(), exportsDirName)
}

func (b *FSBundle) WriteFileExports(dlCache *caching.FSDownloadCache) error {
	if err := ffs.EnsureExists(filepath.FromSlash(b.getExportsPath())); err != nil {
		return errors.Wrapf(err, "couldn't make directory for all file exports")
	}
	for deplName := range b.Manifest.Deploys {
		resolved, err := b.LoadResolvedDepl(deplName)
		if err != nil {
			return errors.Wrapf(err, "couldn't resolve deployment %s", deplName)
		}
		exports, err := resolved.GetFileExports()
		if err != nil {
			return errors.Wrapf(err, "couldn't determine file exports for deployment %s", deplName)
		}
		for _, export := range exports {
			exportPath := path.Join(b.getExportsPath(), export.Target)
			if err := ffs.EnsureExists(filepath.FromSlash(path.Dir(exportPath))); err != nil {
				return errors.Wrapf(
					err, "couldn't make export directory %s in bundle", path.Dir(exportPath),
				)
			}
			switch export.SourceType {
			case fpkg.FileExportSourceTypeLocal:
				if err := exportLocalFile(resolved, export, exportPath); err != nil {
					return err
				}
			case fpkg.FileExportSourceTypeHTTP:
				if err := exportHTTPFile(export, exportPath, dlCache); err != nil {
					return err
				}
			case fpkg.FileExportSourceTypeHTTPArchive, fpkg.FileExportSourceTypeOCIImage:
				if err := exportArchiveFile(export, exportPath, dlCache); err != nil {
					return err
				}
			default:
				return errors.Errorf("unknown file export source type: %s", export.SourceType)
			}
		}
	}
	return nil
}
