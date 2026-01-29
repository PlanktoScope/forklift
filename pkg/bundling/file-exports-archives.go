package bundling

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/h2non/filetype"
	ftt "github.com/h2non/filetype/types"
	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/pkg/caching"
	ffs "github.com/forklift-run/forklift/pkg/fs"
	fpkg "github.com/forklift-run/forklift/pkg/packaging"
)

func exportArchiveFile(
	export fpkg.FileExportRes, exportPath string, dlCache *caching.FSDownloadCache,
) error {
	kind, err := determineArchiveType(export, dlCache)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't determine file type of cached download archive %s", export.URL,
		)
	}

	var archiveFile fs.File
	switch export.SourceType {
	default:
		return errors.Errorf("couldn't open downloaded archive of type %s", export.SourceType)
	case fpkg.FileExportSourceTypeHTTPArchive:
		if archiveFile, err = dlCache.OpenFile(export.URL); err != nil {
			return errors.Wrapf(err, "couldn't open cached http download archive %s", export.URL)
		}
	case fpkg.FileExportSourceTypeOCIImage:
		if archiveFile, err = dlCache.OpenOCIImage(export.URL); err != nil {
			return errors.Wrapf(err, "couldn't open cached oci image download tarball %s", export.URL)
		}
	}
	defer func() {
		if err := archiveFile.Close(); err != nil {
			// TODO: handle this error more rigorously
			fmt.Fprintf(os.Stderr, "Error: couldn't close cached download archive %s\n", export.URL)
		}
	}()

	var archiveReader *tar.Reader
	switch kind.MIME.Value {
	case "application/x-tar":
		archiveReader = tar.NewReader(archiveFile)
	case "application/gzip":
		uncompressed, err := gzip.NewReader(archiveFile)
		if err != nil {
			return errors.Wrapf(err, "couldn't create a gzip decompressor for %s", export.URL)
		}
		// TODO: check to ensure that the uncompressed file is actually a tar archive
		defer func() {
			_ = uncompressed.Close()
		}()
		archiveReader = tar.NewReader(uncompressed)
	default:
		return errors.Errorf(
			"unrecognized archive file type: %s (.%s)", kind.MIME.Value, kind.Extension,
		)
	}
	if err = extractFromArchive(
		archiveReader, export.Source, exportPath, export.Permissions,
	); err != nil {
		return errors.Wrapf(
			err, "couldn't extract %s from cached download archive %s to %s",
			export.Source, export.URL, exportPath,
		)
	}
	return nil
}

func determineArchiveType(
	export fpkg.FileExportRes, dlCache *caching.FSDownloadCache,
) (ft ftt.Type, err error) {
	var archiveFile fs.File
	switch export.SourceType {
	default:
		return filetype.Unknown, errors.Errorf(
			"couldn't open downloaded archive of type %s", export.SourceType,
		)
	case fpkg.FileExportSourceTypeHTTPArchive:
		if archiveFile, err = dlCache.OpenFile(export.URL); err != nil {
			return filetype.Unknown, errors.Wrapf(
				err, "couldn't open cached http download archive %s", export.URL,
			)
		}
	case fpkg.FileExportSourceTypeOCIImage:
		if archiveFile, err = dlCache.OpenOCIImage(export.URL); err != nil {
			return filetype.Unknown, errors.Wrapf(
				err, "couldn't open cached oci image download tarball %s", export.URL,
			)
		}
	}
	defer func() {
		if err := archiveFile.Close(); err != nil {
			// TODO: handle this error more rigorously
			fmt.Fprintf(os.Stderr, "Error: couldn't close cached download %s\n", export.URL)
		}
	}()
	return filetype.MatchReader(archiveFile)
}

func extractFromArchive(
	tarReader *tar.Reader, sourcePath, exportPath string, destPerms fs.FileMode,
) error {
	if sourcePath == "/" || sourcePath == "." {
		sourcePath = ""
	}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if sourcePath != "" && sourcePath != header.Name &&
			!strings.HasPrefix(header.Name, sourcePath+"/") {
			continue
		}

		if err = extractFile(header, tarReader, sourcePath, exportPath, destPerms); err != nil {
			return err
		}
	}
	return nil
}

func extractFile(
	// FIXME: also handle destPerms for directories and symlinks!
	header *tar.Header, tarReader *tar.Reader, sourcePath, exportPath string, destPerms fs.FileMode,
) error {
	targetPath := path.Join(exportPath, strings.TrimPrefix(header.Name, sourcePath))
	switch header.Typeflag {
	default:
		return errors.Errorf("unknown type of file %s in archive: %b", header.Name, header.Typeflag)
	case tar.TypeDir:
		if err := ffs.EnsureExists(filepath.FromSlash(targetPath)); err != nil {
			return errors.Wrapf(
				err, "couldn't export directory %s from archive to %s", header.Name, targetPath,
			)
		}
	case tar.TypeReg:
		if err := extractRegularFile(header, tarReader, sourcePath, targetPath, destPerms); err != nil {
			return errors.Wrapf(
				err, "couldn't export regular file %s from archive to %s", header.Name, targetPath,
			)
		}
	case tar.TypeSymlink:
		if err := os.Symlink(
			filepath.FromSlash(header.Linkname), filepath.FromSlash(targetPath),
		); err != nil {
			return errors.Wrapf(
				err, "couldn't export symlink %s from archive to %s", header.Name, targetPath,
			)
		}
	case tar.TypeLink:
		if err := os.Link(
			filepath.FromSlash(path.Join(exportPath, strings.TrimPrefix(header.Linkname, sourcePath))),
			filepath.FromSlash(targetPath),
		); err != nil {
			return errors.Wrapf(
				err, "couldn't export hardlink %s from archive to %s", header.Name, targetPath,
			)
		}
	}
	return nil
}

func extractRegularFile(
	header *tar.Header, tarReader *tar.Reader, sourcePath, targetPath string, destPerms fs.FileMode,
) error {
	if destPerms == 0 {
		destPerms = fs.FileMode( //nolint:gosec // (G115) tar's Mode won't(?) overflow fs.FileMode
			header.Mode,
		) & fs.ModePerm
	}
	// FIXME: we suppress gosec G304 below, but for security we should check targetPath to ensure it's
	// a valid path (i.e. within the Forklift workspace)!
	targetFile, err := os.OpenFile(
		filepath.Clean(filepath.FromSlash(targetPath)), os.O_RDWR|os.O_CREATE|os.O_TRUNC, destPerms,
	)
	if err != nil {
		return errors.Wrapf(err, "couldn't create export file at %s", targetPath)
	}
	defer func(file fs.File, filePath string) {
		if err := file.Close(); err != nil {
			// FIXME: handle this error more rigorously
			fmt.Fprintf(os.Stderr, "Error: couldn't close export file %s\n", filePath)
		}
	}(targetFile, targetPath)

	if _, err = io.Copy(targetFile, tarReader); err != nil {
		return errors.Wrapf(
			err, "couldn't copy file %s in tar archive to %s", sourcePath, targetPath,
		)
	}
	return nil
}
