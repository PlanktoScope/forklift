package resources

// An Attached is a binding between a resource and the origin of that resource.
type Attached[Res any, Origin any] struct {
	// Res is a resource subject to various possible constraints.
	Res Res
	// Origin is describes how to locate the resource in a package spec, e.g. so that it can be shown
	// to users when a resource constraint is not met or when a resource conflict exists.
	Origin Origin
}

// Resource conflicts

// A ConflictChecker is something which may conflict with a resource, and which can determine
// whether it conflicts with another resource. Typically, resource types will implement this
// interface.
type ConflictChecker[Res any] interface {
	CheckConflict(candidate Res) []error
}

// A Conflict is a report of a conflict between two resources.
type Conflict[Res any, Origin any] struct {
	// First is one of the two conflicting resources.
	First Attached[Res, Origin]
	// Second is the other of the two conflicting resources.
	Second Attached[Res, Origin]
	// Errs is a list of errors describing how the two resources conflict with each other.
	Errs []error
}

// Resource dependencies

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
