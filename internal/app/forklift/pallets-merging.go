package forklift

import (
	"cmp"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/core"
	"github.com/PlanktoScope/forklift/pkg/structures"
)

// MergeFSPallet creates a new FSPallet with a virtual (read-only) filesystem created by evaluating
// the pallet's file imports with its required pallets (which should be loadable using the provided
// loader).
func MergeFSPallet(
	shallow *FSPallet, palletLoader FSPalletLoader, prohibitedPallets structures.Set[string],
) (merged *FSPallet, err error) {
	merged = &FSPallet{
		Pallet: shallow.Pallet,
		Repo:   &core.FSRepo{Repo: shallow.Repo.Repo},
	}
	imports, err := shallow.LoadImports("**/*")
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't check for import groups")
	}
	hasEnabledImports := false
	for _, imp := range imports {
		if !imp.Def.Disabled {
			hasEnabledImports = true
			break
		}
	}
	if !hasEnabledImports { // base case for recursive merging
		return shallow, nil
	}

	allResolved, err := ResolveImports(shallow, palletLoader, imports)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't resolve import groups")
	}
	allProhibitedPallets := make(structures.Set[string])
	maps.Copy(prohibitedPallets, allProhibitedPallets)
	allProhibitedPallets.Add(shallow.Path())
	palletFileMappings, pallets, err := evaluatePalletImports( // recursive step for merging
		allResolved, palletLoader, prohibitedPallets,
	)
	if err != nil {
		return nil, errors.Wrap(
			err, "couldn't evaluate import groups for imports from required pallets",
		)
	}

	underlayRefs, err := mergePalletImports(palletFileMappings, pallets)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't merge file imports across all required pallets")
	}
	// fmt.Printf("Merging file imports into %s:\n", shallow.Path())
	// for _, target := range sortKeys(underlayRefs) {
	// 	fmt.Printf("  - %s\n", target)
	// }
	// fmt.Println()
	merged.FS = newMergeFS(shallow.FS, underlayRefs)
	merged.Repo.FS = merged.FS
	return merged, nil
}

// evaluatePalletImports splits up a flat list of ResolvedImports into a map from pallet paths to
// maps from target paths to source paths for all file imports from the respective pallets; it also
// builds a map from pallet paths to the results of merging the respective pallets.
func evaluatePalletImports(
	allResolved []*ResolvedImport, palletLoader FSPalletLoader,
	prohibitedPallets structures.Set[string],
) (palletFileMappings map[string]map[string]string, pallets map[string]*FSPallet, err error) {
	resolvedByPallet := make(map[string][]*ResolvedImport) // pallet path -> imports from that pallet
	pallets = make(map[string]*FSPallet)                   // pallet path -> pallet
	for _, resolved := range allResolved {
		palletPath := resolved.Pallet.Path()
		if prohibitedPallets.Has(palletPath) {
			return nil, nil, errors.Errorf(
				"import group %s is for pallet %s, which is not allowed as an import (maybe it's part of "+
					"a circular dependency of pallet requirements?)",
				resolved.Name, palletPath,
			)
		}
		resolvedByPallet[palletPath] = append(resolvedByPallet[palletPath], resolved)
		pallets[palletPath] = resolved.Pallet
	}

	palletFileMappings = make(map[string]map[string]string) // pallet path -> target -> source
	for palletPath, palletResolved := range resolvedByPallet {
		if palletFileMappings[palletPath], err = consolidatePalletImports(palletResolved); err != nil {
			return nil, nil, errors.Wrapf(
				err, "couldn't evaluate import groups for pallet %s", palletPath,
			)
		}
	}

	for palletPath, pallet := range pallets {
		// Note: if we find that recursively merging pallets is computationally expensive, we can cache
		// the results of merging pallets. However, correctly caching merged pallets to/from disk adds
		// nontrivial complexity due to the need to decide when to invalidate cache entries, so for now
		// we don't implement any caching.
		if pallets[palletPath], err = MergeFSPallet(
			pallet, palletLoader, prohibitedPallets,
		); err != nil {
			return nil, nil, errors.Wrapf(
				err, "couldn't compute merged pallet for required pallet %s", palletPath,
			)
		}
	}
	return palletFileMappings, pallets, nil
}

// consolidatePalletImports checks the import groups loaded for a single required pallet and
// consolidates into a single mapping between target paths and source paths relative to the
// required pallet.
func consolidatePalletImports(imports []*ResolvedImport) (map[string]string, error) {
	union := make(map[string]string)           // target -> source
	mappingOrigin := make(map[string][]string) // target -> import group names
	for _, imp := range imports {
		importMappings, err := imp.Evaluate()
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't evaluate import group %s", imp.Import.Name)
		}
		for target, source := range importMappings {
			if unionSource, ok := union[target]; ok && unionSource != source {
				return nil, errors.Errorf(
					"import group %s adds a mapping from %s to target %s, but other import groups %s add "+
						"a mapping from %s to target %s",
					imp.Name, source, target, mappingOrigin[target], unionSource, target,
				)
			}
			union[target] = source
			mappingOrigin[target] = append(mappingOrigin[target], imp.Name)
		}
	}
	for target := range union {
		if strings.HasPrefix(target, "/requirements/pallets") {
			return union, errors.Errorf(
				"target %s is in /requirements/pallets, which is not allowed", target,
			)
		}
		if path.Dir(target) == "/" || path.Dir(target) == "." {
			return union, errors.Errorf(
				"target %s is a root-level file, which is not allowed", target,
			)
		}
	}
	return union, nil
}

// A fileRef is a reference to a file in a [core.PathedFS] by its file path.
type fileRef struct {
	fs   core.PathedFS
	path string
}

// mergePalletImports builds a mapping from all target file paths to their respective source files
// suitable for instantiating a MergeFS.
func mergePalletImports(
	palletFileMappings map[string]map[string]string, pallets map[string]*FSPallet,
) (map[string]fileRef, error) {
	merged := make(map[string]fileRef) // target -> source
	for palletPath, fileMappings := range palletFileMappings {
		for target, source := range fileMappings {
			ref := fileRef{
				fs:   pallets[palletPath].FS,
				path: strings.TrimPrefix(source, "/"),
			}
			prevRef, ok := merged[strings.TrimPrefix(target, "/")]
			if !ok {
				merged[strings.TrimPrefix(target, "/")] = ref
				continue
			}

			// TODO: check for file contents+metadata conflicts among file mappings of different
			// required pallets; when multiple pallets map the same source file to the same target file,
			// choose the alphabetically first pallet for resolving that target file. For now we just
			// reject all such situations as invalid, because that's simpler.
			if prevRef.fs.Path() != ref.fs.Path() || prevRef.path != ref.path {
				return nil, errors.Errorf(
					"couldn't add a mapping from %s to target %s, when a mapping was previously added "+
						"from %s to target %s (we have not yet implemented a relaxation of this check to "+
						"allow cases where duplicate mappings refer to the exact same file)",
					path.Join(ref.fs.Path(), ref.path), target,
					path.Join(prevRef.fs.Path(), prevRef.path), target,
				)
			}
		}
	}
	return merged, nil
}

// MergeFS

// A MergeFS is an FS constructed by combining a [core.PathedFS] as an overlay over an underlay
// constructed as a collection of references to files in other [core.PathedFS] instances.
// The path of the FS is the path of the overlay.
type MergeFS struct {
	Overlay      core.PathedFS
	underlayRefs map[string]fileRef     // target -> source
	impliedDirs  structures.Set[string] // target
}

func newMergeFS(overlay core.PathedFS, underlayRefs map[string]fileRef) *MergeFS {
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
	name = path.Clean(name)
	// fmt.Printf("Resolve(%s|%s)\n", f.Path(), name)
	_, err := fs.Stat(f.Overlay, name)
	switch {
	default:
		return "", &fs.PathError{
			Op:   "resolve",
			Path: name,
			Err:  errors.Wrapf(err, "couldn't stat file %s in overlay", name),
		}
	case errors.Is(err, fs.ErrNotExist):
		if name == "." {
			return f.Path(), nil
		}
		ref, ok := f.underlayRefs[name]
		if !ok {
			if !f.impliedDirs.Has(name) {
				return "", &fs.PathError{
					Op:   "resolve",
					Path: name,
					Err:  errors.Errorf("file %s not found in either overlay or underlay", name),
				}
			}
			// fmt.Printf("  %s is an implied dir!\n", name)
			return "", &fs.PathError{
				Op:   "resolve",
				Path: name,
				Err:  errors.Errorf("file %s is a directory implied by the underlay", name),
			}
		}
		if _, err := fs.Stat(ref.fs, ref.path); err != nil {
			return "", &fs.PathError{
				Op:   "resolve",
				Path: name,
				Err: errors.Wrapf(
					err, "couldn't stat file %s in underlay as %s", name, path.Join(ref.fs.Path(), ref.path),
				),
			}
		}
		return path.Join(ref.fs.Path(), ref.path), nil
	case err == nil:
		return path.Join(f.Overlay.Path(), name), nil
	}
}

// MergeFS: core.PathedFS

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
		file, err := ref.fs.Open(ref.path)
		if err != nil {
			return nil, &fs.PathError{
				Op:   "open",
				Path: name,
				Err: errors.Wrapf(
					err, "couldn't open file %s in underlay as %s", name, path.Join(ref.fs.Path(), ref.path),
				),
			}
		}
		return file, nil
	case err == nil:
		return file, nil
	}
}

// Sub returns a MergeFS corresponding to the subtree rooted at dir.
func (f *MergeFS) Sub(dir string) (core.PathedFS, error) {
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
	underlayRefsSub := make(map[string]fileRef)
	for target, ref := range f.underlayRefs {
		if !strings.HasPrefix(target, prefix) {
			continue
		}
		underlayRefsSub[strings.TrimPrefix(target, prefix)] = ref
		// fmt.Printf("  - %s\n", strings.TrimPrefix(target, prefix))
	}
	return newMergeFS(overlaySub, underlayRefsSub), nil
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
	fileName, underlayTarget string, underlayRef fileRef,
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
	if entry.fileInfo, err = fs.Stat(underlayRef.fs, underlayRef.path); err != nil {
		return nil, &fs.PathError{
			Op:   "read",
			Path: fileName,
			Err:  errors.Wrapf(err, "couldn't stat file %s in %s", entry.ref.path, entry.ref.fs.Path()),
		}
	}
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
		contents, err := fs.ReadFile(ref.fs, ref.path)
		return contents, errors.Wrapf(
			err, "couldn't read file %s in underlay as %s", name, path.Join(ref.fs.Path(), ref.path),
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
		info, err := fs.Stat(ref.fs, ref.path)
		if err != nil {
			return nil, &fs.PathError{
				Op:   "stat",
				Path: name,
				Err: errors.Wrapf(
					err, "couldn't stat file %s in underlay as %s", name, path.Join(ref.fs.Path(), ref.path),
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
		target, err := ReadLink(ref.fs, ref.path)
		if err != nil {
			return "", &fs.PathError{
				Op:   "lstat",
				Path: name,
				Err: errors.Wrapf(
					err, "couldn't stat file (without following symlinks) %s in underlay as %s",
					name, path.Join(ref.fs.Path(), ref.path),
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
		info, err := StatLink(ref.fs, ref.path)
		if err != nil {
			return nil, &fs.PathError{
				Op:   "lstat",
				Path: name,
				Err: errors.Wrapf(
					err, "couldn't stat file (without following symlinks) %s in underlay as %s",
					name, path.Join(ref.fs.Path(), ref.path),
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
	ref fileRef
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
