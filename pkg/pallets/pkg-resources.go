package pallets

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// ProvidedResources

const (
	providesSourcePart = "provides resource"
	requiresSourcePart = "requires resource"
)

// AttachedListeners returns a list of [AttachedResource] instances for each respective host port
// listener in the ProvidedResources instance, adding a string to the provided list of source
// elements which describes the source of the ProvidedResources instance.
func (r ProvidedResources) AttachedListeners(source []string) []AttachedResource[ListenerResource] {
	return attachResources(r.Listeners, append(source, providesSourcePart))
}

// AttachedNetworks returns a list of [AttachedResource] instances for each respective Docker
// network in the ProvidedResources instance, adding a string to the provided list of source
// elements which describes the source of the ProvidedResources instance.
func (r ProvidedResources) AttachedNetworks(source []string) []AttachedResource[NetworkResource] {
	return attachResources(r.Networks, append(source, providesSourcePart))
}

// AttachedServices returns a list of [AttachedResource] instances for each respective network
// service in the ProvidedResources instance, adding a string to the provided list of source
// elements which describes the source of the ProvidedResources instance.
func (r ProvidedResources) AttachedServices(source []string) []AttachedResource[ServiceResource] {
	return attachResources(r.Services, append(source, providesSourcePart))
}

// RequiredResources

// AttachedNetworks returns a list of [AttachedResource] instances for each respective Docker
// network resource requirement in the RequiredResources instance, adding a string to the provided
// list of source elements which describes the source of the RequiredResources instance.
func (r RequiredResources) AttachedNetworks(source []string) []AttachedResource[NetworkResource] {
	return attachResources(r.Networks, append(source, requiresSourcePart))
}

// AttachedServices returns a list of [AttachedResource] instances for each respective network
// service resource requirement in the RequiredResources instance, adding a string to the provided
// list of source elements which describes the source of the RequiredResources instance.
func (r RequiredResources) AttachedServices(source []string) []AttachedResource[ServiceResource] {
	return attachResources(r.Services, append(source, requiresSourcePart))
}

// ListenerResource

// CheckDependency checks whether the host port listener resource requirement, represented by the
// ListenerResource instance, is satisfied by the candidate host port listener resource.
func (r ListenerResource) CheckDependency(candidate ListenerResource) (errs []error) {
	if r.Port != candidate.Port {
		errs = append(errs, fmt.Errorf("unmatched port '%d'", r.Port))
	}
	if r.Protocol != candidate.Protocol {
		errs = append(errs, fmt.Errorf("unmatched protocol '%s'", r.Protocol))
	}
	return errs
}

// CheckConflict checks whether the host port listener resource, represented by the ListenerResource
// instance, conflicts with the candidate host port listener resource.
func (r ListenerResource) CheckConflict(candidate ListenerResource) (errs []error) {
	if r.Port == candidate.Port && r.Protocol == candidate.Protocol {
		errs = append(errs, fmt.Errorf("same port/protocol '%d/%s'", r.Port, r.Protocol))
	}
	return errs
}

// NetworkResource

// CheckDependency checks whether the Docker network resource requirement, represented by the
// NetworkResource instance, is satisfied by the candidate Docker network resource.
func (r NetworkResource) CheckDependency(candidate NetworkResource) (errs []error) {
	if r.Name != candidate.Name {
		errs = append(errs, fmt.Errorf("unmatched name '%s'", r.Name))
	}
	return errs
}

// CheckConflict checks whether the Docker network resource, represented by the NetworkResource
// instance, conflicts with the candidate Docker network resource.
func (r NetworkResource) CheckConflict(candidate NetworkResource) (errs []error) {
	if r.Name == candidate.Name {
		errs = append(errs, fmt.Errorf("same name '%s'", r.Name))
	}
	return errs
}

// ServiceResource

// CheckDependency checks whether the network service resource requirement, represented by the
// ServiceResource instance, is satisfied by the candidate network service resource.
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

// parseServicePaths splits the provided list of paths into a set of exact paths and a set of prefix
// paths, with the trailing asterisk (`*`) removed from the prefixes.
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

// pathMatchesExactly checks whether the provided path matches the provided set of exact paths.
func pathMatchesExactly(path string, exactPaths map[string]struct{}) bool {
	_, ok := exactPaths[path]
	return ok
}

// pathMatchesPrefix checks whether the provided path matches the provided set of prefix paths.
func pathMatchesPrefix(path string, pathPrefixes map[string]struct{}) (match bool, prefix string) {
	for prefix := range pathPrefixes {
		if strings.HasPrefix(strings.TrimSuffix(path, "*"), prefix) {
			return true, prefix
		}
	}
	return false, ""
}

// CheckConflict checks whether the network service resource, represented by the ServiceResource
// instance, conflicts with the candidate network service resource.
func (r ServiceResource) CheckConflict(candidate ServiceResource) (errs []error) {
	if r.Port != candidate.Port || r.Protocol != candidate.Protocol {
		return nil
	}

	if len(r.Paths) == 0 && len(candidate.Paths) == 0 {
		errs = append(errs, fmt.Errorf(
			"same port/protocol '%d/%s' without specified service paths", r.Port, r.Protocol),
		)
		return errs
	}

	errs = append(errs, checkConflictingPaths(r.Paths, candidate.Paths)...)

	// Tags should be ignored in checking conflicts
	return errs
}

// checkConflictingPaths checks every path in the list of provided paths against every path in the
// list of candidate paths to identify any conflicts between the two lists of paths.
func checkConflictingPaths(provided, candidate []string) (errs []error) {
	pathConflicts := make(map[string]struct{})
	candidatePaths, candidatePathPrefixes := parseServicePaths(candidate)
	providedPaths, providedPathPrefixes := parseServicePaths(provided)

	for _, path := range provided {
		if pathMatchesExactly(path, candidatePaths) {
			errorMessage := fmt.Sprintf("same path '%s'", path)
			if _, ok := pathConflicts[errorMessage]; ok {
				continue
			}
			pathConflicts[errorMessage] = struct{}{}
			errs = append(errs, fmt.Errorf(errorMessage))
			continue
		}

		if match, prefix := pathMatchesPrefix(path, candidatePathPrefixes); match {
			errorMessage := fmt.Sprintf(
				"overlapping paths '%s' and '%s*'", path, prefix,
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
				"overlapping paths '%s*' and '%s'", prefix, candidatePath,
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

// SplitServicesByPath produces a slice of network service resources from the input slice, where
// each network service resource in the input slice with multiple paths results in multiple
// corresponding network service resources with one path each.
func SplitServicesByPath(
	serviceResources []AttachedResource[ServiceResource],
) (split []AttachedResource[ServiceResource]) {
	split = make([]AttachedResource[ServiceResource], 0, len(serviceResources))
	for _, service := range serviceResources {
		if len(service.Resource.Paths) == 0 {
			split = append(split, service)
		}
		for _, path := range service.Resource.Paths {
			pathService := service.Resource
			pathService.Paths = []string{path}
			split = append(split, AttachedResource[ServiceResource]{
				Resource: pathService,
				Source:   service.Source,
			})
		}
	}
	return split
}
