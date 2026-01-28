package forklift

import (
	"bytes"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/pkg/errors"

	ffs "github.com/forklift-run/forklift/pkg/fs"
	"github.com/forklift-run/forklift/pkg/structures"
)

// MergeFSPallet creates a new FSPallet with a virtual (read-only) filesystem created by evaluating
// the pallet's file imports with its required pallets (which should be loadable using the provided
// loader).
func MergeFSPallet(
	shallow *FSPallet, palletLoader FSPalletLoader, prohibitedPallets structures.Set[string],
) (merged *FSPallet, err error) {
	merged = &FSPallet{
		Pallet:    shallow.Pallet,
		FSPkgTree: shallow.FSPkgTree,
	}
	imports, err := shallow.LoadImports("**/*")
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't check for import groups")
	}
	hasEnabledImports := false
	for _, imp := range imports {
		if !imp.Decl.Disabled {
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
	merged.FS = ffs.NewMergeFS(shallow.FS, underlayRefs)
	merged.FSPkgTree.FS = merged.FS
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

// mergePalletImports builds a mapping from all target file paths to their respective source files
// suitable for instantiating a MergeFS.
func mergePalletImports(
	palletFileMappings map[string]map[string]string, pallets map[string]*FSPallet,
) (merged map[string]ffs.FileRef, err error) {
	merged = make(map[string]ffs.FileRef) // target -> source
	for palletPath, fileMappings := range palletFileMappings {
		for target, source := range fileMappings {
			ref := ffs.FileRef{
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
			if fsys, ok := pallets[palletPath].FS.(*ffs.MergeFS); ok {
				transitiveRef, err := fsys.GetFileRef(strings.TrimPrefix(source, "/"))
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
func filesAreIdentical(a, b ffs.FileRef) (result error, err error) {
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
