package staging

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ffs "github.com/forklift-run/forklift/pkg/fs"
)

const (
	StageStoreManifestFile     = "forklift-stage-store.yml"
	StageStoreManifestSwapFile = "forklift-stage-store-swap.yml"
)

// A StageStoreManifest holds the state of the stage store.
type StageStoreManifest struct {
	// ForkliftVersion indicates that the stage store manifest was written assuming the semantics of
	// a given version of Forklift. The version must be a valid Forklift version, and it sets the
	// minimum version of Forklift required to use the stage store. The Forklift tool refuses to use
	// stage stores declaring newer Forklift versions for any operations beyond printing information.
	ForkliftVersion string `yaml:"forklift-version"`
	// Stages keeps track of special stages
	Stages StagesSpec `yaml:"staged"`
}

// StagesSpec describes the state of a stage store.
type StagesSpec struct {
	// Next is the index of the next staged pallet bundle which should be applied. Once it's applied
	// successfully, it'll be pushed onto the History stack.
	Next int `yaml:"next,omitempty"`
	// NextFailed records whether the next staged pallet bundle had failed to be applied.
	NextFailed bool `yaml:"next-failed,omitempty"`
	// History is the stack of staged pallet bundles which have been applied successfully, with the
	// most-recently-applied bundle last. The most-recently-applied bundle can be used as a fallback
	// If the next staged pallet bundle (if it exists) is not applied successfully.
	History []int `yaml:"history,omitempty"`
	// Names is a list of aliases for staged pallet bundles.
	Names map[string]int `yaml:"names,omitempty"`
}

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
