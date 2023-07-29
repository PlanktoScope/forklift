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

// MissingDeplDependencies

func (c MissingDeplDependencies) HasMissingNetworkDependency() bool {
	return len(c.Networks) > 0
}

func (c MissingDeplDependencies) HasMissingServiceDependency() bool {
	return len(c.Services) > 0
}

func (c MissingDeplDependencies) HasMissingDependency() bool {
	return c.HasMissingNetworkDependency() || c.HasMissingServiceDependency()
}
