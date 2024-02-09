package forklift

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

func (c DeplConflict) HasConflict() bool {
	return c.HasNameConflict() ||
		c.HasListenerConflict() || c.HasNetworkConflict() || c.HasServiceConflict() ||
		c.HasFilesetConflict()
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
	return d.HasSatisfiedNetworkDep() || d.HasSatisfiedServiceDep() || d.HasSatisfiedFilesetDep()
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
	return d.HasMissingNetworkDep() || d.HasMissingServiceDep() || d.HasMissingFilesetDep()
}
