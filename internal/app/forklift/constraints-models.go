package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

type DeplConflict struct {
	First  *ResolvedDepl
	Second *ResolvedDepl

	// Possible conflicts
	Name      bool
	Listeners []pallets.ResConflict[pallets.ListenerRes]
	Networks  []pallets.ResConflict[pallets.NetworkRes]
	Services  []pallets.ResConflict[pallets.ServiceRes]
}

type SatisfiedDeplDeps struct {
	Depl *ResolvedDepl

	Networks []pallets.SatisfiedResDep[pallets.NetworkRes]
	Services []pallets.SatisfiedResDep[pallets.ServiceRes]
}

type MissingDeplDeps struct {
	Depl *ResolvedDepl

	Networks []pallets.MissingResDep[pallets.NetworkRes]
	Services []pallets.MissingResDep[pallets.ServiceRes]
}
