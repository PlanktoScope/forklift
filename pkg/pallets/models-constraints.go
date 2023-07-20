package pallets

type AttachedResource[Resource interface{}] struct {
	Resource Resource
	Source   []string
}

// Resource conflicts

type ConflictChecker[Resource interface{}] interface {
	CheckConflict(candidate Resource) []error
}

type ResourceConflict[Resource interface{}] struct {
	First  AttachedResource[Resource]
	Second AttachedResource[Resource]
	Errs   []error
}

// Resource dependencies

type DependencyChecker[Resource interface{}] interface {
	CheckDependency(candidate Resource) []error
}

type MissingResourceDependency[Resource interface{}] struct {
	Required       AttachedResource[Resource]
	BestCandidates []ResourceDependencyCandidate[Resource]
}

type ResourceDependencyCandidate[Resource interface{}] struct {
	Provided AttachedResource[Resource]
	Errs     []error
}
