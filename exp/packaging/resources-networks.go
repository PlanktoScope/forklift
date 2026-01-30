package packaging

import (
	"fmt"
)

// NetworkRes describes a Docker network.
type NetworkRes struct {
	// Description is a short description of the Docker network to be shown to users.
	Description string `yaml:"description,omitempty"`
	// Name is the name of the Docker network.
	Name string `yaml:"name"`
}

// CheckDep checks whether the Docker network resource requirement, represented by the
// NetworkRes instance, is satisfied by the candidate Docker network resource.
func (r NetworkRes) CheckDep(candidate NetworkRes) (errs []error) {
	if r.Name != candidate.Name {
		errs = append(errs, fmt.Errorf("unmatched name '%s'", r.Name))
	}
	return errs
}

// CheckConflict checks whether the Docker network resource, represented by the NetworkRes
// instance, conflicts with the candidate Docker network resource.
func (r NetworkRes) CheckConflict(candidate NetworkRes) (errs []error) {
	if r.Name == candidate.Name {
		errs = append(errs, fmt.Errorf("same name '%s'", r.Name))
	}
	return errs
}
