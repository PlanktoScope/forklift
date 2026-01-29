package cli

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/internal/app/forklift"
	"github.com/forklift-run/forklift/internal/clients/docker"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
	"github.com/forklift-run/forklift/pkg/structures"
)

// Plan builds a plan for changes to make to the Docker host in order to reconcile it with the
// desired state as expressed by the pallet or bundle. The plan is expressed as a dependency graph
// which can be used to build a partial ordering of the changes (where each change is a node in the
// graph) for concurrent execution, and - if serial execution is required either because the
// parallel arg is set to true or because a dependency cycle was detected - a total ordering of the
// changes for serial (rather than concurrent) execution. Redundant dependencies in the dependency
// graph are pruned away (e.g. if A depends on B and B depends on C and A also depends on C, the
// dependency of A on C is omitted).
func Plan(
	indent int, deplsLoader ResolvedDeplsLoader, pkgLoader fplt.FSPkgLoader, parallel bool,
) (
	prunedChangeDeps structures.Digraph[*forklift.ReconciliationChange],
	serialization []*forklift.ReconciliationChange,
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
	deps := fplt.ResolveDeps(satisfiedDeps, true)

	IndentedFprintln(indent, os.Stderr, "Determining and ordering package deployment changes...")
	apps, err := dc.ListApps(context.Background())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't list active Docker Compose apps")
	}
	_, prunedChangeDeps, cycles, serialization, err := forklift.PlanChanges(
		depls, deps, apps, !parallel,
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't compute a plan for changes")
	}

	IndentedFprintln(indent, os.Stderr, "Ordering relationships:")
	printDigraph(indent+1, prunedChangeDeps, "after")
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
		return prunedChangeDeps, nil, nil
	}

	fmt.Fprintln(os.Stderr)
	IndentedFprintln(indent, os.Stderr, "Serialized ordering of package deployment changes:")
	for _, change := range serialization {
		IndentedFprintln(indent+1, os.Stderr, change.PlanString())
	}
	return prunedChangeDeps, serialization, nil
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
