package forklift

import (
	"errors"
	"fmt"
	"strings"
)

func NewAttachedResource[Resource Describer](
	resource Resource, source []string,
) AttachedResource[Resource] {
	return AttachedResource[Resource]{
		Resource: resource,
		Source:   append(source, resource.Describe()),
	}
}

func attachResources[Resource Describer](
	resources []Resource, source []string,
) (attached []AttachedResource[Resource]) {
	attached = make([]AttachedResource[Resource], 0, len(resources))
	for _, resource := range resources {
		attached = append(attached, NewAttachedResource(resource, source))
	}
	return attached
}

// ProvidedResources

func (r ProvidedResources) AttachedListeners(source []string) []AttachedResource[ListenerResource] {
	return attachResources(r.Listeners, append(source, "provides resource"))
}

func (r ProvidedResources) AttachedNetworks(source []string) []AttachedResource[NetworkResource] {
	return attachResources(r.Networks, append(source, "provides resource"))
}

func (r ProvidedResources) AttachedServices(source []string) []AttachedResource[ServiceResource] {
	return attachResources(r.Services, append(source, "provides resource"))
}

// RequiredResources

func (r RequiredResources) AttachedNetworks(source []string) []AttachedResource[NetworkResource] {
	return attachResources(r.Networks, append(source, "requires resource"))
}

func (r RequiredResources) AttachedServices(source []string) []AttachedResource[ServiceResource] {
	return attachResources(r.Services, append(source, "requires resource"))
}

// PkgHostSpec

func (s PkgHostSpec) attachmentSource(parentSource []string) []string {
	return append(parentSource, "host")
}

// PkgDeplSpec

func (s PkgDeplSpec) attachmentSource(parentSource []string) []string {
	return append(parentSource, "deployment")
}

// PkgFeatureSpec

func (s PkgFeatureSpec) attachmentSource(parentSource []string, featureName string) []string {
	return append(parentSource, fmt.Sprintf("feature %s", featureName))
}

// Listener

func (r ListenerResource) Describe() string {
	return r.Description
}

func (r ListenerResource) CheckDependency(candidate ListenerResource) (errs []error) {
	if r.Port != candidate.Port {
		errs = append(errs, fmt.Errorf("unmatched port '%d'", r.Port))
	}
	if r.Protocol != candidate.Protocol {
		errs = append(errs, fmt.Errorf("unmatched protocol '%s'", r.Protocol))
	}
	return errs
}

func (r ListenerResource) CheckConflict(candidate ListenerResource) (errs []error) {
	if r.Port == candidate.Port && r.Protocol == candidate.Protocol {
		errs = append(errs, fmt.Errorf("conflicting port/protocol '%d/%s'", r.Port, r.Protocol))
	}
	return errs
}

// Network

func (r NetworkResource) Describe() string {
	return r.Description
}

func (r NetworkResource) CheckDependency(candidate NetworkResource) (errs []error) {
	if r.Name != candidate.Name {
		errs = append(errs, fmt.Errorf("unmatched name '%s'", r.Name))
	}
	return errs
}

func (r NetworkResource) CheckConflict(candidate NetworkResource) (errs []error) {
	if r.Name == candidate.Name {
		errs = append(errs, fmt.Errorf("conflicting name '%s'", r.Name))
	}
	return errs
}

// Service

func (r ServiceResource) Describe() string {
	return r.Description
}

func (r ServiceResource) CheckDependency(candidate ServiceResource) (errs []error) {
	if r.Port != candidate.Port {
		errs = append(errs, fmt.Errorf("unmatched port '%d'", r.Port))
	}
	if r.Protocol != candidate.Protocol {
		errs = append(errs, fmt.Errorf("unmatched protocol '%s'", r.Protocol))
	}

	// TODO: precompute candidatePaths and candidatePathPrefixes, if this is a performance bottleneck
	candidatePaths, candidatePathPrefixes := parseServicePaths(candidate.Paths)
	for _, path := range r.Paths {
		if pathMatchesExactly(path, candidatePaths) {
			continue
		}
		if match, _ := pathMatchesPrefix(path, candidatePathPrefixes); match {
			continue
		}
		errs = append(errs, fmt.Errorf("unmatched path '%s'", path))
	}

	candidateTags := make(map[string]struct{})
	for _, tag := range candidate.Tags {
		candidateTags[tag] = struct{}{}
	}
	for _, tag := range r.Tags {
		if _, ok := candidateTags[tag]; ok {
			continue
		}
		errs = append(errs, fmt.Errorf("unmatched tag '%s'", tag))
	}

	return errs
}

func parseServicePaths(paths []string) (exact, prefixes map[string]struct{}) {
	exact = make(map[string]struct{})
	prefixes = make(map[string]struct{})
	for _, path := range paths {
		exact[path] = struct{}{} // even prefix paths are stored here, for fast exact matching
		if strings.HasSuffix(path, "*") {
			prefix := strings.TrimSuffix(path, "*")
			prefixes[prefix] = struct{}{}
		}
	}
	return exact, prefixes
}

func pathMatchesExactly(path string, exactPaths map[string]struct{}) bool {
	_, ok := exactPaths[path]
	return ok
}

func pathMatchesPrefix(path string, pathPrefixes map[string]struct{}) (match bool, prefix string) {
	for prefix := range pathPrefixes {
		if strings.HasPrefix(strings.TrimSuffix(path, "*"), prefix) {
			return true, prefix
		}
	}
	return false, ""
}

func (r ServiceResource) CheckConflict(candidate ServiceResource) (errs []error) {
	if r.Port != candidate.Port || r.Protocol != candidate.Protocol {
		return nil
	}

	if len(r.Paths) == 0 && len(candidate.Paths) == 0 {
		errs = append(errs, fmt.Errorf("conflicting port/protocol '%d/%s'", r.Port, r.Protocol))
		return errs
	}

	errs = append(errs, checkConflictingPaths(r.Paths, candidate.Paths)...)

	// Tags should be ignored in checking conflicts
	return errs
}

func checkConflictingPaths(provided, candidate []string) (errs []error) {
	pathConflicts := make(map[string]struct{})
	candidatePaths, candidatePathPrefixes := parseServicePaths(candidate)
	providedPaths, providedPathPrefixes := parseServicePaths(provided)

	for _, path := range provided {
		if pathMatchesExactly(path, candidatePaths) {
			errorMessage := fmt.Sprintf("conflicting path '%s'", path)
			if _, ok := pathConflicts[errorMessage]; ok {
				continue
			}
			pathConflicts[errorMessage] = struct{}{}
			errs = append(errs, fmt.Errorf(errorMessage))
			continue
		}

		if match, prefix := pathMatchesPrefix(path, candidatePathPrefixes); match {
			errorMessage := fmt.Sprintf(
				"conflicting paths '%s' and '%s*'", path, prefix,
			)
			if _, ok := pathConflicts[errorMessage]; ok {
				continue
			}
			pathConflicts[errorMessage] = struct{}{}
			errs = append(errs, errors.New(errorMessage))
		}
	}
	for _, candidatePath := range candidate {
		if pathMatchesExactly(candidatePath, providedPaths) {
			// Exact matches were already handled in the previous for loop
			continue
		}
		if match, prefix := pathMatchesPrefix(candidatePath, providedPathPrefixes); match {
			errorMessage := fmt.Sprintf(
				"conflicting paths '%s*' and '%s'", prefix, candidatePath,
			)
			if _, ok := pathConflicts[errorMessage]; ok {
				continue
			}
			pathConflicts[errorMessage] = struct{}{}
			errs = append(errs, errors.New(errorMessage))
		}
	}

	return errs
}
