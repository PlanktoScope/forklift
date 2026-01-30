package fs

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
)

func CopyFS(fsys PathedFS, dest string) error {
	return fs.WalkDir(fsys, ".", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			fileInfo, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(filepath.FromSlash(path.Join(dest, filePath)), fileInfo.Mode())
		}
		return CopyFSFile(fsys, filePath, path.Join(dest, filePath), 0)
	})
}

func CopyFSFile(fsys PathedFS, sourcePath, destPath string, destPerms fs.FileMode) error {
	if readLinkFS, ok := fsys.(ReadLinkFS); ok {
		sourceInfo, err := readLinkFS.StatLink(sourcePath)
		if err != nil {
			return errors.Wrapf(
				err, "couldn't stat source file %s for copying", path.Join(readLinkFS.Path(), sourcePath),
			)
		}
		if (sourceInfo.Mode() & fs.ModeSymlink) != 0 {
			return copyFSSymlink(readLinkFS, sourcePath, destPath)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Warning: %s was not loaded as a ReadLinkFS!\n", fsys.Path())
	}

	sourceFile, err := fsys.Open(sourcePath)
	fullSourcePath := path.Join(fsys.Path(), sourcePath)
	if err != nil {
		return errors.Wrapf(err, "couldn't open source file %s for copying", fullSourcePath)
	}
	defer func() {
		// FIXME: handle this error more rigorously
		if err := sourceFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: couldn't close source file %s\n", fullSourcePath)
		}
	}()
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return errors.Wrapf(err, "couldn't stat source file %s for copying", fullSourcePath)
	}
	if sourceInfo.IsDir() {
		fsys, err := fsys.Sub(sourcePath)
		if err != nil {
			return err
		}
		return CopyFS(fsys, destPath)
	}

	if destPerms == 0 {
		destPerms = sourceInfo.Mode().Perm()
	}
	destFile, err := os.OpenFile(
		destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, destPerms,
	)
	if err != nil {
		return errors.Wrapf(err, "couldn't open dest file %s for copying", destPath)
	}
	defer func() {
		// FIXME: handle this error more rigorously
		if err := destFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: couldn't close dest file %s\n", destPath)
		}
	}()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return errors.Wrapf(err, "couldn't copy %s to %s", fullSourcePath, destPath)
	}

	return nil
}

func copyFSSymlink(fsys PathedFS, sourcePath, destPath string) error {
	readLinkFS, ok := fsys.(ReadLinkFS)
	if !ok {
		return errors.Errorf("%s is not a ReadLinkFS!", fsys.Path())
	}

	linkTarget, err := readLinkFS.ReadLink(sourcePath)
	if err != nil {
		return errors.Wrapf(err, "couldn't determine symlink target of %s", sourcePath)
	}
	return os.Symlink(linkTarget, destPath)
}
