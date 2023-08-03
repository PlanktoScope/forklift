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

func (c DeplConflict) HasConflict() bool {
	return c.HasNameConflict() ||
		c.HasListenerConflict() || c.HasNetworkConflict() || c.HasServiceConflict()
}

// SatisfiedDeplDependencies

func (d SatisfiedDeplDependencies) HasSatisfiedNetworkDependency() bool {
	return len(d.Networks) > 0
}

func (d SatisfiedDeplDependencies) HasSatisfiedServiceDependency() bool {
	return len(d.Services) > 0
}

func (d SatisfiedDeplDependencies) HasSatisfiedDependency() bool {
	return d.HasSatisfiedNetworkDependency() || d.HasSatisfiedServiceDependency()
}

// MissingDeplDependencies

func (d MissingDeplDependencies) HasMissingNetworkDependency() bool {
	return len(d.Networks) > 0
}

func (d MissingDeplDependencies) HasMissingServiceDependency() bool {
	return len(d.Services) > 0
}

func (d MissingDeplDependencies) HasMissingDependency() bool {
	return d.HasMissingNetworkDependency() || d.HasMissingServiceDependency()
}
