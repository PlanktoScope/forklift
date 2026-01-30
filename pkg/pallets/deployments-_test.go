package pallets

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var checkFeatureFlagsWithoutTests = map[string]struct {
	in     FeatureFlags
	remove FeatureFlags
	result FeatureFlags
}{
	"[] []": {},
	"[a] []": {
		in:     FeatureFlags{"a"},
		result: FeatureFlags{"a"},
	},
	"[] [a]": {
		remove: FeatureFlags{"a"},
	},
	"[a] [b]": {
		in:     FeatureFlags{"a"},
		remove: FeatureFlags{"b"},
		result: FeatureFlags{"a"},
	},
	"[a a] [b]": {
		in:     FeatureFlags{"a", "a"},
		remove: FeatureFlags{"b"},
		result: FeatureFlags{"a", "a"}, // i.e. duplicates of "a" are preserved
	},
	"[a a] [a]": {
		in:     FeatureFlags{"a", "a"},
		remove: FeatureFlags{"a"},
	},
	"[a a b] [a]": {
		in:     FeatureFlags{"a", "a", "b"},
		remove: FeatureFlags{"a"},
		result: FeatureFlags{"b"}, // i.e. all copies of "a" are removed
	},
	"[c a b] [a]": {
		in:     FeatureFlags{"c", "a", "b"},
		remove: FeatureFlags{"a"},
		result: FeatureFlags{"c", "b"}, // i.e. no reordering
	},
}

func TestFeatureFlagsWithout(t *testing.T) {
	t.Parallel()
	for name, test := range checkFeatureFlagsWithoutTests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			t.Log(name)
			ff := slices.Clone(test.in)
			if got, want := test.in.Without(test.remove), test.result; !cmp.Equal(
				got, want, cmpopts.EquateEmpty(),
			) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}

			t.Logf("%s (unmodified)", name)
			if got, want := test.in, ff; !cmp.Equal(
				got, want, cmpopts.EquateEmpty(),
			) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}
		})
	}
}

var checkFeatureFlagsWithTests = map[string]struct {
	in           FeatureFlags
	add          FeatureFlags
	allowed      FeatureFlags
	result       FeatureFlags
	unrecognized FeatureFlags
}{
	"[] [] []": {},
	"[a] [] []": {
		in:     FeatureFlags{"a"},
		result: FeatureFlags{"a"},
	},
	"[] [a] []": {
		add:          FeatureFlags{"a"},
		result:       FeatureFlags{"a"},
		unrecognized: FeatureFlags{"a"},
	},
	"[] [a] [a]": {
		add:     FeatureFlags{"a"},
		allowed: FeatureFlags{"a"},
		result:  FeatureFlags{"a"},
	},
	"[a] [b] []": {
		in:           FeatureFlags{"a"},
		add:          FeatureFlags{"b"},
		result:       FeatureFlags{"a", "b"},
		unrecognized: FeatureFlags{"b"},
	},
	"[a] [b] [a]": {
		in:           FeatureFlags{"a"},
		add:          FeatureFlags{"b"},
		allowed:      FeatureFlags{"a"},
		result:       FeatureFlags{"a", "b"},
		unrecognized: FeatureFlags{"b"},
	},
	"[a] [b] [b]": {
		in:      FeatureFlags{"a"},
		add:     FeatureFlags{"b"},
		allowed: FeatureFlags{"b"},
		result:  FeatureFlags{"a", "b"},
	},
	"[b] [a] [a]": {
		in:      FeatureFlags{"b"},
		add:     FeatureFlags{"a"},
		allowed: FeatureFlags{"a"},
		result:  FeatureFlags{"b", "a"},
	},
	"[b] [a] [a a]": {
		in:      FeatureFlags{"b"},
		add:     FeatureFlags{"a"},
		allowed: FeatureFlags{"a", "a"},
		result:  FeatureFlags{"b", "a"},
	},
	"[b] [a] [b]": {
		in:           FeatureFlags{"b"},
		add:          FeatureFlags{"a"},
		allowed:      FeatureFlags{"b"},
		result:       FeatureFlags{"b", "a"},
		unrecognized: FeatureFlags{"a"},
	},
	"[a a] [b] [a]": {
		in:           FeatureFlags{"a", "a"},
		add:          FeatureFlags{"b"},
		allowed:      FeatureFlags{"a"},
		result:       FeatureFlags{"a", "a", "b"}, // i.e. duplicates of "a" are preserved
		unrecognized: FeatureFlags{"b"},
	},
	"[a a] [b] [b]": {
		in:      FeatureFlags{"a", "a"},
		add:     FeatureFlags{"b"},
		allowed: FeatureFlags{"b"},
		result:  FeatureFlags{"a", "a", "b"}, // i.e. duplicates of "a" are preserved
	},
	"[a a] [a] [a]": {
		in:      FeatureFlags{"a", "a"},
		add:     FeatureFlags{"a"},
		allowed: FeatureFlags{"a"},
		result:  FeatureFlags{"a", "a"}, // i.e. no extra duplicates of "a" added
	},
	"[a a] [a] [b]": {
		in:           FeatureFlags{"a", "a"},
		add:          FeatureFlags{"a"},
		allowed:      FeatureFlags{"b"},
		result:       FeatureFlags{"a", "a"},
		unrecognized: FeatureFlags{"a"},
	},
	"[a a b] [a] [a]": {
		in:      FeatureFlags{"a", "a", "b"},
		add:     FeatureFlags{"a"},
		allowed: FeatureFlags{"a"},
		result:  FeatureFlags{"a", "a", "b"}, // i.e. all original duplicates of "a" are preserved
	},
	"[c a b] [a] [a]": {
		in:      FeatureFlags{"c", "a", "b"},
		add:     FeatureFlags{"a"},
		allowed: FeatureFlags{"a"},
		result:  FeatureFlags{"c", "a", "b"}, // i.e. no reordering
	},
	"[c a b] [a a] []": {
		in:           FeatureFlags{"c", "a", "b"},
		add:          FeatureFlags{"a", "a"},
		result:       FeatureFlags{"c", "a", "b"}, // i.e. no reordering, no duplicates added
		unrecognized: FeatureFlags{"a", "a"},      // i.e. duplicates are preserved
	},
}

func TestFeatureFlagsWith(t *testing.T) {
	t.Parallel()
	for name, test := range checkFeatureFlagsWithTests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			t.Logf("%s (result)", name)
			ff := slices.Clone(test.in)
			result, unrecognized := test.in.With(test.add, test.allowed)
			if got, want := result, test.result; !cmp.Equal(
				got, want, cmpopts.EquateEmpty(),
			) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}
			t.Logf("%s (unrecognized)", name)
			if got, want := unrecognized, test.unrecognized; !cmp.Equal(
				got, want, cmpopts.EquateEmpty(),
			) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}

			t.Logf("%s (unmodified)", name)
			if got, want := test.in, ff; !cmp.Equal(
				got, want, cmpopts.EquateEmpty(),
			) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}
		})
	}
}
