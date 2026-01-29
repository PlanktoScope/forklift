package forklift

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/internal/clients/crane"
	ffs "github.com/forklift-run/forklift/pkg/fs"
)

func DownloadOCIImage(ctx context.Context, imageName, outputPath, platform string) error {
	if err := ffs.EnsureExists(filepath.FromSlash(path.Dir(outputPath))); err != nil {
		return err
	}
	tmpPath := outputPath + ".fkldownload"
	file, err := os.Create(filepath.FromSlash(tmpPath))
	if err != nil {
		return errors.Wrapf(err, "couldn't create temporary download file at %s", tmpPath)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// FIXME: handle this error better
			fmt.Fprintf(os.Stderr, "Error: couldn't close temporary download file %s\n", tmpPath)
		}
	}()

	if err = crane.ExportOCIImage(ctx, imageName, file, platform); err != nil {
		return errors.Wrapf(err, "couldn't download and export image as a tarball: %s", imageName)
	}

	if err = os.Rename(filepath.FromSlash(tmpPath), filepath.FromSlash(outputPath)); err != nil {
		return errors.Wrapf(
			err, "couldn't commit completed download from %s to %s", tmpPath, outputPath,
		)
	}
	return nil
}
