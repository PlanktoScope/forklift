package core

// An AttachedRes is a binding between a resource and the source of that resource.
type AttachedRes[Res interface{}] struct {
	// Res is a resource subject to various possible constraints.
	Res Res
	// Source is a list of strings describing how to locate the resource in a package spec, to be
	// shown to users when a resource constraint is not met. Typically the elements of the list will
	// correspond to the parts of a path.
	Source []string
}

// Resource conflicts

// A ConflictChecker is something which may conflict with a resource, and which can determine
// whether it conflicts with another resource. Typically, resource types will implement this
// interface.
type ConflictChecker[Res interface{}] interface {
	CheckConflict(candidate Res) []error
}

// A ResConflict is a report of a conflict between two resources.
type ResConflict[Res interface{}] struct {
	// First is one of the two conflicting resources.
	First AttachedRes[Res]
	// Second is the other of the two conflicting resources.
	Second AttachedRes[Res]
	// Errs is a list of errors describing how the two resources conflict with each other.
	Errs []error
}

// Resource dependencies

// A DepChecker is something which may depend on a resource, and which can determine whether
// its dependency is satisfied by a resource. Typically, resource requirement types will implement
// this interface.
type DepChecker[Res interface{}] interface {
	CheckDep(candidate Res) []error
}

// A SatisfiedResDep is a report of a resource requirement which is satisfied by a set
// of resources.
type SatisfiedResDep[Res interface{}] struct {
	// Required is the resource requirement.
	Required AttachedRes[Res]
	// Provided is the resource which satisfies the resource requirement.
	Provided AttachedRes[Res]
}

// A MissingResDep is a report of a resource requirement which is not satisfied by any
// resources.
type MissingResDep[Res interface{}] struct {
	// Required is the resource requirement.
	Required AttachedRes[Res]
	// BestCandidates is a list of the resources which are closest to satisfying the resource
	// requirement.
	BestCandidates []ResDepCandidate[Res]
}

// ResDepCandidate is a report of a resource which either satisfied a resource
// requirement or (if Errs contains errors) failed to satisfy that resource requirement.
type ResDepCandidate[Res interface{}] struct {
	// Provided is the resource which did not satisfy the requirement.
	Provided AttachedRes[Res]
	// Errs is a list of errors describing how the resource did not satisfy the requirement.
	Errs []error
}
