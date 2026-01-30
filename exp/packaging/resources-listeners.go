package packaging

import (
	"fmt"
)

// ListenerRes describes a host port listener.
type ListenerRes struct {
	// Description is a short description of the host port listener to be shown to users.
	Description string `yaml:"description,omitempty"`
	// Port is the port number which the host port listener is bound to.
	Port int `yaml:"port,omitempty"`
	// Protocol is the transport protocol (either tcp or udp) which the host port listener is bound
	// to.
	Protocol string `yaml:"protocol,omitempty"`
}

// CheckDep checks whether the host port listener resource requirement, represented by the
// ListenerRes instance, is satisfied by the candidate host port listener resource.
func (r ListenerRes) CheckDep(candidate ListenerRes) (errs []error) {
	if r.Port != candidate.Port {
		errs = append(errs, fmt.Errorf("unmatched port '%d'", r.Port))
	}
	if r.Protocol != candidate.Protocol {
		errs = append(errs, fmt.Errorf("unmatched protocol '%s'", r.Protocol))
	}
	return errs
}

// CheckConflict checks whether the host port listener resource, represented by the ListenerRes
// instance, conflicts with the candidate host port listener resource.
func (r ListenerRes) CheckConflict(candidate ListenerRes) (errs []error) {
	if r.Port == candidate.Port && r.Protocol == candidate.Protocol {
		errs = append(errs, fmt.Errorf("same port/protocol '%d/%s'", r.Port, r.Protocol))
	}
	return errs
}
