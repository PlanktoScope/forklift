package core

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/pkg/structures"
)

// ProvidedRes

const (
	providesSourcePart = "provides resource"
	requiresSourcePart = "requires resource"
)

// AttachedListeners returns a list of [AttachedRes] instances for each respective host port
// listener in the ProvidedRes instance, adding a string to the provided list of source
// elements which describes the source of the ProvidedRes instance.
func (r ProvidedRes) AttachedListeners(source []string) []AttachedRes[ListenerRes] {
	return attachRes(r.Listeners, append(source, providesSourcePart))
}

// AttachedNetworks returns a list of [AttachedRes] instances for each respective Docker
// network in the ProvidedRes instance, adding a string to the provided list of source
// elements which describes the source of the ProvidedRes instance.
func (r ProvidedRes) AttachedNetworks(source []string) []AttachedRes[NetworkRes] {
	return attachRes(r.Networks, append(source, providesSourcePart))
}

// AttachedServices returns a list of [AttachedRes] instances for each respective network
// service in the ProvidedRes instance, adding a string to the provided list of source
// elements which describes the source of the ProvidedRes instance.
func (r ProvidedRes) AttachedServices(source []string) []AttachedRes[ServiceRes] {
	return attachRes(r.Services, append(source, providesSourcePart))
}

// AttachedFilesets returns a list of [AttachedRes] instances for each respective fileset
// in the ProvidedRes instance, adding a string to the provided list of source
// elements which describes the source of the ProvidedRes instance.
func (r ProvidedRes) AttachedFilesets(source []string) []AttachedRes[FilesetRes] {
	return attachRes(r.Filesets, append(source, providesSourcePart))
}

// AttachedFileExports returns a list of [AttachedRes] instances for each respective file export
// in the ProvidedRes instance, adding a string to the provided list of source
// elements which describes the source of the ProvidedRes instance.
func (r ProvidedRes) AttachedFileExports(source []string) []AttachedRes[FileExportRes] {
	return attachRes(r.FileExports, append(source, providesSourcePart))
}

// AddDefaults makes a copy with empty values replaced by default values.
func (r ProvidedRes) AddDefaults() ProvidedRes {
	updatedFileExports := make([]FileExportRes, 0, len(r.FileExports))
	for _, fileExport := range r.FileExports {
		updatedFileExports = append(updatedFileExports, fileExport.AddDefaults())
	}
	r.FileExports = updatedFileExports
	return r
}

// RequiredRes

// AttachedNetworks returns a list of [AttachedRes] instances for each respective Docker
// network resource requirement in the RequiredRes instance, adding a string to the provided
// list of source elements which describes the source of the RequiredRes instance.
func (r RequiredRes) AttachedNetworks(source []string) []AttachedRes[NetworkRes] {
	return attachRes(r.Networks, append(source, requiresSourcePart))
}

// AttachedServices returns a list of [AttachedRes] instances for each respective network
// service resource requirement in the RequiredRes instance, adding a string to the provided
// list of source elements which describes the source of the RequiredRes instance.
func (r RequiredRes) AttachedServices(source []string) []AttachedRes[ServiceRes] {
	return attachRes(r.Services, append(source, requiresSourcePart))
}

// AttachedFilesets returns a list of [AttachedRes] instances for each respective fileset
// resource requirement in the RequiredRes instance, adding a string to the provided
// list of source elements which describes the source of the RequiredRes instance.
func (r RequiredRes) AttachedFilesets(source []string) []AttachedRes[FilesetRes] {
	return attachRes(r.Filesets, append(source, requiresSourcePart))
}

// ListenerRes

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

// NetworkRes

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

// ServiceRes

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
		if strings.HasSuffix(path, "*") {
			prefix := strings.TrimSuffix(path, "*")
			prefixes.Add(prefix)
		}
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
			errs = append(errs, fmt.Errorf(errorMessage))
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

// SplitServicesByPath produces a slice of network service resources from the input slice, where
// each network service resource in the input slice with multiple paths results in multiple
// corresponding network service resources with one path each.
func SplitServicesByPath(serviceRes []AttachedRes[ServiceRes]) (split []AttachedRes[ServiceRes]) {
	split = make([]AttachedRes[ServiceRes], 0, len(serviceRes))
	for _, service := range serviceRes {
		if len(service.Res.Paths) == 0 {
			split = append(split, service)
		}
		for _, path := range service.Res.Paths {
			pathService := service.Res
			pathService.Paths = []string{path}
			split = append(split, AttachedRes[ServiceRes]{
				Res:    pathService,
				Source: service.Source,
			})
		}
	}
	return split
}

// FilesetRes

// CheckDep checks whether the fileset resource requirement, represented by the
// FilesetRes instance, is satisfied by the candidate fileset resource.
func (r FilesetRes) CheckDep(candidate FilesetRes) (errs []error) {
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

// CheckConflict checks whether the fileset resource, represented by the FilesetRes
// instance, conflicts with the candidate fileset resource.
func (r FilesetRes) CheckConflict(candidate FilesetRes) (errs []error) {
	if len(r.Paths) == 0 || len(candidate.Paths) == 0 {
		errs = append(errs, errors.New("no specified fileset paths"))
		return errs
	}

	errs = append(errs, checkConflictingPathsWithPrefixes(r.Paths, candidate.Paths)...)

	// Tags should be ignored in checking conflicts
	return errs
}

// SplitFilesetsByPath produces a slice of fileset resources from the input slice, where
// each fileset resource in the input slice with multiple paths results in multiple
// corresponding fileset resources with one path each.
func SplitFilesetsByPath(filesetRes []AttachedRes[FilesetRes]) (split []AttachedRes[FilesetRes]) {
	split = make([]AttachedRes[FilesetRes], 0, len(filesetRes))
	for _, fileset := range filesetRes {
		if len(fileset.Res.Paths) == 0 {
			split = append(split, fileset)
		}
		for _, path := range fileset.Res.Paths {
			pathFileset := fileset.Res
			pathFileset.Paths = []string{path}
			split = append(split, AttachedRes[FilesetRes]{
				Res:    pathFileset,
				Source: fileset.Source,
			})
		}
	}
	return split
}

// FileExportRes

// CheckConflict checks whether the file export resource, represented by the FileExportRes
// instance, conflicts with the candidate file export resource.
func (r FileExportRes) CheckConflict(candidate FileExportRes) (errs []error) {
	errs = append(errs, checkConflictingPathsWithParents(
		[]string{r.Target}, []string{candidate.Target})...,
	)

	// Tags should be ignored in checking conflicts
	return errs
}

// checkConflictingPathsWithParents checks every path in the list of provided paths against every
// path in the list of candidate paths to identify any conflicts between the two lists of paths,
// where a path which is a file matching a parent directory of the other path will conflict with
// that other path.
func checkConflictingPathsWithParents(provided, candidate []string) (errs []error) {
	pathConflicts := make(structures.Set[string])
	candidatePaths := make(structures.Set[string])
	for _, path := range candidate {
		candidatePaths.Add(path)
	}
	providedPaths := make(structures.Set[string])
	for _, path := range provided {
		providedPaths.Add(path)
	}

	for _, path := range provided {
		if candidatePaths.Has(path) {
			errorMessage := fmt.Sprintf("same path '%s'", path)
			if pathConflicts.Has(errorMessage) {
				continue
			}
			pathConflicts.Add(errorMessage)
			errs = append(errs, fmt.Errorf(errorMessage))
			continue
		}

		if match, parent := pathMatchesParent(path, candidatePaths); match {
			errorMessage := fmt.Sprintf("overlapping paths '%s' and '%s'", path, parent)
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
		if match, parent := pathMatchesParent(candidatePath, providedPaths); match {
			errorMessage := fmt.Sprintf("overlapping paths '%s' and '%s'", parent, candidatePath)
			if pathConflicts.Has(errorMessage) {
				continue
			}
			pathConflicts.Add(errorMessage)
			errs = append(errs, errors.New(errorMessage))
		}
	}

	return errs
}

// pathMatchesParent checks whether the provided path is in a subdirectory of any of the parent
// paths (where all parent paths are interpreted as if they were directories).
func pathMatchesParent(
	path string, parentPaths structures.Set[string],
) (match bool, parent string) {
	for parent := range parentPaths {
		if strings.HasPrefix(path, strings.TrimSuffix(parent, "/")+"/") {
			return true, parent
		}
	}
	return false, ""
}

// FileExportRes

// AddDefaults makes a copy with empty values replaced by default values according to the file
// export source type.
func (r FileExportRes) AddDefaults() FileExportRes {
	if r.SourceType == "" {
		r.SourceType = FileExportSourceTypeLocal
	}
	switch r.SourceType {
	case FileExportSourceTypeLocal:
		if r.Source == "" {
			r.Source = r.Target
		}
	case FileExportSourceTypeHTTPArchive:
		if r.Source == "" {
			r.Source = r.Target
		}
	case FileExportSourceTypeOCIImage:
		if r.Source == "" {
			r.Source = r.Target
		}
	}
	return r
}
