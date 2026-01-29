package cli

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/docker/compose/v2/pkg/api"
	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/internal/app/forklift"
	"github.com/forklift-run/forklift/internal/clients/docker"
	"github.com/forklift-run/forklift/pkg/structures"
)

const (
	addReconciliationChange    = "add"
	removeReconciliationChange = "remove"
	updateReconciliationChange = "update"
)

type ReconciliationChange struct {
	Name string
	Type string
	Depl *forklift.ResolvedDepl // this is nil for an app to be removed
	App  api.Stack              // this is empty for an app which does not yet exist
}

func (c *ReconciliationChange) String() string {
	if c.Depl == nil {
		return fmt.Sprintf("(%s %s)", c.Type, c.Name)
	}
	return fmt.Sprintf("(%s %s)", c.Type, c.Depl.Name)
}

func (c *ReconciliationChange) PlanString() string {
	if c.Depl == nil {
		return fmt.Sprintf("%s Compose app %s (from unknown deployment)", c.Type, c.Name)
	}
	return fmt.Sprintf("%s deployment %s as Compose app %s", c.Type, c.Depl.Name, c.Name)
}

func newAddReconciliationChange(
	deplName string, depl *forklift.ResolvedDepl,
) *ReconciliationChange {
	return &ReconciliationChange{
		Name: forklift.GetComposeAppName(deplName),
		Type: addReconciliationChange,
		Depl: depl,
	}
}

func newUpdateReconciliationChange(
	deplName string, depl *forklift.ResolvedDepl, app api.Stack,
) *ReconciliationChange {
	return &ReconciliationChange{
		Name: forklift.GetComposeAppName(deplName),
		Type: updateReconciliationChange,
		Depl: depl,
		App:  app,
	}
}

func newRemoveReconciliationChange(appName string, app api.Stack) *ReconciliationChange {
	return &ReconciliationChange{
		Name: appName,
		Type: removeReconciliationChange,
		App:  app,
	}
}

// Plan builds a plan for changes to make to the Docker host in order to reconcile it with the
// desired state as expressed by the pallet or bundle. The plan is expressed as a dependency graph
// which can be used to build a partial ordering of the changes (where each change is a node in the
// graph) for concurrent execution, and - if serial execution is required either because the
// parallel arg is set to true or because a dependency cycle was detected - a total ordering of the
// changes for serial (rather than concurrent) execution.
func Plan(
	indent int, deplsLoader ResolvedDeplsLoader, pkgLoader forklift.FSPkgLoader, parallel bool,
) (
	changeDeps structures.Digraph[*ReconciliationChange], serialization []*ReconciliationChange,
	err error,
) {
	dc, err := docker.NewClient()
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't make Docker API client")
	}

	depls, satisfiedDeps, err := Check(indent, deplsLoader, pkgLoader)
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't ensure validity")
	}
	// Always skip nonblocking dependency relationships - even for serial execution, they don't need
	// to be considered for a total ordering. And we don't want nonblocking dependency relationships
	// to count towards dependency cycles. And it's simpler to just have the same behavior (and the
	// same resulting dependency graph) regardless of serial vs. concurrent execution.
	deps := forklift.ResolveDeps(satisfiedDeps, true)

	IndentedFprintln(indent, os.Stderr, "Determining and ordering package deployment changes...")
	apps, err := dc.ListApps(context.Background())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't list active Docker Compose apps")
	}
	changeDeps, cycles, serialization, err := planChanges(depls, deps, apps, !parallel)
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't compute a plan for changes")
	}

	IndentedFprintln(indent, os.Stderr, "Ordering relationships:")
	printDigraph(indent+1, changeDeps, "after")
	if len(cycles) > 0 {
		IndentedFprintln(indent, os.Stderr, "Detected ordering cycles:")
		for _, cycle := range cycles {
			IndentedFprintf(indent+1, os.Stderr, "cycle between: %s\n", cycle)
		}
		if parallel {
			return nil, nil, errors.Errorf(
				"concurrent plan would deadlock due to ordering cycles (try a serial plan instead): %+v",
				cycles,
			)
		}
	}
	if serialization == nil {
		return changeDeps, nil, nil
	}

	fmt.Fprintln(os.Stderr)
	IndentedFprintln(indent, os.Stderr, "Serialized ordering of package deployment changes:")
	for _, change := range serialization {
		IndentedFprintln(indent+1, os.Stderr, change.PlanString())
	}
	return changeDeps, serialization, nil
}

type MapDigraph[Node comparable] interface {
	~map[Node]structures.Set[Node]
}

func printDigraph[Node comparable, Digraph MapDigraph[Node]](
	indent int, digraph Digraph, edgeType string,
) {
	sortedNodes := make([]Node, 0, len(digraph))
	for node := range digraph {
		sortedNodes = append(sortedNodes, node)
	}
	slices.SortFunc(sortedNodes, func(i, j Node) int {
		return cmp.Compare(fmt.Sprintf("%v", i), fmt.Sprintf("%v", j))
	})
	for _, node := range sortedNodes {
		printNodeOutboundEdges(indent, digraph, node, edgeType)
	}
}

func printNodeOutboundEdges[Node comparable, Digraph MapDigraph[Node]](
	indent int, digraph Digraph, node Node, edgeType string,
) {
	upstreamNodes := make([]Node, 0, len(digraph[node]))
	for dep := range digraph[node] {
		upstreamNodes = append(upstreamNodes, dep)
	}
	slices.SortFunc(upstreamNodes, func(i, j Node) int {
		return cmp.Compare(fmt.Sprintf("%v", i), fmt.Sprintf("%v", j))
	})
	if len(upstreamNodes) == 0 {
		IndentedFprintf(indent, os.Stderr, "%v %s nothing", node, edgeType)
	} else {
		IndentedFprintf(indent, os.Stderr, "%v %s: %+v", node, edgeType, upstreamNodes)
	}
	fmt.Fprintln(os.Stderr)
}

// planChanges builds a dependency graph of changes to make to the Docker host (as a plan for
// concurrent execution), for a given list of resolved package deployments, a precomputed graph of
// direct dependency relationships between them, and a list of currently active Compose apps.
// This function also identifies any cycles in the returned dependency graph.
// If the serialize arg is set to true, this function will also compute a non-nil sequential order
// for executing the changes serially (rather than concurrently); otherwise, a nil sequential order
// will be returned.
func planChanges(
	depls []*forklift.ResolvedDepl, deplDirectDeps structures.Digraph[string], apps []api.Stack,
	serialize bool,
) (
	changeDirectDeps structures.Digraph[*ReconciliationChange], cycles [][]*ReconciliationChange,
	serialization []*ReconciliationChange, err error,
) {
	// TODO: make a reconciliation plan where relevant resources (i.e. Docker networks) are created
	// simultaneously/independently so that circular dependencies for those resources won't prevent
	// successful application.
	changes, err := identifyReconciliationChanges(depls, apps)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "couldn't identify the changes to make")
	}

	changeDirectDeps = computeChangeDeps(changes, deplDirectDeps)
	changeIndirectDeps := changeDirectDeps.ComputeTransitiveClosure()
	cycles = changeIndirectDeps.IdentifyCycles()
	if !serialize {
		return changeDirectDeps, cycles, nil, nil
	}

	// Serialize changes with a total ordering
	dependents := changeIndirectDeps.Invert()
	slices.SortFunc(changes, func(i, j *ReconciliationChange) int {
		return compareChangesTotal(i, j, changeIndirectDeps, dependents)
	})
	return changeDirectDeps, cycles, changes, nil
}

// identifyReconciliationChanges builds an arbitrarily-ordered list of changes to carry out to
// reconcile the desired list of deployments with the actual list of active Docker Compose apps.
func identifyReconciliationChanges(
	depls []*forklift.ResolvedDepl, apps []api.Stack,
) ([]*ReconciliationChange, error) {
	deplsByName := make(map[string]*forklift.ResolvedDepl)
	for _, depl := range depls {
		deplsByName[depl.Name] = depl
	}
	appsByName := make(map[string]api.Stack)
	for _, app := range apps {
		appsByName[app.Name] = app
	}
	composeAppDefinerSet, err := identifyComposeAppDefiners(deplsByName)
	if err != nil {
		return nil, err
	}

	appDeplNames := make(map[string]string)
	changes := make([]*ReconciliationChange, 0, len(depls)+len(apps))
	for name, depl := range deplsByName {
		appDeplNames[forklift.GetComposeAppName(name)] = name
		app, ok := appsByName[forklift.GetComposeAppName(name)]
		if !ok {
			if composeAppDefinerSet.Has(name) {
				changes = append(changes, newAddReconciliationChange(name, depl))
			}
			continue
		}
		if composeAppDefinerSet.Has(name) {
			changes = append(changes, newUpdateReconciliationChange(name, depl, app))
		}
	}
	for name, app := range appsByName {
		if deplName, ok := appDeplNames[name]; ok {
			if composeAppDefinerSet.Has(deplName) {
				continue
			}
		}
		changes = append(changes, newRemoveReconciliationChange(name, app))
	}
	return changes, nil
}

// identifyComposeAppDefiners builds a set of the names of deployments which define Compose apps.
func identifyComposeAppDefiners(
	depls map[string]*forklift.ResolvedDepl,
) (structures.Set[string], error) {
	composeAppDefinerSet := make(structures.Set[string])
	for _, depl := range depls {
		definesApp, err := depl.DefinesComposeApp()
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't determine whether package deployment %s defines a Compose app", depl.Name,
			)
		}
		if definesApp {
			composeAppDefinerSet.Add(depl.Name)
		}
	}
	return composeAppDefinerSet, nil
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
		if change.Type == removeReconciliationChange {
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
