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
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/pkg/core"
)

// FSStageStore

// loadFSStageStore loads a FSStageStore from the specified directory path in the provided base
// filesystem.
func loadFSStageStore(fsys core.PathedFS, subdirPath string) (s *FSStageStore, err error) {
	s = &FSStageStore{}
	if s.FS, err = fsys.Sub(subdirPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", subdirPath, fsys.Path(),
		)
	}
	if s.Def, err = loadStageStoreDef(s.FS, StageStoreDefFile); err != nil {
		return nil, errors.Errorf("couldn't load stage store state")
	}
	return s, nil
}

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
func (s *FSStageStore) LoadFSBundle(index int) (*FSBundle, error) {
	return LoadFSBundle(s.FS, fmt.Sprintf("%d", index))
}

// List returns a numerically-sorted (in ascending order) list of staged pallet bundles in the
// store. Only positive indices are included.
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
		if index <= 0 {
			continue
		}
		indices = append(indices, index)
	}
	slices.Sort(indices)
	return indices, nil
}

// IdentifyLast identifies the staged pallet in the store with the highest index. Only positive
// indices are considered.
// If the cache is empty, an error is returned with zero as an index.
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
	prevIndex, _ := s.IdentifyLast()
	// Warning: we're assuming that no pallets have been staged so far if we can't identify the last
	// staged pallet. This might be an invalid assumption?
	// Note: if no pallets have been staged so far, the first index we allow is 1. This way, a "0"
	// index (i.e. Go's default-initialization for an int) can represent a missing index.
	index = prevIndex + 1
	newPath := filepath.FromSlash(path.Join(s.FS.Path(), fmt.Sprintf("%d", index)))
	if Exists(newPath) {
		return index, errors.Wrapf(err, "a stage already exists at %s", newPath)
	}
	if err = EnsureExists(newPath); err != nil {
		return index, errors.Wrapf(err, "couldn't ensure the existence of %s", newPath)
	}
	return index, nil
}

// SetNext sets the specified stage as the next one to be applied.
func (s *FSStageStore) SetNext(index int) {
	s.Def.Stages.Next = index
}

// GetNext returns the next stage to be applied. It returns not-`ok` if no stage has been
// is to be applied.
func (s *FSStageStore) GetNext() (index int, ok bool) {
	return s.Def.Stages.Next, s.Def.Stages.Next > 0
}

// GetCurrent returns the last stage which was successfully applied. It returns not-`ok` if no
// stage has been successfully applied so far.
func (s *FSStageStore) GetCurrent() (index int, ok bool) {
	if len(s.Def.Stages.History) == 0 {
		return 0, false
	}
	return s.Def.Stages.History[len(s.Def.Stages.History)-1], true
}

// GetRollback returns the previous stage which was successfully applied before the last stage which
// was successfully applied. It returns not-`ok` if no such stage exists.
func (s *FSStageStore) GetRollback() (index int, ok bool) {
	const rollbackOffset = 1
	if len(s.Def.Stages.History) < rollbackOffset+1 {
		return 0, false
	}
	return s.Def.Stages.History[len(s.Def.Stages.History)-1-rollbackOffset], true
}

// RecordNextSuccess records the stage which was to be applied as having successfully been applied.
func (s *FSStageStore) RecordNextSuccess() {
	s.Def.Stages.History = append(s.Def.Stages.History, s.Def.Stages.Next)
}

// CommitState atomically updates the stage store's definition file.
// Warning: on non-Unix platforms, the update is not entirely atomic!
func (s *FSStageStore) CommitState() error {
	// TODO: we might want to be less sloppy about read locks vs. write locks in the future. After
	// successfully acquiring a write lock, then we could just overwrite the swap file.
	swapPath := filepath.FromSlash(path.Join(s.FS.Path(), StageStoreDefSwapFile))
	if Exists(swapPath) {
		return errors.Errorf(
			"stage store swap file %s already exists, so either another operation is currently running "+
				"or the previous operation may have been interrupted before it could finish; please ensure "+
				"that no other operations are currently running and delete the swap file before retrying",
			swapPath,
		)
	}
	if err := s.Def.Write(swapPath); err != nil {
		return errors.Wrapf(err, "couldn't save stage store to swap file %s", swapPath)
	}
	outputPath := filepath.FromSlash(path.Join(s.FS.Path(), StageStoreDefFile))
	// Warning: on non-Unix platforms, os.Rename is not an atomic operation! So if the program dies
	// during the os.Rename call, we could end up breaking the state of the stage store.
	if err := os.Rename(swapPath, outputPath); err != nil {
		return errors.Wrapf(
			err, "couldn't commit stage store update from %s to %s", swapPath, outputPath,
		)
	}
	return nil
}

// StageStoreDef

// loadStageStoreDef loads a stageStoreDef from the specifie file path in the provided base
// filesystem.
func loadStageStoreDef(fsys core.PathedFS, filePath string) (StageStoreDef, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return StageStoreDef{}, errors.Wrapf(
			err, "couldn't read stage store state file %s/%s", fsys.Path(), filePath,
		)
	}
	config := StageStoreDef{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return StageStoreDef{}, errors.Wrap(err, "couldn't parse stage store state")
	}
	if config.Stages.Names == nil {
		config.Stages.Names = make(map[string]int)
	}
	return config, nil
}

func (d StageStoreDef) Write(outputPath string) error {
	marshaled, err := yaml.Marshal(d)
	if err != nil {
		return errors.Wrapf(err, "couldn't marshal stage store state")
	}
	const perm = 0o644 // owner rw, group r, public r
	if err = os.WriteFile(outputPath, marshaled, perm); err != nil {
		return errors.Wrapf(err, "couldn't save stage store to %s", outputPath)
	}
	return nil
}
