package forklift

type Describer interface {
	Describe() string
}

type AttachedResource[Resource interface{}] struct {
	Resource Resource
	Source   []string
}

// Dependency constraints

type DependencyChecker[Resource interface{}] interface {
	CheckDependency(candidate Resource) []error
}

type MissingResourceDependency[Resource interface{}] struct {
	Resource       AttachedResource[Resource]
	BestCandidates []AttachedResource[Resource]
	Errs           []error
}

type MissingDeplDependencies struct {
	Depl Depl

	Listeners []MissingResourceDependency[ListenerResource]
	Networks  []MissingResourceDependency[NetworkResource]
	Services  []MissingResourceDependency[ServiceResource]
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
