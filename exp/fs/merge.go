package fs

import (
	"cmp"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/exp/structures"
)

// A FileRef is a reference to a file in a [PathedFS] by its file path.
type FileRef struct {
	// Ordered identifiers of the sources of the file reference, e.g. the pallet at the root of the FS
	Sources []string
	// The FS which the file can be loaded from
	FS PathedFS
	// The path where the file exists relative to the root of the FS
	Path string
}

// MergeFS

// A MergeFS is an FS constructed by combining a [PathedFS] as an overlay over an underlay
// constructed as a collection of references to files in other [PathedFS] instances.
// The path of the FS is the path of the overlay.
type MergeFS struct {
	// Ordered identifiers of the source(s) of the MergeFS, e.g. the pallet at the root of the FS
	Overlay      PathedFS
	underlayRefs map[string]FileRef     // target -> source
	impliedDirs  structures.Set[string] // target
}

func NewMergeFS(overlay PathedFS, underlayRefs map[string]FileRef) *MergeFS {
	impliedDirs := make(structures.Set[string])
	for target := range underlayRefs {
		for {
			target = path.Dir(target)
			if target == "/" || target == "." {
				break
			}
			if _, err := fs.Stat(overlay, target); err == nil {
				break
			}
			impliedDirs.Add(target)
		}
	}
	// fmt.Printf("newMergeFS(%s, %d, %d)\n", overlay.Path(), len(underlayRefs), len(impliedDirs))
	return &MergeFS{
		Overlay:      overlay,
		underlayRefs: underlayRefs,
		impliedDirs:  impliedDirs,
	}
}

// Resolve returns the path of the named file from the overlay (if it exists in the overlay), or
// else from an underlay filesystem depending on which one is recorded to have that file.
func (f *MergeFS) Resolve(name string) (string, error) {
	ref, err := f.GetFileRef(name)
	if err != nil {
		return "", err
	}
	return path.Join(ref.FS.Path(), ref.Path), nil
}

func (f *MergeFS) ListImports() (map[string]FileRef, error) {
	imports := make(map[string]FileRef)
	for target, sourceRef := range f.underlayRefs {
		target = path.Clean(target)
		_, err := fs.Stat(f.Overlay, target)
		switch {
		default:
			return nil, errors.Errorf("couldn't check whether file %s exists in overlay", target)
		case err == nil:
			// file is in overlay, so it's not an import
			continue
		case errors.Is(err, fs.ErrNotExist):
			if target == "." {
				// file is a directory, which we don't want to list as an import
				continue
			}
			imports[target] = sourceRef
		}
	}
	return imports, nil
}

func (f *MergeFS) GetFileRef(name string) (FileRef, error) {
	name = path.Clean(name)
	// fmt.Printf("Resolve(%s|%s)\n", f.Path(), name)
	_, err := fs.Stat(f.Overlay, name)
	switch {
	default:
		return FileRef{}, &fs.PathError{
			Op:   "resolve",
			Path: name,
			Err:  errors.Wrapf(err, "couldn't stat file %s in overlay", name),
		}
	case errors.Is(err, fs.ErrNotExist):
		if name == "." {
			return FileRef{
				FS:   f,
				Path: ".",
			}, nil
		}
		ref, ok := f.underlayRefs[name]
		if !ok {
			if !f.impliedDirs.Has(name) {
				return FileRef{}, &fs.PathError{
					Op:   "resolve",
					Path: name,
					Err:  errors.Errorf("file %s not found in either overlay or underlay", name),
				}
			}
			// fmt.Printf("  %s is an implied dir!\n", name)
			return FileRef{}, &fs.PathError{
				Op:   "resolve",
				Path: name,
				Err:  errors.Errorf("file %s is a directory implied by the underlay", name),
			}
		}
		if _, err := fs.Stat(ref.FS, ref.Path); err != nil {
			return FileRef{}, &fs.PathError{
				Op:   "resolve",
				Path: name,
				Err: errors.Wrapf(
					err, "couldn't stat file %s in underlay as %s", name, path.Join(ref.FS.Path(), ref.Path),
				),
			}
		}
		return FileRef{
			Sources: slices.Clone(ref.Sources),
			FS:      ref.FS,
			Path:    ref.Path,
		}, nil
	case err == nil:
		return FileRef{
			FS:   f.Overlay,
			Path: name,
		}, nil
	}
}

// MergeFS: PathedFS

// Path returns the path of the overlay.
func (f *MergeFS) Path() string {
	return f.Overlay.Path()
}

// Open opens the named file from the overlay (if it exists in the overlay), or else from an
// underlay filesystem depending on which one is recorded to have that file.
func (f *MergeFS) Open(name string) (fs.File, error) {
	name = path.Clean(name)
	// fmt.Printf("Open(%s|%s)\n", f.Path(), name)
	file, err := f.Overlay.Open(name)
	switch {
	default:
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  errors.Wrapf(err, "couldn't open file %s in overlay", name),
		}
	case errors.Is(err, fs.ErrNotExist):
		ref, ok := f.underlayRefs[name]
		if !ok {
			if !f.impliedDirs.Has(name) {
				return nil, &fs.PathError{
					Op:   "open",
					Path: name,
					Err:  errors.Errorf("file %s not found in either overlay or underlay", name),
				}
			}
			// TODO: implement this
			return nil, errors.New("unimplemented: opening file for implied dir")
		}
		file, err := ref.FS.Open(ref.Path)
		if err != nil {
			return nil, &fs.PathError{
				Op:   "open",
				Path: name,
				Err: errors.Wrapf(
					err, "couldn't open file %s in underlay as %s", name, path.Join(ref.FS.Path(), ref.Path),
				),
			}
		}
		return file, nil
	case err == nil:
		return file, nil
	}
}

// Sub returns a MergeFS corresponding to the subtree rooted at dir.
func (f *MergeFS) Sub(dir string) (PathedFS, error) {
	dir = path.Clean(dir)
	// fmt.Printf("Sub(%s|%s)\n", f.Path(), dir)
	if dir == "." {
		return f, nil
	}
	overlaySub, err := f.Overlay.Sub(dir)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make subtree for overlay")
	}

	prefix := dir + "/"
	underlayRefsSub := make(map[string]FileRef)
	for target, ref := range f.underlayRefs {
		if !strings.HasPrefix(target, prefix) {
			continue
		}
		underlayRefsSub[strings.TrimPrefix(target, prefix)] = ref
		// fmt.Printf("  - %s\n", strings.TrimPrefix(target, prefix))
	}
	return NewMergeFS(overlaySub, underlayRefsSub), nil
}

// MergeFS: fs.ReadDirFS

// ReadDir reads the named directory and returns a list of directory entries sorted by filename.
func (f *MergeFS) ReadDir(name string) (entries []fs.DirEntry, err error) {
	name = path.Clean(name)
	// fmt.Printf("ReadDir(%s|%s)\n", f.Path(), name)
	entryNames := make(structures.Set[string])

	info, err := fs.Stat(f.Overlay, name)
	if err == nil {
		if !info.IsDir() {
			return nil, &fs.PathError{
				Op:   "read",
				Path: name,
				Err:  errors.Wrapf(err, "%s is a non-directory file in overlay", name),
			}
		}
		if entries, err = fs.ReadDir(f.Overlay, name); err != nil {
			return nil, &fs.PathError{
				Op:   "read",
				Path: name,
				Err:  errors.Wrapf(err, "couldn't read directory %s in overlay", name),
			}
		}
		for _, entry := range entries {
			entryNames.Add(entry.Name())
		}
	}

	for target, ref := range f.underlayRefs {
		entryName := path.Base(target)
		if entryNames.Has(entryName) { // e.g. entry was already added by the overlay
			continue
		}
		entry, err := matchUnderlayRef(name, target, ref)
		if err != nil {
			return nil, err
		}
		if entry == nil {
			continue
		}

		entries = append(entries, entry)
		entryNames.Add(entryName)
	}

	for dir := range f.impliedDirs {
		if path.Dir(dir) != name {
			continue
		}
		entryName := path.Base(dir)
		if entryNames.Has(entryName) { // e.g. entry was already added by the overlay
			continue
		}
		entries = append(entries, &impliedDirEntry{name: entryName})
		entryNames.Add(entryName)
	}

	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	return entries, nil
}

func matchUnderlayRef(
	fileName, underlayTarget string, underlayRef FileRef,
) (entry *importedDirEntry, err error) {
	prefixPattern := path.Join(fileName, "*")
	if fileName == "." {
		prefixPattern = "*"
	}

	match, err := doublestar.Match(prefixPattern, underlayTarget)
	if err != nil {
		return nil, &fs.PathError{
			Op:   "read",
			Path: fileName,
			Err:  errors.Wrap(err, "couldn't enumerate files in underlays"),
		}
	}
	if !match {
		return nil, nil
	}

	entry = &importedDirEntry{
		name: path.Base(underlayTarget),
		ref:  underlayRef,
	}
	if fsys, ok := underlayRef.FS.(ReadLinkFS); ok {
		if entry.fileInfo, err = fsys.StatLink(underlayRef.Path); err != nil {
			return nil, &fs.PathError{
				Op:   "read",
				Path: fileName,
				Err: errors.Wrapf(
					err, "couldn't stat (without following symlinks) file %s in %s",
					entry.ref.Path, entry.ref.FS.Path(),
				),
			}
		}
	} /* else if entry.fileInfo, err = fs.Stat(underlayRef.FS, underlayRef.Path); err != nil {
		return nil, &fs.PathError{
			Op:   "read",
			Path: fileName,
			Err:  errors.Wrapf(err, "couldn't stat file %s in %s", entry.ref.Path, entry.ref.FS.Path()),
		}
	}*/
	return entry, nil
}

// MergeFS: fs.ReadFileFS

// ReadFile returns the contents from reading the named file from the overlay (if it exists in the
// overlay), or else from an underlay filesystem depending on which one is recorded to have that
// file.
func (f *MergeFS) ReadFile(name string) ([]byte, error) {
	name = path.Clean(name)
	// fmt.Printf("ReadFile(%s|%s)\n", f.Path(), name)
	contents, err := fs.ReadFile(f.Overlay, name)
	switch {
	default:
		return nil, errors.Wrapf(err, "couldn't read file %s in overlay", name)
	case errors.Is(err, fs.ErrNotExist):
		ref, ok := f.underlayRefs[name]
		if !ok {
			if f.impliedDirs.Has(name) {
				return nil, &fs.PathError{
					Op:   "read",
					Path: name,
					Err:  errors.Errorf("file %s is a directory implied by the underlay", name),
				}
			}
			return nil, errors.Errorf(
				"file %s not found in either overlay or underlay of %s", name, f.Path(),
			)
		}
		contents, err := fs.ReadFile(ref.FS, ref.Path)
		return contents, errors.Wrapf(
			err, "couldn't read file %s in underlay as %s", name, path.Join(ref.FS.Path(), ref.Path),
		)
	case err == nil:
		return contents, nil
	}
}

// MergeFS: fs.StatFS

// Stat returns a [fs.FileInfo] describing the file from the overlay (if it exists in the overlay),
// or else from an underlay filesystem depending on which one is recorded to have that file.
func (f *MergeFS) Stat(name string) (fs.FileInfo, error) {
	name = path.Clean(name)
	// fmt.Printf("Stat(%s|%s)\n", f.Path(), name)
	info, err := fs.Stat(f.Overlay, name)
	switch {
	default:
		return nil, &fs.PathError{
			Op:   "stat",
			Path: name,
			Err:  errors.Wrapf(err, "couldn't stat file %s in overlay", name),
		}
	case errors.Is(err, fs.ErrNotExist):
		if name == "." {
			return &impliedDirFileInfo{name: path.Base(f.Path())}, nil
		}
		ref, ok := f.underlayRefs[name]
		if !ok {
			if !f.impliedDirs.Has(name) {
				return nil, &fs.PathError{
					Op:   "stat",
					Path: name,
					Err:  errors.Errorf("file %s not found in either overlay or underlay", name),
				}
			}
			// fmt.Printf("  %s is an implied dir!\n", name)
			return &impliedDirFileInfo{name: path.Base(name)}, nil
		}
		info, err := fs.Stat(ref.FS, ref.Path)
		if err != nil {
			return nil, &fs.PathError{
				Op:   "stat",
				Path: name,
				Err: errors.Wrapf(
					err, "couldn't stat file %s in underlay as %s", name, path.Join(ref.FS.Path(), ref.Path),
				),
			}
		}
		return info, nil
	case err == nil:
		return info, nil
	}
}

// MergeFS: fs.ReadLinkFS

func (f *MergeFS) ReadLink(name string) (string, error) {
	name = path.Clean(name)
	// fmt.Printf("ReadLink(%s|%s)\n", f.Path(), name)
	target, err := ReadLink(f.Overlay, name)
	switch {
	default:
		return "", &fs.PathError{
			Op:   "lstat",
			Path: name,
			Err: errors.Wrapf(
				err, "couldn't stat (without following symlinks) file %s in overlay", name,
			),
		}
	case errors.Is(err, fs.ErrNotExist):
		ref, ok := f.underlayRefs[name]
		if !ok {
			return "", &fs.PathError{
				Op:   "lstat",
				Path: name,
				Err:  errors.Errorf("file %s not a symlink in overlay or underlay", name),
			}
		}
		target, err := ReadLink(ref.FS, ref.Path)
		if err != nil {
			return "", &fs.PathError{
				Op:   "lstat",
				Path: name,
				Err: errors.Wrapf(
					err, "couldn't stat file (without following symlinks) %s in underlay as %s",
					name, path.Join(ref.FS.Path(), ref.Path),
				),
			}
		}
		return target, nil
	case err == nil:
		return target, nil
	}
}

func (f *MergeFS) StatLink(name string) (fs.FileInfo, error) {
	name = path.Clean(name)
	// fmt.Printf("StatLink(%s|%s)\n", f.Path(), name)
	info, err := StatLink(f.Overlay, name)
	switch {
	default:
		return nil, &fs.PathError{
			Op:   "lstat",
			Path: name,
			Err: errors.Wrapf(
				err, "couldn't stat (without following symlinks) file %s in overlay", name,
			),
		}
	case errors.Is(err, fs.ErrNotExist):
		if name == "." {
			return &impliedDirFileInfo{name: path.Base(f.Path())}, nil
		}
		ref, ok := f.underlayRefs[name]
		if !ok {
			if !f.impliedDirs.Has(name) {
				return nil, &fs.PathError{
					Op:   "lstat",
					Path: name,
					Err:  errors.Errorf("file %s not found in either overlay or underlay", name),
				}
			}
			// fmt.Printf("  %s is an implied dir!\n", name)
			return &impliedDirFileInfo{name: path.Base(name)}, nil
		}
		info, err := StatLink(ref.FS, ref.Path)
		if err != nil {
			return nil, &fs.PathError{
				Op:   "lstat",
				Path: name,
				Err: errors.Wrapf(
					err, "couldn't stat file (without following symlinks) %s in underlay as %s",
					name, path.Join(ref.FS.Path(), ref.Path),
				),
			}
		}
		return info, nil
	case err == nil:
		return info, nil
	}
}

// importedDirEntry

// An importedDirEntry is a [fs.DirEntry] for a file which is imported from an underlay of a
// [MergeFS].
type importedDirEntry struct {
	// name is the name of the imported file described by the entry; it's only the final element
	// of the target path (the base name) in the [MergeFS], not the entire target path.
	name string
	// ref holds information for looking up the source file to be imported.
	ref FileRef
	// fileInfo holds information about the source file to be imported.
	fileInfo fs.FileInfo
}

// importedDirEntry: fs.DirEntry

func (de *importedDirEntry) Name() string {
	return de.name
}

func (de *importedDirEntry) IsDir() bool {
	return de.fileInfo.IsDir()
}

func (de *importedDirEntry) Type() fs.FileMode {
	return de.fileInfo.Mode()
}

func (de *importedDirEntry) Info() (fs.FileInfo, error) {
	return de.fileInfo, nil
}

// impliedDirEntry

// An impliedDirEntry is a [fs.DirEntry] for a directory whose existence is implied by one or
// more underlays of a [MergeFS], but which does not necessarily exist in an underlay and does not
// not necessarily have a unique source among the underlays.
type impliedDirEntry struct {
	// Name is the name of the implied directory described by the entry; it's only the final element
	// of the path (the base name), not the entire path.
	name string
}

// impliedDirEntry: fs.DirEntry

func (de *impliedDirEntry) Name() string {
	return de.name
}

func (de *impliedDirEntry) IsDir() bool {
	return true
}

func (de *impliedDirEntry) Type() fs.FileMode {
	return fs.ModeDir
}

func (de *impliedDirEntry) Info() (fs.FileInfo, error) {
	return &impliedDirFileInfo{name: de.name}, nil
}

// impliedDirFileInfo

// An impliedDirFileInfo is a [fs.FileInfo] for a directory whose existence is implied by one or
// more underlays of a [MergeFS], but which does not necessarily exist in an underlay and does not
// not necessarily have a unique source among the underlays.
type impliedDirFileInfo struct {
	// Name is the name of the implied directory described by the entry; it's only the final element
	// of the path (the base name), not the entire path.
	name string
}

// impliedDirFileInfo: fs.FileInfo

func (fi *impliedDirFileInfo) Name() string {
	return fi.name
}

func (fi *impliedDirFileInfo) Size() int64 {
	return 0
}

func (fi *impliedDirFileInfo) Mode() fs.FileMode {
	const perm = 0o755 // owner rwx, group rx, public rx
	return fs.ModeDir | perm
}

func (fi *impliedDirFileInfo) ModTime() time.Time {
	return time.Time{}
}

func (fi *impliedDirFileInfo) IsDir() bool {
	return true
}

func (fi *impliedDirFileInfo) Sys() any {
	return nil
}
