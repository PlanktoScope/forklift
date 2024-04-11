package forklift

import (
	"cmp"
)

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
