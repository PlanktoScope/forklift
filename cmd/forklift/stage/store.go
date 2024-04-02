package stage

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
)

var errMissingStore = errors.Errorf(
	"you first need to stage a pallet, e.g. with `forklift plt stage`",
)

func getStageStore(wpath string, ensureWorkspace bool) (*forklift.FSStageStore, error) {
	if ensureWorkspace {
		if !forklift.Exists(wpath) {
			fmt.Printf("Making a new workspace at %s...", wpath)
		}
		if err := forklift.EnsureExists(wpath); err != nil {
			return nil, errors.Wrapf(err, "couldn't make new workspace at %s", wpath)
		}
	}
	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	cache, err := workspace.GetStageStore()
	if err != nil {
		return nil, err
	}
	return cache, nil
}
