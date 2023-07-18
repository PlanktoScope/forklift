package forklift

// Conflicts

func CheckResourcesConflicts[Resource ConflictChecker[Resource]](
	first []AttachedResource[Resource], second []AttachedResource[Resource],
) (conflicts []ResourceConflict[Resource]) {
	for _, f := range first {
		for _, s := range second {
			if errs := f.Resource.CheckConflict(s.Resource); errs != nil {
				conflicts = append(conflicts, ResourceConflict[Resource]{
					First:  f,
					Second: s,
					Errs:   errs,
				})
			}
		}
	}
	return conflicts
}

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

// Dependencies

func (c MissingDeplDependencies) HasMissingNetworkDependency() bool {
	return len(c.Networks) > 0
}

func (c MissingDeplDependencies) HasMissingServiceDependency() bool {
	return len(c.Services) > 0
}

func (c MissingDeplDependencies) HasMissingDependency() bool {
	return c.HasMissingNetworkDependency() || c.HasMissingServiceDependency()
}

func CheckResourcesDependencies[Resource DependencyChecker[Resource]](
	required []AttachedResource[Resource], provided []AttachedResource[Resource],
) (missingDeps []MissingResourceDependency[Resource]) {
	for _, r := range required {
		bestErrsCount := -1
		bestCandidates := make([]ResourceDependencyCandidate[Resource], 0, len(provided))
		for _, p := range provided {
			errs := r.Resource.CheckDependency(p.Resource)
			if bestErrsCount == -1 || len(errs) <= bestErrsCount {
				if len(errs) < bestErrsCount {
					bestCandidates = make([]ResourceDependencyCandidate[Resource], 0, len(provided))
				}
				bestErrsCount = len(errs)
				bestCandidates = append(bestCandidates, ResourceDependencyCandidate[Resource]{
					Provided: p,
					Errs:     errs,
				})
			}
		}
		if bestErrsCount != 0 {
			missingDeps = append(missingDeps, MissingResourceDependency[Resource]{
				Required:       r,
				BestCandidates: bestCandidates,
			})
		}
	}
	return missingDeps
}
