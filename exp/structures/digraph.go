package structures

import (
	"cmp"
	"fmt"
	"slices"
)

// Digraph

// Digraph is an adjacency matrix.
type Digraph[Node comparable] map[Node]Set[Node]

// AddNode adds the node to the graph. If the node was already in the graph, nothing changes.
func (g Digraph[Node]) AddNode(n Node) {
	if _, ok := g[n]; ok {
		return
	}
	g[n] = make(Set[Node])
}

// AddEdge adds an edge from the first node to the second node. If the edge was already in the
// graph, nothing changes.
func (g Digraph[Node]) AddEdge(from, to Node) {
	g.AddNode(from)
	g.AddNode(to)
	g[from].Add(to)
}

// AddEdge removes any edge from the first node to the second node. If the edge already wasn't in
// the graph, nothing changes.
func (g Digraph[Node]) RemoveEdge(from, to Node) {
	g[from].Remove(to)
}

// HasEdge checks whether an edge exists from the first node to the second node.
func (g Digraph[Node]) HasEdge(from, to Node) bool {
	targetNodes, ok := g[from]
	if !ok {
		return false
	}
	return targetNodes.Has(to)
}

// Invert converts a digraph of children pointing to parents into a new digraph of parents pointing
// to children.
func (g Digraph[Node]) Invert() Digraph[Node] {
	inverted := make(Digraph[Node])
	for node := range g {
		inverted.AddNode(node)
	}
	for child, parents := range g {
		for parent := range parents {
			inverted.AddNode(parent)
			inverted[parent].Add(child)
			inverted.AddNode(child)
		}
	}
	return inverted
}

// ComputeTransitiveClosure removes edges between every pair of nodes which are connected by some
// other longer path of directed edges. For DAGs, this is just the transitive reduction of the
// relation expressed by the digraph. Note: edges in any cycles will be kept.
func (g Digraph[Node]) ComputeTransitiveReduction() (
	tr Digraph[Node], tc TransitiveClosure[Node], cycles [][]Node,
) {
	tc = g.ComputeTransitiveClosure()
	cycles = tc.IdentifyCycles()
	tr = make(Digraph[Node])
	for node := range g {
		tr.AddNode(node)
		for parent := range g[node] {
			tr.AddEdge(node, parent)
		}
	}
	for node := range g {
		ancestors := tc[node]
		for p := range ancestors {
			for q := range ancestors {
				if p == q || p == node || q == node {
					continue
				}
				if tc.edgeInCycle(p, node) || tc.edgeInCycle(p, q) || tc.edgeInCycle(node, q) {
					continue
				}
				if tc.HasEdge(node, p) && tc.HasEdge(p, q) {
					tr.RemoveEdge(node, q)
				}
			}
		}
	}
	return tr, tc, cycles
}

// ComputeTransitiveClosure adds edges between every pair of nodes which are transitively connected
// by some path of directed edges. This is just the transitive closure of the relation expressed by
// the digraph. Iff the digraph isn't a DAG (i.e. iff it has cycles), then each node in the cycle
// will have an edge directed to itself.
func (g Digraph[Node]) ComputeTransitiveClosure() TransitiveClosure[Node] {
	// Seed the transitive closure with the initial digraph
	closure := make(TransitiveClosure[Node])
	prevChangedNodes := make(map[Node]bool)
	changedNodes := make(map[Node]bool)
	for node, parents := range g {
		closure.addNode(node)
		for parent := range parents {
			closure.addEdge(node, parent)
		}
		prevChangedNodes[node] = true
		changedNodes[node] = true
	}
	// This algorithm is very asymptotically inefficient when long paths exist between nodes, but it's
	// easy to understand, and performance is good enough for a typical use case in dependency
	// resolution where dependency trees should be kept relatively shallow.
	for {
		converged := true
		for node, ancestors := range closure {
			initial := len(ancestors)
			for ancestor := range ancestors {
				if !prevChangedNodes[ancestor] {
					// this is just a performance optimization: the ancestor didn't gain any new ancestors
					// in the previous iteration, so we don't need to review it again
					continue
				}
				// Add the ancestor's own ancestors to the child's set of ancestors
				for transitiveAncestor := range closure[ancestor] {
					ancestors.Add(transitiveAncestor)
				}
			}
			final := len(ancestors)
			if changedNodes[node] = initial != final; changedNodes[node] {
				converged = false
			}
		}
		if converged {
			return closure
		}

		prevChangedNodes = changedNodes
		changedNodes = make(map[Node]bool)
	}
}

// Transitive closure

type TransitiveClosure[Node comparable] Digraph[Node]

// addNode adds the node to the graph. If the node was already in the graph, nothing changes.
// This method is private because it should only be used to create a new TransitiveClosure, e.g. as
// part of the Invert method; any other uses will corrupt the state of the transitive closure.
func (g TransitiveClosure[Node]) addNode(n Node) {
	if _, ok := g[n]; ok {
		return
	}
	g[n] = make(Set[Node])
}

// addEdge adds an edge from the first node to the second node. If the edge was already in the
// graph, nothing changes.
func (g TransitiveClosure[Node]) addEdge(from, to Node) {
	g.addNode(from)
	g.addNode(to)
	g[from].Add(to)
}

// HasEdge checks whether an edge exists from the first node to the second node.
func (g TransitiveClosure[Node]) HasEdge(from, to Node) bool {
	targetNodes, ok := g[from]
	if !ok {
		return false
	}
	return targetNodes.Has(to)
}

// IdentifyCycles builds a sorted list of cycles in the transitive closure, where each cycle is a
// list of nodes sorted lexigoraphically by the node's string representation. Sub-cycles of larger
// cycles are simply merged into the larger cycle instead of being listed separately.
func (g TransitiveClosure[Node]) IdentifyCycles() [][]Node {
	cycles := make(map[string][]Node)
	for node, parents := range g {
		if !parents.Has(node) { // this node is not part of a cycle
			continue
		}

		cycle := make([]Node, 0, len(parents))
		for parent := range parents {
			if grandparents := g[parent]; !grandparents.Has(node) { // this parent isn't in the cycle
				continue
			}
			cycle = append(cycle, parent)
		}
		slices.SortFunc(cycle, func(a, b Node) int {
			return cmp.Compare(fmt.Sprintf("%+v", a), fmt.Sprintf("%+v", b))
		})
		cycles[fmt.Sprintf("%+v", cycle)] = cycle
	}

	orderedCycles := make([][]Node, 0, len(cycles))
	for _, cycle := range cycles {
		orderedCycles = append(orderedCycles, cycle)
	}
	slices.SortFunc(orderedCycles, func(a, b []Node) int {
		return cmp.Compare(fmt.Sprintf("%+v", a), fmt.Sprintf("%+v", b))
	})
	return orderedCycles
}

func (g TransitiveClosure[Node]) edgeInCycle(from, to Node) bool {
	parents := g[from]
	if !parents.Has(from) { // the "from" node is not in a cycle
		return false
	}
	if !parents.Has(to) { // the edge does not exist
		return false
	}
	return g[to].Has(from) // the "from" node is a grandparent of itself via the "to" node
}

// Invert converts a digraph of children pointing to parents into a new digraph of parents pointing
// to children.
func (g TransitiveClosure[Node]) Invert() TransitiveClosure[Node] {
	inverted := make(TransitiveClosure[Node])
	for node := range g {
		inverted.addNode(node)
	}
	for child, parents := range g {
		for parent := range parents {
			inverted.addNode(parent)
			inverted[parent].Add(child)
			inverted.addNode(child)
		}
	}
	return inverted
}
