package forklift

type AttachedResource[Resource interface{}] struct {
	Resource Resource
	Source   []string
}

// Conflict constraints

type ConflictChecker[Resource interface{}] interface {
	CheckConflict(candidate Resource) []error
}

type ResourceConflict[Resource interface{}] struct {
	First  AttachedResource[Resource]
	Second AttachedResource[Resource]
	Errs   []error
}

type DeplConflict struct {
	First  Depl
	Second Depl

	// Possible conflicts
	Name      bool
	Listeners []ResourceConflict[ListenerResource]
	Networks  []ResourceConflict[NetworkResource]
	Services  []ResourceConflict[ServiceResource]
}

// Dependency constraints

type DependencyChecker[Resource interface{}] interface {
	CheckDependency(candidate Resource) []error
}

type ResourceDependencyCandidate[Resource interface{}] struct {
	Provided AttachedResource[Resource]
	Errs     []error
}

type MissingResourceDependency[Resource interface{}] struct {
	Required       AttachedResource[Resource]
	BestCandidates []ResourceDependencyCandidate[Resource]
}

type MissingDeplDependencies struct {
	Depl Depl

	Networks []MissingResourceDependency[NetworkResource]
	Services []MissingResourceDependency[ServiceResource]
}
