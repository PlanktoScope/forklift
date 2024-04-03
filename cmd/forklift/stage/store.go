package stage

import (
	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
)

var errMissingStore = errors.Errorf(
	"you first need to stage a pallet, e.g. with `forklift plt stage`",
)

func getStageStore(wpath string, versions Versions) (*forklift.FSStageStore, error) {
	workspace, err := forklift.LoadWorkspace(wpath)
	if err != nil {
		return nil, err
	}
	cache, err := workspace.GetStageStore(versions.NewStageStore)
	if err != nil {
		return nil, err
	}
	return cache, nil
}
