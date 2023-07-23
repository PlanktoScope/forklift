package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

type DeplConflict struct {
	First  *ResolvedDepl
	Second *ResolvedDepl

	// Possible conflicts
	Name      bool
	Listeners []pallets.ResourceConflict[pallets.ListenerResource]
	Networks  []pallets.ResourceConflict[pallets.NetworkResource]
	Services  []pallets.ResourceConflict[pallets.ServiceResource]
}

type MissingDeplDependencies struct {
	Depl *ResolvedDepl

	Networks []pallets.MissingResourceDependency[pallets.NetworkResource]
	Services []pallets.MissingResourceDependency[pallets.ServiceResource]
}
