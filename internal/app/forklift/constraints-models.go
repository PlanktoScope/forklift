package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/core"
)

type DeplConflict struct {
	First  *ResolvedDepl
	Second *ResolvedDepl

	// Possible conflicts
	Name      bool
	Listeners []core.ResConflict[core.ListenerRes]
	Networks  []core.ResConflict[core.NetworkRes]
	Services  []core.ResConflict[core.ServiceRes]
}

type SatisfiedDeplDeps struct {
	Depl *ResolvedDepl

	Networks []core.SatisfiedResDep[core.NetworkRes]
	Services []core.SatisfiedResDep[core.ServiceRes]
}

type MissingDeplDeps struct {
	Depl *ResolvedDepl

	Networks []core.MissingResDep[core.NetworkRes]
	Services []core.MissingResDep[core.ServiceRes]
}
