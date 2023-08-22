package core

// AttachedRes

// attachRes attaches the specified source to all of the specified resources.
func attachRes[Res any](
	resources []Res, source []string,
) (attached []AttachedRes[Res]) {
	attached = make([]AttachedRes[Res], 0, len(resources))
	for _, resource := range resources {
		attached = append(attached, AttachedRes[Res]{
			Res:    resource,
			Source: source,
		})
	}
	return attached
}

// Resource conflicts

// CheckResConflicts identifies all resource conflicts between the first list of resources and
// the second list of resources. It does not identify resource conflicts within the first list of
// resources, nor within the second list of resources.
func CheckResConflicts[Res ConflictChecker[Res]](
	first []AttachedRes[Res], second []AttachedRes[Res],
) (conflicts []ResConflict[Res]) {
	for _, f := range first {
		for _, s := range second {
			if errs := f.Res.CheckConflict(s.Res); errs != nil {
				conflicts = append(conflicts, ResConflict[Res]{
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

// CheckResDeps identifies all unsatisfied resource dependencies between the provided
// list of resource requirements and the provided list of resources.
func CheckResDeps[Res DepChecker[Res]](
	required []AttachedRes[Res], provided []AttachedRes[Res],
) (
	satisfied []SatisfiedResDep[Res], missing []MissingResDep[Res],
) {
	for _, r := range required {
		bestErrsCount := -1
		bestCandidates := make([]ResDepCandidate[Res], 0, len(provided))
		for i, p := range provided {
			errs := r.Res.CheckDep(p.Res)
			if bestErrsCount != -1 && len(errs) > bestErrsCount {
				continue
			}
			if bestErrsCount == -1 || len(errs) < bestErrsCount {
				// we've found a provided resource which is strictly better than all previous candidates
				bestErrsCount = len(errs)
				bestCandidates = make([]ResDepCandidate[Res], 0, len(provided)-i)
			}
			bestCandidates = append(bestCandidates, ResDepCandidate[Res]{
				Provided: p,
				Errs:     errs,
			})
		}
		if bestErrsCount != 0 {
			missing = append(missing, MissingResDep[Res]{
				Required:       r,
				BestCandidates: bestCandidates,
			})
			continue
		}
		satisfied = append(satisfied, SatisfiedResDep[Res]{
			Required: r,
			Provided: bestCandidates[0].Provided,
		})
	}
	return satisfied, missing
}
