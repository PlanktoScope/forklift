package packaging

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	res "github.com/forklift-run/forklift/pkg/resources"
	"github.com/forklift-run/forklift/pkg/structures"
)

// ServiceRes describes a network service.
type ServiceRes struct {
	// Description is a short description of the network service to be shown to users.
	Description string `yaml:"description,omitempty"`
	// Port is the network port used for accessing the service.
	Port int `yaml:"port,omitempty"`
	// Protocol is the application-level protocol (e.g. http or mqtt) used for accessing the service.
	Protocol string `yaml:"protocol,omitempty"`
	// Tags is a list of strings associated with the service. Tags are considered in determining which
	// service resources meet service resource requirements.
	Tags []string `yaml:"tags,omitempty"`
	// Paths is a list of paths used for accessing the service. A path may also be a prefix, indicated
	// by ending the path with an asterisk (`*`).
	Paths []string `yaml:"paths,omitempty"`
	// Nonblocking, when specified as a resource requirement, specifies that the client of the service
	// does not need to wait for the resource to exist before the client can start.
	Nonblocking bool `yaml:"nonblocking,omitempty"`
}

// CheckDep checks whether the network service resource requirement, represented by the
// ServiceRes instance, is satisfied by the candidate network service resource.
func (r ServiceRes) CheckDep(candidate ServiceRes) (errs []error) {
	if r.Port != candidate.Port {
		errs = append(errs, fmt.Errorf("unmatched port '%d'", r.Port))
	}
	if r.Protocol != candidate.Protocol {
		errs = append(errs, fmt.Errorf("unmatched protocol '%s'", r.Protocol))
	}

	// TODO: precompute candidatePaths and candidatePathPrefixes, if this is a performance bottleneck
	candidatePaths, candidatePathPrefixes := parsePathsWithPrefixes(candidate.Paths)
	for _, path := range r.Paths {
		if candidatePaths.Has(path) {
			continue
		}
		if match, _ := pathMatchesPrefix(path, candidatePathPrefixes); match {
			continue
		}
		errs = append(errs, fmt.Errorf("unmatched path '%s'", path))
	}

	candidateTags := make(structures.Set[string])
	for _, tag := range candidate.Tags {
		candidateTags.Add(tag)
	}
	for _, tag := range r.Tags {
		if candidateTags.Has(tag) {
			continue
		}
		errs = append(errs, fmt.Errorf("unmatched tag '%s'", tag))
	}

	return errs
}

// parsePathsWithPrefixes splits the provided list of paths into a set of exact paths and a set of
// prefix paths, with the trailing asterisk (`*`) removed from the prefixes.
func parsePathsWithPrefixes(paths []string) (exact, prefixes structures.Set[string]) {
	exact = make(structures.Set[string])
	prefixes = make(structures.Set[string])
	for _, path := range paths {
		exact.Add(path) // even prefix paths are stored here, for fast exact matching
		if !strings.HasSuffix(path, "*") {
			continue
		}
		prefixes.Add(strings.TrimSuffix(path, "*"))
	}
	return exact, prefixes
}

// pathMatchesPrefix checks whether the provided path matches the provided set of prefix paths.
func pathMatchesPrefix(
	path string, pathPrefixes structures.Set[string],
) (match bool, prefix string) {
	for prefix := range pathPrefixes {
		if strings.HasPrefix(strings.TrimSuffix(path, "*"), prefix) {
			return true, prefix
		}
	}
	return false, ""
}

// CheckConflict checks whether the network service resource, represented by the ServiceRes
// instance, conflicts with the candidate network service resource.
func (r ServiceRes) CheckConflict(candidate ServiceRes) (errs []error) {
	if r.Port != candidate.Port || r.Protocol != candidate.Protocol {
		return nil
	}

	if len(r.Paths) == 0 && len(candidate.Paths) == 0 {
		errs = append(errs, fmt.Errorf(
			"same port/protocol '%d/%s' without specified service paths", r.Port, r.Protocol),
		)
		return errs
	}

	errs = append(errs, checkConflictingPathsWithPrefixes(r.Paths, candidate.Paths)...)

	// Tags should be ignored in checking conflicts
	return errs
}

// checkConflictingPathsWithPrefixes checks every path in the list of provided paths against every
// path in the list of candidate paths to identify any conflicts between the two lists of paths,
// where paths can be specified as prefixes using `*` as a suffix.
func checkConflictingPathsWithPrefixes(provided, candidate []string) (errs []error) {
	pathConflicts := make(structures.Set[string])
	candidatePaths, candidatePathPrefixes := parsePathsWithPrefixes(candidate)
	providedPaths, providedPathPrefixes := parsePathsWithPrefixes(provided)

	for _, path := range provided {
		if candidatePaths.Has(path) {
			errorMessage := fmt.Sprintf("same path '%s'", path)
			if pathConflicts.Has(errorMessage) {
				continue
			}
			pathConflicts.Add(errorMessage)
			errs = append(errs, errors.New(errorMessage))
			continue
		}

		if match, prefix := pathMatchesPrefix(path, candidatePathPrefixes); match {
			errorMessage := fmt.Sprintf(
				"overlapping paths '%s' and '%s*'", path, prefix,
			)
			if pathConflicts.Has(errorMessage) {
				continue
			}
			pathConflicts.Add(errorMessage)
			errs = append(errs, errors.New(errorMessage))
		}
	}
	for _, candidatePath := range candidate {
		if providedPaths.Has(candidatePath) {
			// Exact matches were already handled in the previous for loop
			continue
		}
		if match, prefix := pathMatchesPrefix(candidatePath, providedPathPrefixes); match {
			errorMessage := fmt.Sprintf(
				"overlapping paths '%s*' and '%s'", prefix, candidatePath,
			)
			if pathConflicts.Has(errorMessage) {
				continue
			}
			pathConflicts.Add(errorMessage)
			errs = append(errs, errors.New(errorMessage))
		}
	}

	return errs
}

// SplitServicesByPath produces a slice of network service res from the input slice, where
// each network service resource in the input slice with multiple paths results in multiple
// corresponding network service res with one path each.
func SplitServicesByPath(
	serviceRes []res.Attached[ServiceRes, []string],
) (split []res.Attached[ServiceRes, []string]) {
	split = make([]res.Attached[ServiceRes, []string], 0, len(serviceRes))
	for _, service := range serviceRes {
		if len(service.Res.Paths) == 0 {
			split = append(split, service)
		}
		for _, path := range service.Res.Paths {
			pathService := service.Res
			pathService.Paths = []string{path}
			split = append(split, res.Attached[ServiceRes, []string]{
				Res:    pathService,
				Origin: service.Origin,
			})
		}
	}
	return split
}
