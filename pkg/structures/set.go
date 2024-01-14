// Package structures provides a few generic data structures.
package structures

type Set[Node comparable] map[Node]struct{}

// Add adds the node to the set. If the node was already in the set, nothing changes.
func (s Set[Node]) Add(n Node) {
	s[n] = struct{}{}
}

func (s Set[Node]) Has(n Node) bool {
	_, ok := s[n]
	return ok
}
