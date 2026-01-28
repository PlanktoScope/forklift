package structures

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type edge struct {
	from int
	to   int
}

var checkDigraphAddTests = map[string]struct {
	nodes []int
	edges []edge
	out   Digraph[int]
	non   []edge
}{
	"{} {}": {},
	"{1} {}": {
		nodes: []int{1},
		out:   Digraph[int]{1: nil},
		non: []edge{
			{from: 1, to: 1},
			{from: 1, to: 2},
			{from: 2, to: 1},
			{from: 2, to: 2},
		},
	},
	"{1 1} {}": {
		nodes: []int{1, 1},
		out:   Digraph[int]{1: nil},
		non: []edge{
			{from: 1, to: 1},
			{from: 1, to: 2},
			{from: 2, to: 1},
			{from: 2, to: 2},
		},
	},
	"{1} {1->1}": {
		nodes: []int{1},
		edges: []edge{{from: 1, to: 1}},
		out:   Digraph[int]{1: Set[int]{1: o}},
		non: []edge{
			{from: 1, to: 2},
			{from: 2, to: 1},
			{from: 2, to: 2},
		},
	},
	"{1} {1->1 1->1}": {
		nodes: []int{1},
		edges: []edge{{from: 1, to: 1}, {from: 1, to: 1}},
		out:   Digraph[int]{1: Set[int]{1: o}},
		non: []edge{
			{from: 1, to: 2},
			{from: 2, to: 1},
			{from: 2, to: 2},
		},
	},
	"{} {1->1}": {
		edges: []edge{{from: 1, to: 1}},
		out:   Digraph[int]{1: Set[int]{1: o}},
		non: []edge{
			{from: 1, to: 2},
			{from: 2, to: 1},
			{from: 2, to: 2},
		},
	},
	"{1 2} {}": {
		nodes: []int{1, 2},
		out:   Digraph[int]{1: nil, 2: nil},
		non: []edge{
			{from: 1, to: 1},
			{from: 1, to: 2},
			{from: 2, to: 1},
			{from: 2, to: 2},
		},
	},
	"{1 2} {1->1}": {
		nodes: []int{1, 2},
		edges: []edge{{from: 1, to: 1}},
		out:   Digraph[int]{1: Set[int]{1: o}, 2: nil},
		non: []edge{
			{from: 1, to: 2},
			{from: 2, to: 1},
			{from: 2, to: 2},
		},
	},
	"{2} {1->1}": {
		nodes: []int{2},
		edges: []edge{{from: 1, to: 1}},
		out:   Digraph[int]{1: Set[int]{1: o}, 2: nil},
		non: []edge{
			{from: 1, to: 2},
			{from: 2, to: 1},
			{from: 2, to: 2},
		},
	},
	"{1 2} {1->2}": {
		nodes: []int{1, 2},
		edges: []edge{{from: 1, to: 2}},
		out:   Digraph[int]{1: Set[int]{2: o}, 2: nil},
		non: []edge{
			{from: 1, to: 1},
			{from: 2, to: 1},
			{from: 2, to: 2},
		},
	},
	"{1} {1->2}": {
		nodes: []int{1},
		edges: []edge{{from: 1, to: 2}},
		out:   Digraph[int]{1: Set[int]{2: o}, 2: nil},
		non: []edge{
			{from: 1, to: 1},
			{from: 2, to: 1},
			{from: 2, to: 2},
		},
	},
	"{2} {1->2}": {
		nodes: []int{2},
		edges: []edge{{from: 1, to: 2}},
		out:   Digraph[int]{1: Set[int]{2: o}, 2: nil},
		non: []edge{
			{from: 1, to: 1},
			{from: 2, to: 1},
			{from: 2, to: 2},
		},
	},
	"{} {1->2}": {
		edges: []edge{{from: 1, to: 2}},
		out:   Digraph[int]{1: Set[int]{2: o}, 2: nil},
		non: []edge{
			{from: 1, to: 1},
			{from: 2, to: 1},
			{from: 2, to: 2},
		},
	},
	"{} {1->2 1->2}": {
		edges: []edge{{from: 1, to: 2}, {from: 1, to: 2}},
		out:   Digraph[int]{1: Set[int]{2: o}, 2: nil},
		non: []edge{
			{from: 1, to: 1},
			{from: 2, to: 1},
			{from: 2, to: 2},
		},
	},
	"{3} {1->2 2->1}": {
		nodes: []int{3},
		edges: []edge{{from: 1, to: 2}, {from: 2, to: 1}},
		out:   Digraph[int]{1: Set[int]{2: o}, 2: Set[int]{1: o}, 3: nil},
		non: []edge{
			{from: 1, to: 1},
			{from: 2, to: 2},
			{from: 3, to: 3},
			{from: 1, to: 3},
			{from: 3, to: 1},
			{from: 2, to: 3},
			{from: 3, to: 2},
		},
	},
}

func TestDigraphAdd(t *testing.T) {
	t.Parallel()
	for name, test := range checkDigraphAddTests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			t.Logf("%s (add nodes first)", name)
			g := make(Digraph[int])
			for _, elem := range test.nodes {
				g.AddNode(elem)
			}
			for _, elem := range test.edges {
				g.AddEdge(elem.from, elem.to)
			}
			if got, want := g, test.out; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}

			t.Logf("%s (has)", name)
			for _, elem := range test.edges {
				if got := g; !got.HasEdge(elem.from, elem.to) {
					t.Errorf("got is missing edge: %d->%d", elem.from, elem.to)
				}
			}

			t.Logf("%s (not has)", name)
			for _, elem := range test.non {
				if got := g; got.HasEdge(elem.from, elem.to) {
					t.Errorf("got has spurious edge: %d->%d", elem.from, elem.to)
				}
			}
		})
	}
}

func TestDigraphAddCommutativity(t *testing.T) {
	t.Parallel()
	for name, test := range checkDigraphAddTests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			t.Logf("%s (add nodes first)", name)
			g := make(Digraph[int])
			for _, elem := range test.nodes {
				g.AddNode(elem)
			}
			for _, elem := range test.edges {
				g.AddEdge(elem.from, elem.to)
			}
			if got, want := g, test.out; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}

			t.Logf("%s (add edges first)", name)
			gg := make(Digraph[int])
			for _, elem := range test.edges {
				gg.AddEdge(elem.from, elem.to)
			}
			for _, elem := range test.nodes {
				gg.AddNode(elem)
			}
			if got, want := gg, test.out; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}
		})
	}
}

var checkDigraphInvertTests = map[string]struct {
	nodes []int
	edges []edge
	out   Digraph[int]
}{
	"{} {}": {},
	"{1} {}": {
		nodes: []int{1},
		out:   Digraph[int]{1: nil},
	},
	"{} {1->1}": {
		edges: []edge{{from: 1, to: 1}},
		out:   Digraph[int]{1: Set[int]{1: o}},
	},
	"{} {1->2}": {
		edges: []edge{{from: 1, to: 2}},
		out:   Digraph[int]{1: nil, 2: Set[int]{1: o}},
	},
	"{3} {1->2}": {
		nodes: []int{3},
		edges: []edge{{from: 1, to: 2}},
		out:   Digraph[int]{1: nil, 2: Set[int]{1: o}, 3: nil},
	},
	"{} {1->2 1->3}": {
		edges: []edge{{from: 1, to: 2}, {from: 1, to: 3}},
		out:   Digraph[int]{1: nil, 2: Set[int]{1: o}, 3: Set[int]{1: o}},
	},
	"{} {1->2 2->3}": {
		edges: []edge{{from: 1, to: 2}, {from: 2, to: 3}},
		out:   Digraph[int]{1: nil, 2: Set[int]{1: o}, 3: Set[int]{2: o}},
	},
}

func TestDigraphInvert(t *testing.T) {
	t.Parallel()
	for name, test := range checkDigraphInvertTests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			t.Log(name)
			g := make(Digraph[int])
			gg := make(Digraph[int])
			for _, elem := range test.nodes {
				g.AddNode(elem)
				gg.AddNode(elem)
			}
			for _, elem := range test.edges {
				g.AddEdge(elem.from, elem.to)
				gg.AddEdge(elem.from, elem.to)
			}
			if got, want := g.Invert(), test.out; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}

			t.Logf("%s (unmodified)", name)
			if got, want := g, gg; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}

			t.Logf("%s (double-invert)", name)
			if got, want := g.Invert().Invert(), g; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}
		})
	}
}

var checkDigraphTransClosureTests = map[string]struct {
	nodes []int
	edges []edge
	out   TransitiveClosure[int]
	cyc   [][]int
	inv   TransitiveClosure[int]
}{
	"{} {}": {},
	"{1} {}": {
		nodes: []int{1},
		out:   TransitiveClosure[int]{1: nil},
	},
	"{} {1->1}": {
		edges: []edge{{from: 1, to: 1}},
		out:   TransitiveClosure[int]{1: Set[int]{1: o}},
		cyc:   [][]int{{1}},
	},
	"{} {1->2}": {
		edges: []edge{{from: 1, to: 2}},
		out:   TransitiveClosure[int]{1: Set[int]{2: o}, 2: nil},
	},
	"{3} {1->2 1->1}": {
		nodes: []int{3},
		edges: []edge{{from: 1, to: 2}, {from: 1, to: 1}},
		out:   TransitiveClosure[int]{1: Set[int]{1: o, 2: o}, 2: nil, 3: nil},
		cyc:   [][]int{{1}},
	},
	"{3} {1->2 2->2}": {
		nodes: []int{3},
		edges: []edge{{from: 1, to: 2}, {from: 2, to: 2}},
		out:   TransitiveClosure[int]{1: Set[int]{2: o}, 2: Set[int]{2: o}, 3: nil},
		cyc:   [][]int{{2}},
	},
	"{3} {1->2 1->2}": {
		nodes: []int{3},
		edges: []edge{{from: 1, to: 2}, {from: 1, to: 2}},
		out:   TransitiveClosure[int]{1: Set[int]{2: o}, 2: nil, 3: nil},
	},
	"{3} {1->2 2->1}": {
		nodes: []int{3},
		edges: []edge{{from: 1, to: 2}, {from: 2, to: 1}},
		out:   TransitiveClosure[int]{1: Set[int]{1: o, 2: o}, 2: Set[int]{1: o, 2: o}, 3: nil},
		cyc:   [][]int{{1, 2}},
	},
	"{} {1->2 1->3}": {
		edges: []edge{{from: 1, to: 2}, {from: 1, to: 3}},
		out:   TransitiveClosure[int]{1: Set[int]{2: o, 3: o}, 2: nil, 3: nil},
	},
	"{} {1->2 2->3}": {
		edges: []edge{{from: 1, to: 2}, {from: 2, to: 3}},
		out:   TransitiveClosure[int]{1: Set[int]{2: o, 3: o}, 2: Set[int]{3: o}, 3: nil},
	},
	"{} {1->2 2->1 3->1}": {
		nodes: []int{},
		edges: []edge{{from: 1, to: 2}, {from: 2, to: 1}, {from: 3, to: 1}},
		out: TransitiveClosure[int]{
			1: Set[int]{1: o, 2: o},
			2: Set[int]{1: o, 2: o},
			3: Set[int]{1: o, 2: o},
		},
		cyc: [][]int{{1, 2}},
	},
	"{} {1->2 2->1 1->3}": {
		nodes: []int{},
		edges: []edge{{from: 1, to: 2}, {from: 2, to: 1}, {from: 1, to: 3}},
		out: TransitiveClosure[int]{
			1: Set[int]{1: o, 2: o, 3: o},
			2: Set[int]{1: o, 2: o, 3: o},
			3: nil,
		},
		cyc: [][]int{{1, 2}},
	},
	"{} {1->2 2->1 1->3 3->1}": {
		nodes: []int{},
		edges: []edge{{from: 1, to: 2}, {from: 2, to: 1}, {from: 1, to: 3}, {from: 3, to: 1}},
		out: TransitiveClosure[int]{
			1: Set[int]{1: o, 2: o, 3: o},
			2: Set[int]{1: o, 2: o, 3: o},
			3: Set[int]{1: o, 2: o, 3: o},
		},
		cyc: [][]int{{1, 2, 3}},
	},
	"{} {1->2 2->3 3->1}": {
		edges: []edge{{from: 1, to: 2}, {from: 2, to: 3}, {from: 3, to: 1}},
		out: TransitiveClosure[int]{
			1: Set[int]{1: o, 2: o, 3: o},
			2: Set[int]{1: o, 2: o, 3: o},
			3: Set[int]{1: o, 2: o, 3: o},
		},
		cyc: [][]int{{1, 2, 3}},
	},
	"{} {1->2 2->3 1->4}": {
		edges: []edge{{from: 1, to: 2}, {from: 2, to: 3}, {from: 1, to: 4}},
		out: TransitiveClosure[int]{
			1: Set[int]{2: o, 3: o, 4: o},
			2: Set[int]{3: o},
			3: nil,
			4: nil,
		},
	},
	"{} {1->2 2->3 2->4}": {
		edges: []edge{{from: 1, to: 2}, {from: 2, to: 3}, {from: 2, to: 4}},
		out: TransitiveClosure[int]{
			1: Set[int]{2: o, 3: o, 4: o},
			2: Set[int]{3: o, 4: o},
			3: nil,
			4: nil,
		},
	},
	"{} {1->2 2->1 3->4 4->3}": {
		edges: []edge{{from: 1, to: 2}, {from: 2, to: 1}, {from: 3, to: 4}, {from: 4, to: 3}},
		out: TransitiveClosure[int]{
			1: Set[int]{1: o, 2: o},
			2: Set[int]{1: o, 2: o},
			3: Set[int]{3: o, 4: o},
			4: Set[int]{3: o, 4: o},
		},
		cyc: [][]int{{1, 2}, {3, 4}},
	},
	"{} {1->2 2->1 1->3 3->2}": {
		nodes: []int{},
		edges: []edge{{from: 1, to: 2}, {from: 2, to: 1}, {from: 1, to: 3}, {from: 3, to: 2}},
		out: TransitiveClosure[int]{
			1: Set[int]{1: o, 2: o, 3: o},
			2: Set[int]{1: o, 2: o, 3: o},
			3: Set[int]{1: o, 2: o, 3: o},
		},
		cyc: [][]int{{1, 2, 3}},
	},
	"{} {1->2 2->1 1->3 3->4 4->3}": {
		nodes: []int{},
		edges: []edge{
			{from: 1, to: 2},
			{from: 2, to: 1},
			{from: 1, to: 3},
			{from: 3, to: 4},
			{from: 4, to: 3},
		},
		out: TransitiveClosure[int]{
			1: Set[int]{1: o, 2: o, 3: o, 4: o},
			2: Set[int]{1: o, 2: o, 3: o, 4: o},
			3: Set[int]{3: o, 4: o},
			4: Set[int]{3: o, 4: o},
		},
		cyc: [][]int{{1, 2}, {3, 4}},
	},
	"{} {1->2 2->1 1->3 3->4 4->3 4->2}": {
		nodes: []int{},
		edges: []edge{
			{from: 1, to: 2},
			{from: 2, to: 1},
			{from: 1, to: 3},
			{from: 3, to: 4},
			{from: 4, to: 3},
			{from: 4, to: 2},
		},
		out: TransitiveClosure[int]{
			1: Set[int]{1: o, 2: o, 3: o, 4: o},
			2: Set[int]{1: o, 2: o, 3: o, 4: o},
			3: Set[int]{1: o, 2: o, 3: o, 4: o},
			4: Set[int]{1: o, 2: o, 3: o, 4: o},
		},
		cyc: [][]int{{1, 2, 3, 4}},
	},
}

func TestDigraphTransClosure(t *testing.T) {
	t.Parallel()
	for name, test := range checkDigraphTransClosureTests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			t.Log(name)
			g := make(Digraph[int])
			gg := make(Digraph[int])
			for _, elem := range test.nodes {
				g.AddNode(elem)
				gg.AddNode(elem)
			}
			for _, elem := range test.edges {
				g.AddEdge(elem.from, elem.to)
				gg.AddEdge(elem.from, elem.to)
			}
			tc := g.ComputeTransitiveClosure()
			if got, want := tc, test.out; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}

			t.Logf("%s (unmodified)", name)
			if got, want := g, gg; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}

			t.Logf("%s (double-transclosure)", name)
			if got, want := Digraph[int](tc).ComputeTransitiveClosure(), tc; !cmp.Equal(
				got, want, cmpopts.EquateEmpty(),
			) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}

			t.Logf("%s (cycles)", name)
			if got, want := tc.IdentifyCycles(), test.cyc; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}

			t.Logf("%s (inverse)", name)
			if got, want := tc.Invert(), TransitiveClosure[int](Digraph[int](tc).Invert()); !cmp.Equal(
				got, want, cmpopts.EquateEmpty(),
			) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}
		})
	}
}

func TestDigraphTransClosureHasEdge(t *testing.T) {
	t.Parallel()
	for name, test := range checkDigraphTransClosureTests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			t.Log(name)
			g := make(Digraph[int])
			for _, elem := range test.nodes {
				g.AddNode(elem)
			}
			for _, elem := range test.edges {
				g.AddEdge(elem.from, elem.to)
			}
			tc := g.ComputeTransitiveClosure()
			if got, want := tc, test.out; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}
			tcg := Digraph[int](tc)

			for _, a := range []int{1, 2, 3, 4} {
				for _, b := range []int{1, 2, 3, 4} {
					if got, want := tc.HasEdge(a, b), tcg.HasEdge(a, b); got != want {
						t.Errorf("TransitiveClosure.HasEdge != Digraph.HasEdge: %d->%d", a, b)
					}
				}
			}
		})
	}
}
