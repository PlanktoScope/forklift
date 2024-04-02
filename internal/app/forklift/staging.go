package forklift

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/pkg/errors"
)

// FSStageStore

// Exists checks whether the store actually exists on the OS's filesystem.
func (s *FSStageStore) Exists() bool {
	return Exists(filepath.FromSlash(s.FS.Path()))
}

// Remove deletes the store from the OS's filesystem, if it exists.
func (s *FSStageStore) Remove() error {
	return os.RemoveAll(filepath.FromSlash(s.FS.Path()))
}

// Path returns the path of the store's filesystem.
func (s *FSStageStore) Path() string {
	return s.FS.Path()
}

// LoadFSBundle loads the FSBundle with the specified index.
// The loaded FSBundle instance is fully initialized.
func (s *FSStageStore) LoadFSBundle(repoPath string, version string) (*FSBundle, error) {
	return nil, errors.New("Unimplemented")
}

// List returns a numerically-sorted (in ascending order) list of staged pallet bundles in the
// store. Only negative indices are included.
func (s *FSStageStore) List() (indices []int, err error) {
	indices = make([]int, 0)
	dirEntries, err := fs.ReadDir(s.FS, ".")
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't list staged pallet bundles in %s", s.FS.Path())
	}
	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}
		index, err := strconv.Atoi(dirEntry.Name())
		if err != nil { // i.e. directory is not a staged pallet
			continue
		}
		if index < 0 {
			continue
		}
		indices = append(indices, index)
	}
	slices.Sort(indices)
	return indices, nil
}

// IdentifyLast identifies the staged pallet in the store with the highest index. Only nonnegative
// indices are considered.
// If the cache is empty, an error is returned instead.
func (s *FSStageStore) IdentifyLast() (index int, err error) {
	indices, err := s.List()
	if err != nil {
		return 0, err
	}
	if len(indices) == 0 {
		return 0, errors.New("No staged pallet bundles were found in the store")
	}
	return slices.Max(indices), nil
}

// AllocateNew creates a new directory for a staged pallet in the store with a new highest
// index.
func (s *FSStageStore) AllocateNew() (index int, err error) {
	index = 0
	if prevIndex, err := s.IdentifyLast(); err == nil {
		// We assume that no pallets have been staged so far if we can't identify the last staged
		// pallet. This might be an invalid assumption?
		index = prevIndex + 1
	}
	newPath := filepath.FromSlash(path.Join(s.FS.Path(), fmt.Sprintf("%d", index)))
	if err = EnsureExists(newPath); err != nil {
		return index, errors.Wrapf(err, "couldn't ensure the existence of %s", newPath)
	}
	return index, nil
}
