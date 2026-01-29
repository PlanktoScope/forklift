package forklift

import (
	"cmp"
	"slices"

	"github.com/docker/compose/v2/pkg/api"
	"github.com/pkg/errors"

	fplt "github.com/forklift-run/forklift/pkg/pallets"
	"github.com/forklift-run/forklift/pkg/structures"
)

// PlanChanges builds a dependency graph of changes to make to the Docker host (as a plan for
// concurrent execution), for a given list of resolved package deployments, a precomputed graph of
// direct dependency relationships between them, and a list of currently active Compose apps.
// This function also identifies any cycles in the returned dependency graph.
// If the serialize arg is set to true, this function will also compute a non-nil sequential order
// for executing the changes serially (rather than concurrently); otherwise, a nil sequential order
// will be returned.
func PlanChanges(
	depls []*fplt.ResolvedDepl, deplDirectDeps structures.Digraph[string], apps []api.Stack,
	serialize bool,
) (
	changeDirectDeps structures.Digraph[*ReconciliationChange],
	changePrunedDeps structures.Digraph[*ReconciliationChange],
	cycles [][]*ReconciliationChange, serialization []*ReconciliationChange,
	err error,
) {
	// TODO: make a reconciliation plan where relevant resources (i.e. Docker networks) are created
	// simultaneously/independently so that circular dependencies for those resources won't prevent
	// successful application.
	changes, err := identifyReconciliationChanges(depls, apps)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "couldn't identify the changes to make")
	}

	changeDirectDeps = computeChangeDeps(changes, deplDirectDeps)
	changePrunedDeps, changeIndirectDeps, _ := changeDirectDeps.ComputeTransitiveReduction()
	cycles = changeIndirectDeps.IdentifyCycles()
	if !serialize {
		return changeDirectDeps, changePrunedDeps, cycles, nil, nil
	}

	// Serialize changes with a total ordering
	dependents := changeIndirectDeps.Invert()
	slices.SortFunc(changes, func(i, j *ReconciliationChange) int {
		return compareChangesTotal(i, j, changeIndirectDeps, dependents)
	})
	return changeDirectDeps, changePrunedDeps, cycles, changes, nil
}

// computeChangeDeps produces a dependency graph of changes to make on the Docker host based on the
// desired list of deployments, a graph of direct dependencies among those deployments, and a list
// of Docker Compose apps describing the current complete state of the Docker host. The returned
// dependency graph is a map between each reconciliation change and the respective set of any other
// reconciliation changes which must be completed first.
func computeChangeDeps(
	changes []*ReconciliationChange, directDeps structures.Digraph[string],
) structures.Digraph[*ReconciliationChange] {
	removalChanges := make(map[string]*ReconciliationChange)    // keyed by app name
	nonremovalChanges := make(map[string]*ReconciliationChange) // keyed by depl name
	graph := make(structures.Digraph[*ReconciliationChange])
	for _, change := range changes {
		graph.AddNode(change)
		if change.Type == RemoveReconciliationChange {
			removalChanges[change.Name] = change
			continue
		}
		nonremovalChanges[change.Depl.Name] = change
	}
	// FIXME: ideally we would order the removal changes based on dependency relationships between
	// the Compose apps, e.g. with networks. With removals we don't have deployments which would
	// tell us about Docker resource dependency relationships, so we'd need to determine this from
	// Docker. If app r depends on a resource provided by app s, then app r must be removed first -
	// so the removal of app s depends upon the removal of app r.
	// Remove old resources first, in case additions/updates would add overlapping resources.
	for _, change := range nonremovalChanges {
		for _, removalChange := range removalChanges {
			graph.AddEdge(change, removalChange)
		}
	}
	for _, dependent := range nonremovalChanges {
		for deplName := range directDeps[dependent.Depl.Name] {
			if dependency, ok := nonremovalChanges[deplName]; ok {
				graph.AddEdge(dependent, dependency)
			}
		}
	}

	return graph
}

// compareChangesTotal returns a comparison for generating a total ordering of reconciliation
// changes so that they are applied serially and sequentially in a way that will (hopefully) succeed
// for all changes. deps should be a transitive closure of dependencies between changes, and
// dependents should be the inverse of deps.
// This function returns -1 if r should occur before s and 1 if s should occur before r.
func compareChangesTotal(
	r, s *ReconciliationChange, deps, dependents structures.TransitiveClosure[*ReconciliationChange],
) int {
	// Apply the partial ordering from dependencies
	if result := compareReconciliationChangesByDeps(r, s, deps); result != 0 {
		return result
	}

	// Now r and s either are in a circular dependency or have no dependency relationships
	if result := compareDeplsByDepCounts(r, s, deps, dependents); result != 0 {
		return result
	}

	// Compare by names as a last resort
	if r.Depl != nil && s.Depl != nil {
		return cmp.Compare(r.Depl.Name, s.Depl.Name)
	}
	return cmp.Compare(r.Name, s.Name)
}

func compareReconciliationChangesByDeps(
	r, s *ReconciliationChange, deps structures.TransitiveClosure[*ReconciliationChange],
) int {
	rDependsOnS := deps.HasEdge(r, s)
	sDependsOnR := deps.HasEdge(s, r)
	if rDependsOnS && !sDependsOnR {
		return 1
	}
	if !rDependsOnS && sDependsOnR {
		return -1
	}
	return 0
}

func compareDeplsByDepCounts(
	r, s *ReconciliationChange, deps, dependents structures.TransitiveClosure[*ReconciliationChange],
) int {
	// Deployments with greater numbers of dependents go first (needed for correct ordering among
	// unrelated deployments sorted by slices.SortFunc).
	if result := cmp.Compare(len(dependents[s]), len(dependents[r])); result != 0 {
		return result
	}
	// Deployments with greater numbers of dependencies go first (for aesthetic reasons)
	if result := cmp.Compare(len(deps[s]), len(deps[r])); result != 0 {
		return result
	}
	return 0
}
