package forklift

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"

	ffs "github.com/forklift-run/forklift/pkg/fs"
)

func DownloadFile(ctx context.Context, url, outputPath string, hc *http.Client) error {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errors.Wrapf(err, "couldn't make http get request for %s", url)
	}
	res, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			// FIXME: handle this error better
			fmt.Fprintf(os.Stderr, "Error: couldn't close http response for %s\n", url)
		}
	}()

	_, err = io.Copy(file, res.Body)
	if err != nil {
		return errors.Wrapf(err, "couldn't download %s to %s", url, tmpPath)
	}

	if err = os.Rename(filepath.FromSlash(tmpPath), filepath.FromSlash(outputPath)); err != nil {
		return errors.Wrapf(
			err, "couldn't commit completed download from %s to %s", tmpPath, outputPath,
		)
	}
	return nil
}
