package packaging

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/pkg/structures"
)

// FileExportRes describes a file exported by Forklift.
type FileExportRes struct {
	// Description is a short description of the file export to be shown to users.
	Description string `yaml:"description,omitempty"`
	// Tags is a list of strings associated with the file export. Tags are not considered in checking
	// resource constraints.
	Tags []string `yaml:"tags,omitempty"`
	// SourceType is either `local` (for a file in the package whose path is set by `Source`) or
	// `http` (for a file which needs to be downloaded from the URL set by `URL`).
	SourceType string `yaml:"source-type,omitempty"`
	// Source is the path in the package of the file to be exported, for a `local` source. If omitted,
	// the source path will be inferred from the Target path.
	Source string `yaml:"source,omitempty"`
	// URL is the URL of the file to be downloaded for export, for a `http` source.
	URL string `yaml:"url,omitempty"`
	// Permissions is the Unix permission bits to attach to the exported file.
	Permissions fs.FileMode `yaml:"permissions,omitempty"`
	// Target is the path where the file will be exported to, relative to an export directory.
	Target string `yaml:"target"`
}

const (
	FileExportSourceTypeLocal       = "local"
	FileExportSourceTypeHTTP        = "http"
	FileExportSourceTypeHTTPArchive = "http-archive"
	FileExportSourceTypeOCIImage    = "oci-image"
)

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
			errs = append(errs, errors.New(errorMessage))
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
