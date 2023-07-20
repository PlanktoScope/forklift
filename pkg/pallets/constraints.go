package pallets

// AttachedResource

func NewAttachedResource[Resource any](
	resource Resource, source []string,
) AttachedResource[Resource] {
	return AttachedResource[Resource]{
		Resource: resource,
		Source:   source,
	}
}

func attachResources[Resource any](
	resources []Resource, source []string,
) (attached []AttachedResource[Resource]) {
	attached = make([]AttachedResource[Resource], 0, len(resources))
	for _, resource := range resources {
		attached = append(attached, NewAttachedResource(resource, source))
	}
	return attached
}

// ResourceConflict

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

// MissingResourceDependency

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
