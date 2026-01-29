package forklift

import (
	fpkg "github.com/forklift-run/forklift/pkg/packaging"
	res "github.com/forklift-run/forklift/pkg/resources"
)

type DeplConflict struct {
	First  *ResolvedDepl
	Second *ResolvedDepl

	// Possible conflicts
	Name        bool
	Listeners   []res.Conflict[fpkg.ListenerRes, []string]
	Networks    []res.Conflict[fpkg.NetworkRes, []string]
	Services    []res.Conflict[fpkg.ServiceRes, []string]
	Filesets    []res.Conflict[fpkg.FilesetRes, []string]
	FileExports []res.Conflict[fpkg.FileExportRes, []string]
}

type SatisfiedDeplDeps struct {
	Depl *ResolvedDepl

	Networks []res.SatisfiedDep[fpkg.NetworkRes, []string]
	Services []res.SatisfiedDep[fpkg.ServiceRes, []string]
	Filesets []res.SatisfiedDep[fpkg.FilesetRes, []string]
}

type MissingDeplDeps struct {
	Depl *ResolvedDepl

	Networks []res.MissingDep[fpkg.NetworkRes, []string]
	Services []res.MissingDep[fpkg.ServiceRes, []string]
	Filesets []res.MissingDep[fpkg.FilesetRes, []string]
}
