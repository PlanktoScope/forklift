package forklift

import (
	"bytes"
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
		// fmt.Printf("No need to merge pallet %s!\n", shallow.Path())
		return shallow, nil
	}

	// fmt.Printf("Merging pallet %s...\n", shallow.Path())
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
	// fmt.Printf("Merged pallet %s!\n", shallow.Path())
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

	palletFileMappings = make(map[string]map[string]string) // pallet path -> target -> source
	for palletPath, palletResolved := range resolvedByPallet {
		mergedPalletResolved := make([]*ResolvedImport, 0, len(palletResolved))
		for _, resolved := range palletResolved {
			mergedPalletResolved = append(mergedPalletResolved, &ResolvedImport{
				Import: resolved.Import,
				Pallet: pallets[palletPath],
			})
		}
		if palletFileMappings[palletPath], err = consolidatePalletImports(
			mergedPalletResolved, palletLoader,
		); err != nil {
			return nil, nil, errors.Wrapf(
				err, "couldn't evaluate import groups for pallet %s", palletPath,
			)
		}
	}
	return palletFileMappings, pallets, nil
}

// consolidatePalletImports checks the import groups loaded for a single required pallet and
// consolidates into a single mapping between target paths and source paths relative to the
// required pallet.
func consolidatePalletImports(
	imports []*ResolvedImport, loader FSPalletLoader,
) (map[string]string, error) {
	union := make(map[string]string)           // target -> source
	mappingOrigin := make(map[string][]string) // target -> import group names
	for _, imp := range imports {
		importMappings, err := imp.Evaluate(loader)
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

// A FileRef is a reference to a file in a [core.PathedFS] by its file path.
type FileRef struct {
	// Ordered identifiers of the sources of the file reference, e.g. the pallet at the root of the FS
	Sources []string
	// The FS which the file can be loaded from
	FS core.PathedFS
	// The path where the file exists relative to the root of the FS
	Path string
}

// mergePalletImports builds a mapping from all target file paths to their respective source files
// suitable for instantiating a MergeFS.
func mergePalletImports(
	palletFileMappings map[string]map[string]string, pallets map[string]*FSPallet,
) (merged map[string]FileRef, err error) {
	merged = make(map[string]FileRef) // target -> source
	for palletPath, fileMappings := range palletFileMappings {
		for target, source := range fileMappings {
			ref := FileRef{
				Sources: []string{palletPath},
				FS:      pallets[palletPath].FS,
				Path:    strings.TrimPrefix(source, "/"),
			}
			if version := pallets[palletPath].Version; version != "" {
				// i.e. the pallet isn't an override with dirty changes (clean overrides, i.e. those
				// without uncommitted changes, still have a version number attached when they're loaded as
				// overrides by the Forklift CLI)
				ref.Sources[0] += "@" + version
			}
			if fsys, ok := pallets[palletPath].FS.(*MergeFS); ok {
				transitiveRef, err := fsys.getFileRef(strings.TrimPrefix(source, "/"))
				if err != nil {
					return nil, errors.Wrapf(
						err, "couldn't transitively resolve file reference for importing %s from %s",
						strings.TrimPrefix(source, "/"), pallets[palletPath].FS.Path(),
					)
				}
				ref.Sources = slices.Concat(ref.Sources, transitiveRef.Sources)
				ref.FS = transitiveRef.FS
				ref.Path = transitiveRef.Path
			}
			prevRef, ok := merged[strings.TrimPrefix(target, "/")]
			if !ok {
				merged[strings.TrimPrefix(target, "/")] = ref
				continue
			}

			result, err := filesAreIdentical(prevRef, ref)
			if err != nil {
				return nil, errors.Wrapf(
					err, "couldn't check whether source files %s and %s (both mapping to %s) are identical",
					path.Join(ref.FS.Path(), ref.Path), path.Join(prevRef.FS.Path(), prevRef.Path), target,
				)
			}
			if result != nil {
				return nil, errors.Wrapf(
					result, "couldn't add a mapping from %s to target %s, when a mapping was previously "+
						"added from %s to the same target",
					path.Join(ref.FS.Path(), ref.Path), target, path.Join(prevRef.FS.Path(), prevRef.Path),
				)
			}

			if ref.FS.Path() < prevRef.FS.Path() {
				merged[strings.TrimPrefix(target, "/")] = ref
			}
		}
	}
	return merged, nil
}

// filesAreIdentical checks whether the two file references are identical or are referencing files
// with identical file type (dir vs. non-dir), size, permissions, and contents. A non-nil result
// is returned with a nill err if the files are not identical, explaining why the files are not
// identical.
func filesAreIdentical(a, b FileRef) (result error, err error) {
	if a.FS.Path() == b.FS.Path() && a.Path == b.Path {
		return nil, nil
	}

	aInfo, err := fs.Stat(a.FS, a.Path)
	if err != nil {
		return nil, err
	}
	bInfo, err := fs.Stat(b.FS, b.Path)
	if err != nil {
		return nil, err
	}
	if aInfo.IsDir() != bInfo.IsDir() {
		return errors.New("source files have different types (directory vs. non-directory)"), nil
	}
	if aInfo.Size() != bInfo.Size() {
		return errors.Errorf(
			"source files have different sizes (%d vs. %d)", aInfo.Size(), bInfo.Size(),
		), nil
	}
	if aInfo.Mode().Perm() != bInfo.Mode().Perm() {
		return errors.Errorf(
			"source files have different permissions (%s vs. %s)",
			aInfo.Mode().Perm(), bInfo.Mode().Perm(),
		), nil
	}

	// Note: we load both files entirely into memory because that's simpler. If memory constraints
	// or performance requirements eventually make this a problem, we can optimize this later by
	// comparing bytes as we read them from the files.
	aBytes, err := fs.ReadFile(a.FS, a.Path)
	if err != nil {
		return nil, err
	}
	bBytes, err := fs.ReadFile(b.FS, b.Path)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(aBytes, bBytes) {
		return errors.New("source files have different contents"), nil
	}
	return nil, nil
}

// MergeFS

// A MergeFS is an FS constructed by combining a [core.PathedFS] as an overlay over an underlay
// constructed as a collection of references to files in other [core.PathedFS] instances.
// The path of the FS is the path of the overlay.
type MergeFS struct {
	// Ordered identifiers of the source(s) of the MergeFS, e.g. the pallet at the root of the FS
	Overlay      core.PathedFS
	underlayRefs map[string]FileRef     // target -> source
	impliedDirs  structures.Set[string] // target
}

func newMergeFS(overlay core.PathedFS, underlayRefs map[string]FileRef) *MergeFS {
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
	ref, err := f.getFileRef(name)
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

func (f *MergeFS) getFileRef(name string) (FileRef, error) {
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
	underlayRefsSub := make(map[string]FileRef)
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
