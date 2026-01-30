package resources

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type checkConflictsIn struct {
	First  []attached
	Second []attached
}

var checkConflictsTests = map[string]struct {
	in  checkConflictsIn
	out []conflict
}{
	"first-x0 second-x0": {},
	"first-x1 second-x0": {
		in: checkConflictsIn{
			First: []attached{{Res: resA1, Origin: pkg1}},
		},
	},
	"first-x1 second-x1 | conflict-x0": {
		in: checkConflictsIn{
			First:  []attached{{Res: resA1, Origin: pkg1}},
			Second: []attached{{Res: resB2, Origin: pkg2}},
		},
	},
	"first-x1 second-x1 | conflict-x1": {
		in: checkConflictsIn{
			First:  []attached{{Res: resA1, Origin: pkg1}},
			Second: []attached{{Res: resA2, Origin: pkg2}},
		},
		out: []conflict{{
			First:  attached{Res: resA1, Origin: pkg1},
			Second: attached{Res: resA2, Origin: pkg2},
			Errs:   []error{errSameA},
		}},
	},
	"first-x1 second-x1 | conflict-x2": {
		in: checkConflictsIn{
			First:  []attached{{Res: resA1, Origin: pkg1}},
			Second: []attached{{Res: resA1, Origin: pkg2}},
		},
		out: []conflict{{
			First:  attached{Res: resA1, Origin: pkg1},
			Second: attached{Res: resA1, Origin: pkg2},
			Errs:   []error{errSameA, errSame1},
		}},
	},
	"first-x2-ident second-x1 | conflict-x0": {
		in: checkConflictsIn{
			First: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA1, Origin: pkg1},
			},
			Second: []attached{{Res: resB2, Origin: pkg2}},
		},
	},
	"first-x2-ident second-x2-ident | conflict-x0": {
		in: checkConflictsIn{
			First: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA1, Origin: pkg1},
			},
			Second: []attached{
				{Res: resB2, Origin: pkg2},
				{Res: resB2, Origin: pkg2},
			},
		},
	},
	"first-x2-ident second x1 | conflict-x2": {
		in: checkConflictsIn{
			First: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA1, Origin: pkg1},
			},
			Second: []attached{{Res: resA2, Origin: pkg2}},
		},
		out: []conflict{
			{
				First:  attached{Res: resA1, Origin: pkg1},
				Second: attached{Res: resA2, Origin: pkg2},
				Errs:   []error{errSameA},
			},
			{
				First:  attached{Res: resA1, Origin: pkg1},
				Second: attached{Res: resA2, Origin: pkg2},
				Errs:   []error{errSameA},
			},
		},
	},
	"first-x2-ident second-x1 | conflict-x4": {
		in: checkConflictsIn{
			First: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA1, Origin: pkg1},
			},
			Second: []attached{{Res: resA1, Origin: pkg2}},
		},
		out: []conflict{
			{
				First:  attached{Res: resA1, Origin: pkg1},
				Second: attached{Res: resA1, Origin: pkg2},
				Errs:   []error{errSameA, errSame1},
			},
			{
				First:  attached{Res: resA1, Origin: pkg1},
				Second: attached{Res: resA1, Origin: pkg2},
				Errs:   []error{errSameA, errSame1},
			},
		},
	},
	"first-x2 second-x1 | conflict-x1": {
		in: checkConflictsIn{
			First: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA2, Origin: pkg1},
			},
			Second: []attached{{Res: resB2, Origin: pkg2}},
		},
		out: []conflict{
			{
				First:  attached{Res: resA2, Origin: pkg1},
				Second: attached{Res: resB2, Origin: pkg2},
				Errs:   []error{errSame2},
			},
		},
	},
	"first-x2 second-x2 | conflict-x2": {
		in: checkConflictsIn{
			First: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA2, Origin: pkg1},
			},
			Second: []attached{
				{Res: resB1, Origin: pkg2},
				{Res: resB2, Origin: pkg2},
			},
		},
		out: []conflict{
			{
				First:  attached{Res: resA1, Origin: pkg1},
				Second: attached{Res: resB1, Origin: pkg2},
				Errs:   []error{errSame1},
			},
			{
				First:  attached{Res: resA2, Origin: pkg1},
				Second: attached{Res: resB2, Origin: pkg2},
				Errs:   []error{errSame2},
			},
		},
	},
}

func TestCheckConflicts(t *testing.T) {
	t.Parallel()
	for name, test := range checkConflictsTests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			t.Log(name)
			if got, want := CheckConflicts(
				test.in.First, test.in.Second,
			), test.out; !cmp.Equal(got, want, cmpopts.EquateEmpty(), cmpopts.EquateErrors()) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateErrors()))
			}

			t.Logf("%s (flipped)", name)
			wantFlipped := make([]conflict, 0)
			for _, c := range test.out {
				wantFlipped = append(wantFlipped, conflict{
					First:  c.Second,
					Second: c.First,
					Errs:   c.Errs,
				})
			}
			if got, want := CheckConflicts(
				test.in.Second, test.in.First,
			), wantFlipped; !cmp.Equal(got, want, cmpopts.EquateEmpty(), cmpopts.EquateErrors()) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateErrors()))
			}
		})
	}
}
