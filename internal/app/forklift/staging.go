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

	fbun "github.com/forklift-run/forklift/pkg/bundling"
	ffs "github.com/forklift-run/forklift/pkg/fs"
)

// FSStageStore

// EnsureFSStageStore initializes a FSStageStore at the specified directory path in the provided
// base filesystem, if a stage store is not already initialized there.
func EnsureFSStageStore(fsys ffs.PathedFS, subdirPath, newStateStoreVersion string) error {
	storePath := path.Join(fsys.Path(), subdirPath)
	if err := ffs.EnsureExists(filepath.FromSlash(storePath)); err != nil {
		return errors.Wrapf(
			err, "couldn't ensure the existence of the stage store at %s", storePath,
		)
	}
	if _, err := fs.Stat(
		fsys, path.Join(subdirPath, StageStoreManifestFile),
	); errors.Is(err, fs.ErrNotExist) {
		manifest := StageStoreManifest{
			ForkliftVersion: newStateStoreVersion,
		}
		manifestPath := path.Join(fsys.Path(), subdirPath, StageStoreManifestFile)
		if err = manifest.Write(manifestPath); err != nil {
			return errors.Wrapf(err, "couldn't initialize stage store manifest file at %s", manifestPath)
		}
	}
	return nil
}

// LoadFSStageStore loads a FSStageStore from the specified directory path in the provided base
// filesystem.
func LoadFSStageStore(fsys ffs.PathedFS, subdirPath string) (s *FSStageStore, err error) {
	s = &FSStageStore{}
	if s.FS, err = fsys.Sub(subdirPath); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't enter directory %s from fs at %s", subdirPath, fsys.Path(),
		)
	}
	// FIXME: error out if there's a swap directory!
	if s.Manifest, err = loadStageStoreManifest(s.FS, StageStoreManifestFile); err != nil {
		return nil, errors.Errorf("couldn't load stage store state")
	}
	return s, nil
}

// Exists checks whether the store actually exists on the OS's filesystem.
func (s *FSStageStore) Exists() bool {
	return ffs.DirExists(filepath.FromSlash(s.FS.Path()))
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
func (s *FSStageStore) LoadFSBundle(index int) (*fbun.FSBundle, error) {
	return fbun.LoadFSBundle(s.FS, fmt.Sprintf("%d", index))
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

// IdentifyHighest identifies the staged pallet in the store with the highest index. Only positive
// indices are considered.
// If the cache is empty, an error is returned with zero as an index.
func (s *FSStageStore) IdentifyHighest() (index int, err error) {
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
	prevIndex, _ := s.IdentifyHighest()
	// Warning: we're assuming that no pallets have been staged so far if we can't identify the last
	// staged pallet. This might be an invalid assumption?
	// Note: if no pallets have been staged so far, the first index we allow is 1. This way, a "0"
	// index (i.e. Go's default-initialization for an int) can represent a missing index.
	index = prevIndex + 1
	newPath := filepath.FromSlash(s.GetBundlePath(index))
	if ffs.DirExists(newPath) {
		return index, errors.Wrapf(err, "a stage already exists at %s", newPath)
	}
	if err = ffs.EnsureExists(newPath); err != nil {
		return index, errors.Wrapf(err, "couldn't ensure the existence of %s", newPath)
	}
	return index, nil
}

// GetBundlePath returns the full filesystem path of the pallet bundle at the specified index,
// whether or not a bundle actually exists on the filesystem at that index.
func (s *FSStageStore) GetBundlePath(index int) string {
	return path.Join(s.FS.Path(), fmt.Sprintf("%d", index))
}

// SetNext sets the specified stage as the next one to be applied and resets the flag tracking
// whether the next stage to be applied has failed. It assumes that the specified
// stage actually exists. Setting a value of 0 will clear the state of the next stage to be applied,
// so no stage will be applied next.
func (s *FSStageStore) SetNext(index int) {
	s.Manifest.Stages.NextFailed = false
	s.Manifest.Stages.Next = index
}

// GetNext returns the next stage to be applied. It returns not-`ok` if no stage is to be applied.
func (s *FSStageStore) GetNext() (index int, ok bool) {
	return s.Manifest.Stages.Next, s.Manifest.Stages.Next > 0
}

// GetCurrent returns the last stage which was successfully applied. It returns not-`ok` if no
// stage has been successfully applied so far.
func (s *FSStageStore) GetCurrent() (index int, ok bool) {
	if len(s.Manifest.Stages.History) == 0 {
		return 0, false
	}
	return s.Manifest.Stages.History[len(s.Manifest.Stages.History)-1], true
}

// GetPending returns the next stage to be applied, if it's different from the last stage which was
// successfully applied. It returns not-`ok` if there is no next stage to be applied or if the two
// stages are identical.
func (s *FSStageStore) GetPending() (index int, ok bool) {
	current, _ := s.GetCurrent()
	next, hasNext := s.GetNext()
	if !hasNext {
		return 0, false
	}
	return next, current != next
}

// GetRollback returns the previous stage which was successfully applied before the last stage which
// was successfully applied. It returns not-`ok` if no such stage exists.
func (s *FSStageStore) GetRollback() (index int, ok bool) {
	const rollbackOffset = 1
	if len(s.Manifest.Stages.History) < rollbackOffset+1 {
		return 0, false
	}
	return s.Manifest.Stages.History[len(s.Manifest.Stages.History)-1-rollbackOffset], true
}

// RecordNextSuccess records the whether stage which was to be applied had a successful application.
func (s *FSStageStore) RecordNextSuccess(succeeded bool) {
	if s.Manifest.Stages.Next == 0 {
		return
	}
	s.Manifest.Stages.NextFailed = !succeeded
	if !succeeded {
		return
	}
	if current, ok := s.GetCurrent(); ok && s.Manifest.Stages.Next == current {
		return
	}
	s.Manifest.Stages.History = append(s.Manifest.Stages.History, s.Manifest.Stages.Next)
}

// NextFailed returns whether the next stage to be applied has encountered a failed application.
func (s *FSStageStore) NextFailed() bool {
	return s.Manifest.Stages.NextFailed
}

// RemoveBundleNames removes all names for the specified bundle.
func (s *FSStageStore) RemoveBundleNames(index int) {
	for name, namedIndex := range s.Manifest.Stages.Names {
		if index != namedIndex {
			continue
		}
		delete(s.Manifest.Stages.Names, name)
	}
}

// RemoveBundleHistory removes the specified bundle from the history.
func (s *FSStageStore) RemoveBundleHistory(index int) {
	newHistory := make([]int, 0, len(s.Manifest.Stages.History))
	for _, historyIndex := range s.Manifest.Stages.History {
		if index == historyIndex {
			continue
		}
		newHistory = append(newHistory, historyIndex)
	}
	s.Manifest.Stages.History = newHistory
}

// CommitState atomically updates the stage store's manifest file.
// Warning: on non-Unix platforms, the update is not entirely atomic!
func (s *FSStageStore) CommitState() error {
	// TODO: we might want to be less sloppy about read locks vs. write locks in the future. After
	// successfully acquiring a write lock, then we could just overwrite the swap file.
	swapPath := path.Join(s.FS.Path(), StageStoreManifestSwapFile)
	if FileExists(filepath.FromSlash(swapPath)) {
		return errors.Errorf(
			"stage store manifest swap file %s already exists, so either another operation is "+
				"currently running or the previous operation failed or was interrupted before it could "+
				"finish; please ensure that no other operations are currently running and delete the swap "+
				"file before retrying",
			swapPath,
		)
	}
	if err := s.Manifest.Write(swapPath); err != nil {
		return errors.Wrapf(err, "couldn't save stage store to swap file %s", swapPath)
	}
	outputPath := path.Join(s.FS.Path(), StageStoreManifestFile)
	// Warning: on non-Unix platforms, os.Rename is not an atomic operation! So if the program dies
	// during the os.Rename call, we could end up breaking the state of the stage store.
	if err := os.Rename(filepath.FromSlash(swapPath), filepath.FromSlash(outputPath)); err != nil {
		return errors.Wrapf(
			err, "couldn't commit stage store update from %s to %s", swapPath, outputPath,
		)
	}
	return nil
}

// StageStoreManifest

// loadStageStoreManifest loads a StageStoreManifest from the specified file path in the provided
// base filesystem.
func loadStageStoreManifest(fsys ffs.PathedFS, filePath string) (StageStoreManifest, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return StageStoreManifest{}, errors.Wrapf(
			err, "couldn't read stage store manifest file %s/%s", fsys.Path(), filePath,
		)
	}
	config := StageStoreManifest{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return StageStoreManifest{}, errors.Wrap(err, "couldn't parse stage store state")
	}
	if config.Stages.Names == nil {
		config.Stages.Names = make(map[string]int)
	}
	return config, nil
}

func (m StageStoreManifest) Write(outputPath string) error {
	marshaled, err := yaml.Marshal(m)
	if err != nil {
		return errors.Wrapf(err, "couldn't marshal stage store state")
	}
	const perm = 0o644 // owner rw, group r, public r
	if err = os.WriteFile(filepath.FromSlash(outputPath), marshaled, perm); err != nil {
		return errors.Wrapf(err, "couldn't save stage store to %s", outputPath)
	}
	return nil
}
