package structures

import (
	"maps"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var o = struct{}{}

var checkSetAddTests = map[string]struct {
	add []int
	out Set[int]
	non []int
}{
	"{}": {},
	"{1}": {
		add: []int{1},
		out: Set[int]{1: o},
		non: []int{-1, 2},
	},
	"{1 1}": {
		add: []int{1, 1},
		out: Set[int]{1: o},
		non: []int{-1, 2},
	},
	"{1 1 2}": {
		add: []int{1, 1, 2},
		out: Set[int]{1: o, 2: o},
		non: []int{-1},
	},
	"{1 2 1}": {
		add: []int{1, 1, 2},
		out: Set[int]{1: o, 2: o},
		non: []int{-1},
	},
	"{2 1 1}": {
		add: []int{1, 1, 2},
		out: Set[int]{1: o, 2: o},
		non: []int{-1},
	},
}

func TestSetAdd(t *testing.T) {
	t.Parallel()
	for name, test := range checkSetAddTests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			t.Log(name)
			s := make(Set[int])
			for _, elem := range test.add {
				s.Add(elem)
			}
			if got, want := s, test.out; !cmp.Equal(
				got, want, cmpopts.EquateEmpty(), cmpopts.EquateErrors(),
			) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateErrors()))
			}

			t.Logf("%s (has)", name)
			for _, elem := range test.add {
				if got := s; !got.Has(elem) {
					t.Errorf("got is missing elem: %d", elem)
				}
			}

			t.Logf("%s (not has)", name)
			for _, elem := range test.non {
				if got := s; got.Has(elem) {
					t.Errorf("got has spurious elem: %d", elem)
				}
			}
		})
	}
}

var checkSetDiffTests = map[string]struct {
	first  Set[int]
	second Set[int]
	out    Set[int]
}{
	"{} - {}": {},
	"{1} - {}": {
		first: Set[int]{1: o},
		out:   Set[int]{1: o},
	},
	"{} - {1}": {
		second: Set[int]{1: o},
	},
	"{1} - {1}": {
		first:  Set[int]{1: o},
		second: Set[int]{1: o},
	},
	"{1} - {2}": {
		first:  Set[int]{1: o},
		second: Set[int]{2: o},
		out:    Set[int]{1: o},
	},
	"{1 2} - {1}": {
		first:  Set[int]{1: o, 2: o},
		second: Set[int]{1: o},
		out:    Set[int]{2: o},
	},
	"{1} - {1 2}": {
		first:  Set[int]{1: o},
		second: Set[int]{1: o, 2: o},
	},
	"{1 2} - {1 3}": {
		first:  Set[int]{1: o, 2: o},
		second: Set[int]{1: o, 3: o},
		out:    Set[int]{2: o},
	},
}

func TestSetDiff(t *testing.T) {
	t.Parallel()
	for name, test := range checkSetDiffTests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			t.Log(name)
			first := maps.Clone(test.first)
			second := maps.Clone(test.second)
			if got, want := test.first.Difference(test.second), test.out; !cmp.Equal(
				got, want, cmpopts.EquateEmpty(),
			) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}

			t.Logf("%s (first unmodified)", name)
			if got, want := test.first, first; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}

			t.Logf("%s (second unmodified)", name)
			if got, want := test.second, second; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}

			t.Logf("%s (remove equivalence)", name)
			byRemoval := maps.Clone(test.first)
			for elem := range test.second {
				byRemoval.Remove(elem)
			}
			if got, want := byRemoval, test.out; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}
		})
	}
}
