package structures

import (
	"fmt"
	"sort"
)

// MapDigraph

type MapDigraph[Node comparable] interface {
	~map[Node]Set[Node]
}

// Digraph

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

// HasEdge checks whether an edge exists from the first node to the second node.
func (g Digraph[Node]) HasEdge(from, to Node) bool {
	targetNodes, ok := g[from]
	if !ok {
		return false
	}
	return targetNodes.Has(to)
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
	for node, upstreamNodes := range g {
		closure[node] = make(Set[Node])
		for upstreamNode := range upstreamNodes {
			closure[node].Add(upstreamNode)
		}
		prevChangedNodes[node] = true
		changedNodes[node] = true
	}
	// This algorithm is very asymptotically inefficient when long paths exist between nodes, but it's
	// easy to understand, and performance is good enough for a typical use case in dependency
	// resolution where dependency trees should be kept relatively shallow.
	for {
		converged := true
		for node, upstreamNodes := range closure {
			initial := len(upstreamNodes)
			for upstreamNode := range upstreamNodes {
				if !prevChangedNodes[upstreamNode] { // this is just a performance optimization
					continue
				}
				// Add the dependency's own dependencies to the set of dependencies
				transitiveNodes := closure[upstreamNode]
				for transitiveNode := range transitiveNodes {
					upstreamNodes.Add(transitiveNode)
				}
			}
			final := len(upstreamNodes)
			changedNodes[node] = initial != final
			if changedNodes[node] {
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

// Invert converts a digraph of children pointing to parents into a new digraph of parents pointing
// to children.
func (g Digraph[Node]) Invert() Digraph[Node] {
	inverted := make(Digraph[Node])
	for child, parents := range g {
		for parent := range parents {
			inverted.AddNode(parent)
			inverted[parent].Add(child)
			inverted.AddNode(child)
		}
	}
	return inverted
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

// HasEdge checks whether an edge exists from the first node to the second node.
func (g TransitiveClosure[Node]) HasEdge(from, to Node) bool {
	targetNodes, ok := g[from]
	if !ok {
		return false
	}
	return targetNodes.Has(to)
}

// IdentifyCycles builds a sorted list of cycles in the transitive closure, where each cycle is a
// list of nodes sorted lexigoraphically by the node's string representation.
func (g TransitiveClosure[Node]) IdentifyCycles() [][]Node {
	cycles := make(map[string][]Node)
	for node, parents := range g {
		if parents.Has(node) { // this node is part of a cycle
			cycle := make([]Node, 0, len(parents))
			for parent := range parents {
				cycle = append(cycle, parent)
			}
			sort.Slice(cycle, func(i, j int) bool {
				return fmt.Sprintf("%v", cycle[i]) < fmt.Sprintf("%v", cycle[j])
			})
			cycles[fmt.Sprintf("%+v", cycle)] = cycle
		}
	}
	// TODO: sort the cycle
	orderedCycles := make([][]Node, 0, len(cycles))
	for _, cycle := range cycles {
		orderedCycles = append(orderedCycles, cycle)
	}
	sort.Slice(orderedCycles, func(i, j int) bool {
		return fmt.Sprintf("%+v", orderedCycles[i]) < fmt.Sprintf("%+v", orderedCycles[j])
	})
	return orderedCycles
}

// Invert converts a digraph of children pointing to parents into a new digraph of parents pointing
// to children.
func (g TransitiveClosure[Node]) Invert() TransitiveClosure[Node] {
	inverted := make(TransitiveClosure[Node])
	for child, parents := range g {
		for parent := range parents {
			inverted.addNode(parent)
			inverted[parent].Add(child)
			inverted.addNode(child)
		}
	}
	return inverted
}
