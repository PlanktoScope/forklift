package resources

// An Attached is a binding between a resource and the origin of that resource.
type Attached[Res any, Origin any] struct {
	// Res is a resource subject to various possible constraints.
	Res Res
	// Origin is describes how to locate the resource in a package spec, e.g. so that it can be shown
	// to users when a resource constraint is not met or when a resource conflict exists.
	Origin Origin
}

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
