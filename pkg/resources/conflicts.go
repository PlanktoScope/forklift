package resources

// CheckConflicts identifies all resource conflicts between the first list of resources and
// the second list of resources. It does not identify resource conflicts within the first list of
// resources, nor within the second list of resources.
func CheckConflicts[Res ConflictChecker[Res], Origin any](
	first []Attached[Res, Origin], second []Attached[Res, Origin],
) (conflicts []Conflict[Res, Origin]) {
	for _, f := range first {
		for _, s := range second {
			if errs := f.Res.CheckConflict(s.Res); len(errs) > 0 {
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
