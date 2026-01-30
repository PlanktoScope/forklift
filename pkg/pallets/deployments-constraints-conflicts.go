package pallets

import (
	"cmp"

	"github.com/pkg/errors"

	fpkg "github.com/forklift-run/forklift/pkg/packaging"
	res "github.com/forklift-run/forklift/pkg/resources"
)

type DeplConflict struct {
	First  *ResolvedDepl
	Second *ResolvedDepl

	// Possible conflicts
	Name        bool
	Listeners   []res.Conflict[fpkg.ListenerRes, []string]
	Networks    []res.Conflict[fpkg.NetworkRes, []string]
	Services    []res.Conflict[fpkg.ServiceRes, []string]
	Filesets    []res.Conflict[fpkg.FilesetRes, []string]
	FileExports []res.Conflict[fpkg.FileExportRes, []string]
}

// DeplConflict

func (c DeplConflict) HasNameConflict() bool {
	return c.Name
}

func (c DeplConflict) HasListenerConflict() bool {
	return len(c.Listeners) > 0
}

func (c DeplConflict) HasNetworkConflict() bool {
	return len(c.Networks) > 0
}

func (c DeplConflict) HasServiceConflict() bool {
	return len(c.Services) > 0
}

func (c DeplConflict) HasFilesetConflict() bool {
	return len(c.Filesets) > 0
}

func (c DeplConflict) HasFileExportConflict() bool {
	return len(c.FileExports) > 0
}

func (c DeplConflict) HasConflict() bool {
	return cmp.Or(
		c.HasNameConflict(),
		c.HasListenerConflict(),
		c.HasNetworkConflict(),
		c.HasServiceConflict(),
		c.HasFilesetConflict(),
		c.HasFileExportConflict(),
	)
}

// ResolvedDepl: Constraints: Resource Conflicts

// CheckConflicts produces a report of all resource conflicts between the ResolvedDepl instance and
// a candidate ResolvedDepl.
func (d *ResolvedDepl) CheckConflicts(candidate *ResolvedDepl) (DeplConflict, error) {
	enabledFeatures, err := d.EnabledFeatures()
	if err != nil {
		return DeplConflict{}, errors.Wrapf(
			err, "couldn't determine enabled features of deployment %s", d.Name,
		)
	}
	candidateEnabledFeatures, err := candidate.EnabledFeatures()
	if err != nil {
		return DeplConflict{}, errors.Wrapf(
			err, "couldn't determine enabled features of deployment %s", candidate.Name,
		)
	}
	return DeplConflict{
		First:  d,
		Second: candidate,
		Name:   d.Name == candidate.Name,
		Listeners: res.CheckConflicts(
			d.providedListeners(enabledFeatures), candidate.providedListeners(candidateEnabledFeatures),
		),
		Networks: res.CheckConflicts(
			d.providedNetworks(enabledFeatures), candidate.providedNetworks(candidateEnabledFeatures),
		),
		Services: res.CheckConflicts(
			d.providedServices(enabledFeatures), candidate.providedServices(candidateEnabledFeatures),
		),
		Filesets: res.CheckConflicts(
			d.providedFilesets(enabledFeatures), candidate.providedFilesets(candidateEnabledFeatures),
		),
		// FIXME: for some reason, the checker doesn't detect a conflict between one depl which exports
		// a non-directory file at a certain path and another depl which exports a file into a
		// subdirectory of that same path (i.e. which assumes that the path is a directory)
		FileExports: res.CheckConflicts(
			d.providedFileExports(enabledFeatures),
			candidate.providedFileExports(candidateEnabledFeatures),
		),
	}, nil
}

// CheckAllConflicts produces a slice of reports of all resource conflicts between the ResolvedDepl
// instance and each candidate ResolvedDepl.
func (d *ResolvedDepl) CheckAllConflicts(
	candidates []*ResolvedDepl,
) (conflicts []DeplConflict, err error) {
	conflicts = make([]DeplConflict, 0, len(candidates))
	for _, candidate := range candidates {
		conflict, err := d.CheckConflicts(candidate)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't check conflicts with deployment %s", candidate.Name)
		}
		if conflict.HasConflict() {
			conflicts = append(conflicts, conflict)
		}
	}
	return conflicts, nil
}

// Checking

// CheckDeplConflicts produces a slice of reports of all resource conflicts among all provided
// ResolvedDepl instances.
func CheckDeplConflicts(depls []*ResolvedDepl) (conflicts []DeplConflict, err error) {
	for i, depl := range depls {
		deplConflicts, err := depl.CheckAllConflicts(depls[i+1:])
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't check for conflicts with deployment %s", depl.Name)
		}
		conflicts = append(conflicts, deplConflicts...)
	}
	return conflicts, nil
}
