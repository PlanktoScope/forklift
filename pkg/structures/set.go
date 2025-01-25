// Package structures provides a few generic data structures.
package structures

type Set[Node comparable] map[Node]struct{}

// Add adds the node to the set. If the node was already in the set, nothing changes.
func (s Set[Node]) Add(n ...Node) {
	for _, node := range n {
		s[node] = struct{}{}
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
