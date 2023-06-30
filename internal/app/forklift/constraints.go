package forklift

// Conflicts

func CheckResourcesConflicts[Resource ConflictChecker[Resource]](
	first []AttachedResource[Resource], second []AttachedResource[Resource],
) (conflicts []ResourceConflict[Resource]) {
	for _, f := range first {
		for _, s := range second {
			if errs := f.Resource.CheckConflict(s.Resource); errs != nil {
				conflicts = append(conflicts, ResourceConflict[Resource]{
					First:  f,
					Second: s,
					Errs:   errs,
				})
			}
		}
	}
	return conflicts
}

func (c DeplConflict) HasNameConflict() bool {
	return c.Name
}

func (c DeplConflict) HasListenerConflict() bool {
	return c.Listeners != nil && len(c.Listeners) > 0
}

func (c DeplConflict) HasNetworkConflict() bool {
	return c.Networks != nil && len(c.Networks) > 0
}

func (c DeplConflict) HasServiceConflict() bool {
	return c.Services != nil && len(c.Services) > 0
}

func (c DeplConflict) HasConflict() bool {
	return c.HasNameConflict() ||
		c.HasListenerConflict() || c.HasNetworkConflict() || c.HasServiceConflict()
}

// Dependencies
