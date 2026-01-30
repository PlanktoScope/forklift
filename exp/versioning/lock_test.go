package versioning

import (
	"errors"
	"fmt"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/google/go-cmp/cmp"
)

var lockDeclTests = map[string]struct {
	in               LockDecl
	shortCommit      string
	parsedVersion    semver.Version
	parsedVersionErr error
	pseudoversion    string
	pseudoversionErr error
	version          string
	versionErr       error
}{
	"version-prerelease": {
		in: LockDecl{
			Type:      "version",
			Tag:       "v2024.0.0-beta.2",
			Timestamp: "20240919055824",
			Commit:    "abba80631ed76ba6a12a2dcdb5db56c901e45bba",
		},
		shortCommit: "abba80631ed7",
		parsedVersion: semver.Version{
			Major: 2024,
			Pre:   []semver.PRVersion{{VersionStr: "beta"}, {VersionNum: 2, IsNum: true}},
		},
		pseudoversion: "v2024.0.0-beta.2.0.20240919055824-abba80631ed7",
		version:       "v2024.0.0-beta.2",
	},
	"pseudoversion-untagged": {
		in: LockDecl{
			Type:      "pseudoversion",
			Timestamp: "20250325143148",
			Commit:    "7a60c3c39e25830aa249d9348a5c9717ed407876",
		},
		parsedVersionErr: errors.New("invalid tag `` doesn't start with `v`"),
		shortCommit:      "7a60c3c39e25",
		pseudoversion:    "v0.0.0-20250325143148-7a60c3c39e25",
		version:          "v0.0.0-20250325143148-7a60c3c39e25",
	},
	"pseudoversion-tagged": {
		in: LockDecl{
			Type:      "pseudoversion",
			Tag:       "v2024.0.0",
			Timestamp: "20250414231313",
			Commit:    "8fd23fded73b71d391d47a14ca79684cfcb6aeb7",
		},
		shortCommit:   "8fd23fded73b",
		parsedVersion: semver.Version{Major: 2024},
		pseudoversion: "v2024.0.1-0.20250414231313-8fd23fded73b",
		version:       "v2024.0.1-0.20250414231313-8fd23fded73b",
	},
	"pseudoversion-tagged-prerelease": {
		in: LockDecl{
			Type:      "pseudoversion",
			Tag:       "v2025.0.0-alpha.0",
			Timestamp: "20250702170448",
			Commit:    "d6b96488a5c4d8520135c66bd888fc0e933f321e",
		},
		shortCommit: "d6b96488a5c4",
		parsedVersion: semver.Version{
			Major: 2025,
			Pre:   []semver.PRVersion{{VersionStr: "alpha"}, {VersionNum: 0, IsNum: true}},
		},
		pseudoversion: "v2025.0.0-alpha.0.0.20250702170448-d6b96488a5c4",
		version:       "v2025.0.0-alpha.0.0.20250702170448-d6b96488a5c4",
	},
}

func TestLockDecl(t *testing.T) {
	t.Parallel()
	for name, test := range lockDeclTests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			t.Logf("%s (short-commit)", name)
			if got, want := test.in.ShortCommit(), test.shortCommit; !cmp.Equal(got, want) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}

			t.Logf("%s (parse-version)", name)
			parsedVersion, err := test.in.ParseVersion()
			if got, want := fmt.Sprintf("%s", err), fmt.Sprintf("%s", test.parsedVersionErr); !cmp.Equal(
				got, want,
			) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}
			if got, want := parsedVersion, test.parsedVersion; !cmp.Equal(got, want) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}

			t.Logf("%s (pseudoversion)", name)
			pseudoversion, err := test.in.Pseudoversion()
			if got, want := fmt.Sprintf("%s", err), fmt.Sprintf("%s", test.pseudoversionErr); !cmp.Equal(
				got, want,
			) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}
			if got, want := pseudoversion, test.pseudoversion; !cmp.Equal(got, want) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}

			t.Logf("%s (version)", name)
			version, err := test.in.Version()
			if got, want := fmt.Sprintf("%s", err), fmt.Sprintf("%s", test.versionErr); !cmp.Equal(
				got, want,
			) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}
			if got, want := version, test.version; !cmp.Equal(got, want) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}
		})
	}
}

var lockDeclInvalidTests = map[string]struct {
	in               LockDecl
	parsedVersion    semver.Version
	parsedVersionErr error // TODO: test
	pseudoversion    string
	pseudoversionErr error // TODO: test
	version          string
	versionErr       error // TODO: test
}{
	"untyped": {
		in: LockDecl{
			Tag:       "v2024.0.0",
			Timestamp: "20250414231313",
			Commit:    "8fd23fded73b71d391d47a14ca79684cfcb6aeb7",
		},
		parsedVersion: semver.Version{Major: 2024},
		pseudoversion: "v2024.0.1-0.20250414231313-8fd23fded73b",
		versionErr:    errors.New("unknown lock type "),
	},
	"unparsable version": {
		in: LockDecl{
			Type:      "version",
			Tag:       "v2024-0-0_beta-2",
			Timestamp: "20240919055824",
			Commit:    "abba80631ed76ba6a12a2dcdb5db56c901e45bba",
		},
		parsedVersionErr: errors.New("tag `v2024-0-0_beta-2` couldn't be parsed as a semantic version"),
		pseudoversionErr: errors.New("tag `v2024-0-0_beta-2` couldn't be parsed as a semantic version"),
		versionErr: errors.New(
			"invalid version: tag `v2024-0-0_beta-2` couldn't be parsed as a semantic version",
		),
	},
	"pseudoversion without timestamp": {
		in: LockDecl{
			Type:   "pseudoversion",
			Tag:    "v2024.0.0",
			Commit: "8fd23fded73b71d391d47a14ca79684cfcb6aeb7",
		},
		parsedVersion:    semver.Version{Major: 2024},
		pseudoversionErr: errors.New("pseudoversion missing commit timestamp"),
		versionErr: errors.New(
			"couldn't determine pseudo-version: pseudoversion missing commit timestamp",
		),
	},
	"pseudoversion without commit hash": {
		in: LockDecl{
			Type:      "pseudoversion",
			Tag:       "v2024.0.0",
			Timestamp: "20250325143148",
		},
		parsedVersion:    semver.Version{Major: 2024},
		pseudoversionErr: errors.New("pseudoversion missing commit hash"),
		versionErr: errors.New(
			"couldn't determine pseudo-version: pseudoversion missing commit hash",
		),
	},
}

func TestLockDeclInvalid(t *testing.T) {
	t.Parallel()
	for name, test := range lockDeclInvalidTests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			t.Logf("%s (parse-version)", name)
			parsedVersion, err := test.in.ParseVersion()
			if got, want := fmt.Sprintf("%s", err), fmt.Sprintf("%s", test.parsedVersionErr); !cmp.Equal(
				got, want,
			) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}
			if got, want := parsedVersion, test.parsedVersion; !cmp.Equal(got, want) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}

			t.Logf("%s (pseudoversion)", name)
			pseudoversion, err := test.in.Pseudoversion()
			if got, want := fmt.Sprintf("%s", err), fmt.Sprintf("%s", test.pseudoversionErr); !cmp.Equal(
				got, want,
			) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}
			if got, want := pseudoversion, test.pseudoversion; !cmp.Equal(got, want) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}

			t.Logf("%s (version)", name)
			version, err := test.in.Version()
			if got, want := fmt.Sprintf("%s", err), fmt.Sprintf("%s", test.versionErr); !cmp.Equal(
				got, want,
			) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}
			if got, want := version, test.version; !cmp.Equal(got, want) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got))
			}
		})
	}
}
