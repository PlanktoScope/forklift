package forklift

import (
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

type DeplConflict struct {
	First  Depl
	Second Depl

	// Possible conflicts
	Name      bool
	Listeners []pallets.ResourceConflict[pallets.ListenerResource]
	Networks  []pallets.ResourceConflict[pallets.NetworkResource]
	Services  []pallets.ResourceConflict[pallets.ServiceResource]
}

type MissingDeplDependencies struct {
	Depl Depl

	Networks []pallets.MissingResourceDependency[pallets.NetworkResource]
	Services []pallets.MissingResourceDependency[pallets.ServiceResource]
}
