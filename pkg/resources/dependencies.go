package resources

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

// A DepChecker is something which may depend on a resource, and which can determine whether
// its dependency is satisfied by a resource. Typically, resource requirement types will implement
// this interface.
type DepChecker[Res any, Origin any] interface {
	CheckDep(candidate Res) []error
}

// A SatisfiedDep is a report of a resource requirement which is satisfied by a set of resources.
type SatisfiedDep[Res any, Origin any] struct {
	// Required is the resource requirement.
	Required Attached[Res, Origin]
	// Provided is the resource which satisfies the resource requirement.
	Provided Attached[Res, Origin]
}

// A MissingDep is a report of a resource requirement which is not satisfied by any
// resources.
type MissingDep[Res any, Origin any] struct {
	// Required is the resource requirement.
	Required Attached[Res, Origin]
	// BestCandidates is a list of the resources which are closest to satisfying the resource
	// requirement.
	BestCandidates []DepCandidate[Res, Origin]
}

// DepCandidate is a report of a resource which either satisfied a resource
// requirement or (if Errs contains errors) failed to satisfy that resource requirement.
type DepCandidate[Res any, Origin any] struct {
	// Provided is the resource which did not satisfy the requirement.
	Provided Attached[Res, Origin]
	// Errs is a list of errors describing how the resource did not satisfy the requirement.
	Errs []error
}
