package forklift

import (
	"github.com/forklift-run/forklift/pkg/core"
	res "github.com/forklift-run/forklift/pkg/resources"
)

type DeplConflict struct {
	First  *ResolvedDepl
	Second *ResolvedDepl

	// Possible conflicts
	Name        bool
	Listeners   []res.Conflict[core.ListenerRes, []string]
	Networks    []res.Conflict[core.NetworkRes, []string]
	Services    []res.Conflict[core.ServiceRes, []string]
	Filesets    []res.Conflict[core.FilesetRes, []string]
	FileExports []res.Conflict[core.FileExportRes, []string]
}

type SatisfiedDeplDeps struct {
	Depl *ResolvedDepl

	Networks []res.SatisfiedDep[core.NetworkRes, []string]
	Services []res.SatisfiedDep[core.ServiceRes, []string]
	Filesets []res.SatisfiedDep[core.FilesetRes, []string]
}

type MissingDeplDeps struct {
	Depl *ResolvedDepl

	Networks []res.MissingDep[core.NetworkRes, []string]
	Services []res.MissingDep[core.ServiceRes, []string]
	Filesets []res.MissingDep[core.FilesetRes, []string]
}
