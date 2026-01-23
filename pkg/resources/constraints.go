package resources

// Attached

// Attach attaches the specified origin to all of the specified resources.
func Attach[Res any, Origin any](
	resources []Res, origin Origin,
) (attached []Attached[Res, Origin]) {
	attached = make([]Attached[Res, Origin], 0, len(resources))
	for _, resource := range resources {
		attached = append(attached, Attached[Res, Origin]{
			Res:    resource,
			Origin: origin,
		})
	}
	return attached
}

// Resource conflicts

// CheckConflicts identifies all resource conflicts between the first list of resources and
// the second list of resources. It does not identify resource conflicts within the first list of
// resources, nor within the second list of resources.
func CheckConflicts[Res ConflictChecker[Res], Origin any](
	first []Attached[Res, Origin], second []Attached[Res, Origin],
) (conflicts []Conflict[Res, Origin]) {
	for _, f := range first {
		for _, s := range second {
			if errs := f.Res.CheckConflict(s.Res); errs != nil {
				conflicts = append(conflicts, Conflict[Res, Origin]{
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

// CheckDeps identifies all unsatisfied resource dependencies between the provided
// list of resource requirements and the provided list of resources.
func CheckDeps[Res DepChecker[Res, Origin], Origin any](
	required []Attached[Res, Origin], provided []Attached[Res, Origin],
) (
	satisfied []SatisfiedDep[Res, Origin], missing []MissingDep[Res, Origin],
) {
	for _, r := range required {
		bestErrsCount := -1
		bestCandidates := make([]DepCandidate[Res, Origin], 0, len(provided))
		for i, p := range provided {
			errs := r.Res.CheckDep(p.Res)
			if bestErrsCount != -1 && len(errs) > bestErrsCount {
				continue
			}
			if bestErrsCount == -1 || len(errs) < bestErrsCount {
				// we've found a provided resource which is strictly better than all previous candidates
				bestErrsCount = len(errs)
				bestCandidates = make([]DepCandidate[Res, Origin], 0, len(provided)-i)
			}
			bestCandidates = append(bestCandidates, DepCandidate[Res, Origin]{
				Provided: p,
				Errs:     errs,
			})
		}
		if bestErrsCount != 0 {
			missing = append(missing, MissingDep[Res, Origin]{
				Required:       r,
				BestCandidates: bestCandidates,
			})
			continue
		}
		satisfied = append(satisfied, SatisfiedDep[Res, Origin]{
			Required: r,
			Provided: bestCandidates[0].Provided,
		})
	}
	return satisfied, missing
}
