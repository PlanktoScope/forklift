package pallets

// AttachedResource

// attachResources attaches the specified source to all of the specified resources.
func attachResources[Resource any](
	resources []Resource, source []string,
) (attached []AttachedResource[Resource]) {
	attached = make([]AttachedResource[Resource], 0, len(resources))
	for _, resource := range resources {
		attached = append(attached, AttachedResource[Resource]{
			Resource: resource,
			Source:   source,
		})
	}
	return attached
}

// Resource conflicts

// CheckResourcesConflicts identifies all resource conflicts between the first list of resources and
// the second list of resources. It does not identify resource conflicts within the first list of
// resources, nor within the second list of resources.
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

// Resource dependencies

// CheckResourcesDependencies identifies all unsatisfied resource dependencies between the provided
// list of resource requirements and the provided list of resources.
func CheckResourcesDependencies[Resource DependencyChecker[Resource]](
	required []AttachedResource[Resource], provided []AttachedResource[Resource],
) (
	satisfied []SatisfiedResourceDependency[Resource], missing []MissingResourceDependency[Resource],
) {
	for _, r := range required {
		bestErrsCount := -1
		bestCandidates := make([]ResourceDependencyCandidate[Resource], 0, len(provided))
		for i, p := range provided {
			errs := r.Resource.CheckDependency(p.Resource)
			if bestErrsCount != -1 && len(errs) > bestErrsCount {
				continue
			}
			if bestErrsCount == -1 || len(errs) < bestErrsCount {
				// we've found a provided resource which is strictly better than all previous candidates
				bestErrsCount = len(errs)
				bestCandidates = make([]ResourceDependencyCandidate[Resource], 0, len(provided)-i)
			}
			bestCandidates = append(bestCandidates, ResourceDependencyCandidate[Resource]{
				Provided: p,
				Errs:     errs,
			})
		}
		if bestErrsCount != 0 {
			missing = append(missing, MissingResourceDependency[Resource]{
				Required:       r,
				BestCandidates: bestCandidates,
			})
			continue
		}
		satisfied = append(satisfied, SatisfiedResourceDependency[Resource]{
			Required: r,
			Provided: bestCandidates[0].Provided,
		})
	}
	return satisfied, missing
}
