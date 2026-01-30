// Package structures provides a few generic data structures.
package structures

import (
	"iter"
	"maps"
)

type Set[Node comparable] map[Node]struct{}

// Add adds each node to the set for each node not already in the set.
func (s Set[Node]) Add(n ...Node) {
	for _, node := range n {
		s[node] = struct{}{}
	}
}

// Remove removes each node from the set for each node still in the set.
func (s Set[Node]) Remove(n ...Node) {
	for _, node := range n {
		delete(s, node)
	}
}

// Has checks whether the node is already in the set.
func (s Set[Node]) Has(n Node) bool {
	_, ok := s[n]
	return ok
}

// Difference creates a new set with the difference between the set whose method is called and the
// provided set.
func (s Set[Node]) Difference(t Set[Node]) Set[Node] {
	difference := make(Set[Node])
	for node := range s {
		if !t.Has(node) {
			difference.Add(node)
		}
	}
	return difference
}

// All returns an iterator over all elements in s.
func (s Set[Node]) All() iter.Seq[Node] {
	return maps.Keys(s)
}
