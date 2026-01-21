package forklift

import (
	"github.com/forklift-run/forklift/pkg/core"
)

type DeplConflict struct {
	First  *ResolvedDepl
	Second *ResolvedDepl

	// Possible conflicts
	Name        bool
	Listeners   []core.ResConflict[core.ListenerRes]
	Networks    []core.ResConflict[core.NetworkRes]
	Services    []core.ResConflict[core.ServiceRes]
	Filesets    []core.ResConflict[core.FilesetRes]
	FileExports []core.ResConflict[core.FileExportRes]
}

type SatisfiedDeplDeps struct {
	Depl *ResolvedDepl

	Networks []core.SatisfiedResDep[core.NetworkRes]
	Services []core.SatisfiedResDep[core.ServiceRes]
	Filesets []core.SatisfiedResDep[core.FilesetRes]
}

type MissingDeplDeps struct {
	Depl *ResolvedDepl

	Networks []core.MissingResDep[core.NetworkRes]
	Services []core.MissingResDep[core.ServiceRes]
	Filesets []core.MissingResDep[core.FilesetRes]
}
