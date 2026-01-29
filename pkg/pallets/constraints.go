package forklift

import (
	"cmp"

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

type SatisfiedDeplDeps struct {
	Depl *ResolvedDepl

	Networks []res.SatisfiedDep[fpkg.NetworkRes, []string]
	Services []res.SatisfiedDep[fpkg.ServiceRes, []string]
	Filesets []res.SatisfiedDep[fpkg.FilesetRes, []string]
}

type MissingDeplDeps struct {
	Depl *ResolvedDepl

	Networks []res.MissingDep[fpkg.NetworkRes, []string]
	Services []res.MissingDep[fpkg.ServiceRes, []string]
	Filesets []res.MissingDep[fpkg.FilesetRes, []string]
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

// SatisfiedDeplDeps

func (d SatisfiedDeplDeps) HasSatisfiedNetworkDep() bool {
	return len(d.Networks) > 0
}

func (d SatisfiedDeplDeps) HasSatisfiedServiceDep() bool {
	return len(d.Services) > 0
}

func (d SatisfiedDeplDeps) HasSatisfiedFilesetDep() bool {
	return len(d.Filesets) > 0
}

func (d SatisfiedDeplDeps) HasSatisfiedDep() bool {
	return cmp.Or(
		d.HasSatisfiedNetworkDep(),
		d.HasSatisfiedServiceDep(),
		d.HasSatisfiedFilesetDep(),
	)
}

// MissingDeplDeps

func (d MissingDeplDeps) HasMissingNetworkDep() bool {
	return len(d.Networks) > 0
}

func (d MissingDeplDeps) HasMissingServiceDep() bool {
	return len(d.Services) > 0
}

func (d MissingDeplDeps) HasMissingFilesetDep() bool {
	return len(d.Filesets) > 0
}

func (d MissingDeplDeps) HasMissingDep() bool {
	return cmp.Or(
		d.HasMissingNetworkDep(),
		d.HasMissingServiceDep(),
		d.HasMissingFilesetDep(),
	)
}
