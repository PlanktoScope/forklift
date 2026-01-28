package packaging

import (
	"fmt"

	"github.com/pkg/errors"

	res "github.com/forklift-run/forklift/pkg/resources"
	"github.com/forklift-run/forklift/pkg/structures"
)

// FilesetRes describes a set of files/directories.
type FilesetRes struct {
	// Description is a short description of the fileset to be shown to users.
	Description string `yaml:"description,omitempty"`
	// Tags is a list of strings associated with the fileset. Tags are considered in determining which
	// fileset resources meet fileset resource requirements.
	Tags []string `yaml:"tags,omitempty"`
	// Paths is a list of paths where the fileset exists. A path may also be a prefix, indicated
	// by ending the path with an asterisk (`*`).
	Paths []string `yaml:"paths"`
	// Nonblocking, when specified as a resource requirement, specifies that the program requiring the
	// fileset does not need to wait for the fileset to exist before the program can start.
	Nonblocking bool `yaml:"nonblocking,omitempty"`
}

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

// SplitFilesetsByPath produces a slice of fileset res from the input slice, where
// each fileset resource in the input slice with multiple paths results in multiple
// corresponding fileset res with one path each.
func SplitFilesetsByPath(
	filesetRes []res.Attached[FilesetRes, []string],
) (split []res.Attached[FilesetRes, []string]) {
	split = make([]res.Attached[FilesetRes, []string], 0, len(filesetRes))
	for _, fileset := range filesetRes {
		if len(fileset.Res.Paths) == 0 {
			split = append(split, fileset)
		}
		for _, path := range fileset.Res.Paths {
			pathFileset := fileset.Res
			pathFileset.Paths = []string{path}
			split = append(split, res.Attached[FilesetRes, []string]{
				Res:    pathFileset,
				Origin: fileset.Origin,
			})
		}
	}
	return split
}
