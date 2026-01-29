package cli

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/internal/app/forklift"
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

func FprintFile(out io.Writer, pallet *fplt.FSPallet, filePath string) error {
	data, err := fs.ReadFile(pallet.FS, filePath)
	if err != nil {
		return errors.Wrapf(err, "couldn't read file %s in %s", filePath, pallet.FS.Path())
	}

	_, _ = fmt.Fprint(out, string(data))
	return nil
}

func EditFileWithCOW(pallet *fplt.FSPallet, filePath, editor string) error {
	fsys, ok := pallet.FS.(*ffs.MergeFS)
	if !ok {
		fullPath := path.Join(pallet.FS.Path(), filePath)
		return editFile(editor, fullPath, path.Dir(fullPath))
	}
	overlayPath := path.Join(fsys.Overlay.Path(), filePath)
	resolved, err := fsys.Resolve(filePath)
	if err != nil || strings.HasPrefix(resolved, overlayPath) {
		return editFile(editor, overlayPath, path.Dir(overlayPath))
	}

	// Copy file from underlay into an editable temporary file
	sourceInfo, err := fs.Stat(fsys, filePath)
	if err != nil {
		return errors.Wrapf(err, "couldn't stat source file %s in underlay for copying", filePath)
	}
	original, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return errors.Wrapf(err, "couldn't read source file %s in underlay", filePath)
	}
	edited, err := editWithTempFile(editor, filePath, original)
	if err != nil {
		return errors.Wrapf(err, "couldn't edit %s in a temporary file", filePath)
	}
	if bytes.Equal(edited, original) {
		fmt.Fprintf(
			os.Stderr, "Warning: the file wasn't changed from %s, so it won't be saved to %s!\n",
			resolved, overlayPath,
		)
		return nil
	}

	fmt.Fprintf(os.Stderr, "Saving edits on %s to %s...\n", resolved, overlayPath)
	if err = forklift.EnsureExists(filepath.FromSlash(path.Dir(overlayPath))); err != nil {
		return err
	}
	overlayFile, err := os.OpenFile(
		filepath.FromSlash(overlayPath), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, sourceInfo.Mode().Perm(),
	)
	if err != nil {
		return errors.Wrapf(err, "couldn't open dest file %s for copying", overlayPath)
	}
	defer func() {
		// FIXME: handle this error more rigorously
		if err := overlayFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: couldn't close dest file %s\n", overlayPath)
		}
	}()
	if _, err := overlayFile.Write(edited); err != nil {
		return errors.Wrapf(err, "couldn't write edits to %s", overlayPath)
	}
	fmt.Fprintln(os.Stderr, "Done!")
	return nil
}

func editFile(editor, filePath, cwd string) error {
	cmd := exec.Command( //nolint:gosec // we trust the user to provide reasonable args
		editor, filepath.FromSlash(filePath),
	)
	cmd.Dir = filepath.FromSlash(cwd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func editWithTempFile(editor, filePath string, original []byte) (edited []byte, err error) {
	tempFile, err := os.CreateTemp("", path.Base(filePath)+".*")
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't make temporary editable copy of %s", filePath)
	}
	defer func() {
		if err := tempFile.Close(); err != nil {
			// FIXME: handle this error more rigorously
			fmt.Fprintf(os.Stderr, "Error: couldn't close temporary file %s\n", filePath)
		}
		if err := os.Remove(tempFile.Name()); err != nil {
			fmt.Fprintf(
				os.Stderr, "Error: couldn't delete temporary file %s; you may need to delete it yourself\n",
				filePath,
			)
		}
	}()
	if _, err = tempFile.Write(original); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't copy contents of source file into temporary file %s", tempFile.Name(),
		)
	}
	if err = editFile(editor, tempFile.Name(), ""); err != nil {
		return nil, err
	}
	if edited, err = os.ReadFile(tempFile.Name()); err != nil {
		return nil, errors.Wrapf(err, "couldn't read edited temporary file %s", tempFile.Name())
	}
	return edited, nil
}

func RemoveFile(indent int, pallet *fplt.FSPallet, filePath string) error {
	if err := os.RemoveAll(filepath.FromSlash(path.Join(pallet.FS.Path(), filePath))); err != nil {
		return err
	}
	paths, err := ListPalletFiles(pallet, filePath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't check whether any files at/in %s are imported from other pallets", filePath,
		)
	}
	if len(paths) == 0 {
		return nil
	}

	IndentedFprintln(
		indent, os.Stderr, "Warning: the following files are currently imported from other pallets:",
	)
	indent++
	for _, p := range paths {
		IndentedFprintln(indent, os.Stderr, p)
	}
	return nil
}
