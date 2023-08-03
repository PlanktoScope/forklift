package pallets

// An AttachedResource is a binding between a resource and the source of that resource.
type AttachedResource[Resource interface{}] struct {
	// Resource is a resource subject to various possible constraints.
	Resource Resource
	// Source is a list of strings describing how to locate the resource in a package spec, to be
	// shown to users when a resource constraint is not met. Typically the elements of the list will
	// correspond to the parts of a path.
	Source []string
}

// Resource conflicts

// A ConflictChecker is something which may conflict with a resource, and which can determine
// whether it conflicts with another resource. Typically, resource types will implement this
// interface.
type ConflictChecker[Resource interface{}] interface {
	CheckConflict(candidate Resource) []error
}

// A ResourceConflict is a report of a conflict between two resources.
type ResourceConflict[Resource interface{}] struct {
	// First is one of the two conflicting resources.
	First AttachedResource[Resource]
	// Second is the other of the two conflicting resources.
	Second AttachedResource[Resource]
	// Errs is a list of errors describing how the two resources conflict with each other.
	Errs []error
}

// Resource dependencies

// A DependencyChecker is something which may depend on a resource, and which can determine whether
// its dependency is satisfied by a resource. Typically, resource requirement types will implement
// this interface.
type DependencyChecker[Resource interface{}] interface {
	CheckDependency(candidate Resource) []error
}

// A SatisfiedResourceDependency is a report of a resource requirement which is satisfied by a set
// of resources.
type SatisfiedResourceDependency[Resource interface{}] struct {
	// Required is the resource requirement.
	Required AttachedResource[Resource]
	// Provided is the resource which satisfies the resource requirement.
	Provided AttachedResource[Resource]
}

// A MissingResourceDependency is a report of a resource requirement which is not satisfied by any
// resources.
type MissingResourceDependency[Resource interface{}] struct {
	// Required is the resource requirement.
	Required AttachedResource[Resource]
	// BestCandidates is a list of the resources which are closest to satisfying the resource
	// requirement.
	BestCandidates []ResourceDependencyCandidate[Resource]
}

// ResourceDependencyCandidate is a report of a resource which either satisfied a resource
// requirement or (if Errs contains errors) failed to satisfy that resource requirement.
type ResourceDependencyCandidate[Resource interface{}] struct {
	// Provided is the resource which did not satisfy the requirement.
	Provided AttachedResource[Resource]
	// Errs is a list of errors describing how the resource did not satisfy the requirement.
	Errs []error
}
