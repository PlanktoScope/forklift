package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/core"
)

// Stage Store

const (
	StageStoreDefFile     = "forklift-stage-store.yml"
	StageStoreDefSwapFile = "forklift-stage-store-swap.yml"
)

// FSStageStore is a source of bundles rooted at a single path, with bundles stored as
// incrementally-numbered directories within a [core.PathedFS] filesystem.
type FSStageStore struct {
	// Def is the Forklift stage store definition for the stage store.
	Def StageStoreDef
	// FS is the filesystem which corresponds to the store of staged pallets.
	FS core.PathedFS
}

// A StageStoreDef holds the state of the stage store.
type StageStoreDef struct {
	// ForkliftVersion indicates that the stage store definition was written assuming the semantics of
	// a given version of Forklift. The version must be a valid Forklift version, and it sets the
	// minimum version of Forklift required to use the stage store. The Forklift tool refuses to use
	// stage stores declaring newer Forklift versions for any operations beyond printing information.
	ForkliftVersion string `yaml:"forklift-version"`
	// Stages keeps track of special stages
	Stages StagesSpec `yaml:"staged"`
}

// StagesSpec defines the basic state of a stage store.
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
