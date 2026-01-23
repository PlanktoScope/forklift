package resources

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type checkDepsIn struct {
	Required []attached
	Provided []attached
}

type checkDepsOut struct {
	Satisfied []satisfiedDep
	Missing   []missingDep
}

var checkDepsTests = map[string]struct {
	in  checkDepsIn
	out checkDepsOut
}{
	"required-x0 provided-x0 | satisfied-x0 missing-x0": {},
	"required-x0 provided-x1 | satisfied-x0 missing-x0": {
		in: checkDepsIn{
			Provided: []attached{{Res: resA1, Origin: pkg1}},
		},
	},
	"required-x1 provided-x0 | satisfied-x0 missing-x1": {
		in: checkDepsIn{
			Required: []attached{{Res: resA1, Origin: pkg1}},
		},
		out: checkDepsOut{
			Missing: []missingDep{{
				Required: attached{Res: resA1, Origin: pkg1},
			}},
		},
	},
	"required-x1 provided-x1 | satisfied-x0 missing-x1": {
		in: checkDepsIn{
			Required: []attached{{Res: resA1, Origin: pkg1}},
			Provided: []attached{{Res: resA2, Origin: pkg2}},
		},
		out: checkDepsOut{
			Missing: []missingDep{{
				Required: attached{Res: resA1, Origin: pkg1},
				BestCandidates: []depCandidate{{
					Provided: attached{Res: resA2, Origin: pkg2},
					Errs:     []error{errDiff1},
				}},
			}},
		},
	},
	"required-x1 provided-x1 | satisfied-x0 missing-x2": {
		in: checkDepsIn{
			Required: []attached{{Res: resA1, Origin: pkg1}},
			Provided: []attached{{Res: resB2, Origin: pkg2}},
		},
		out: checkDepsOut{
			Missing: []missingDep{{
				Required: attached{Res: resA1, Origin: pkg1},
				BestCandidates: []depCandidate{{
					Provided: attached{Res: resB2, Origin: pkg2},
					Errs:     []error{errDiffA, errDiff1},
				}},
			}},
		},
	},
	"required-x1 provided-x1 | satisfied-x1 missing-x0": {
		in: checkDepsIn{
			Required: []attached{{Res: resA1, Origin: pkg1}},
			Provided: []attached{{Res: resA1, Origin: pkg2}},
		},
		out: checkDepsOut{
			Satisfied: []satisfiedDep{{
				Required: attached{Res: resA1, Origin: pkg1},
				Provided: attached{Res: resA1, Origin: pkg2},
			}},
		},
	},
	"required-x1 provided-x2 | satisfied-x0 missing-x1": {
		in: checkDepsIn{
			Required: []attached{{Res: resA1, Origin: pkg1}},
			Provided: []attached{
				{Res: resB1, Origin: pkg2},
				{Res: resB2, Origin: pkg2},
			},
		},
		out: checkDepsOut{
			Missing: []missingDep{{
				Required: attached{Res: resA1, Origin: pkg1},
				BestCandidates: []depCandidate{{
					Provided: attached{Res: resB1, Origin: pkg2},
					Errs:     []error{errDiffA},
				}},
			}},
		},
	},
	"required-x1 provided-x2 | satisfied-x0 missing-x2": {
		in: checkDepsIn{
			Required: []attached{{Res: resA1, Origin: pkg1}},
			Provided: []attached{
				{Res: resB1, Origin: pkg2},
				{Res: resA2, Origin: pkg2},
			},
		},
		out: checkDepsOut{
			Missing: []missingDep{{
				Required: attached{Res: resA1, Origin: pkg1},
				BestCandidates: []depCandidate{
					{
						Provided: attached{Res: resB1, Origin: pkg2},
						Errs:     []error{errDiffA},
					},
					{
						Provided: attached{Res: resA2, Origin: pkg2},
						Errs:     []error{errDiff1},
					},
				},
			}},
		},
	},
	"required-x1 provided-x2 | satisfied-x1 missing-x0": {
		in: checkDepsIn{
			Required: []attached{{Res: resA1, Origin: pkg1}},
			Provided: []attached{
				{Res: resA1, Origin: pkg2},
				{Res: resA2, Origin: pkg2},
			},
		},
		out: checkDepsOut{
			Satisfied: []satisfiedDep{{
				Required: attached{Res: resA1, Origin: pkg1},
				Provided: attached{Res: resA1, Origin: pkg2},
			}},
		},
	},
	"required-x1 provided-x2-redund | satisfied-x1 missing-x0": {
		in: checkDepsIn{
			Required: []attached{{Res: resA1, Origin: pkg1}},
			Provided: []attached{
				{Res: resA1, Origin: pkg2},
				{Res: resA1, Origin: pkg3},
			},
		},
		out: checkDepsOut{
			Satisfied: []satisfiedDep{{
				Required: attached{Res: resA1, Origin: pkg1},
				Provided: attached{Res: resA1, Origin: pkg2},
			}},
		},
	},
	"required-x2 provided-x0 | satisfied-x0 missing-x2": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA2, Origin: pkg1},
			},
		},
		out: checkDepsOut{
			Missing: []missingDep{
				{Required: attached{Res: resA1, Origin: pkg1}},
				{Required: attached{Res: resA2, Origin: pkg1}},
			},
		},
	},
	"required-x2 provided-x1 | satisfied-x0 missing-x4": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA2, Origin: pkg1},
			},
			Provided: []attached{{Res: resC3, Origin: pkg2}},
		},
		out: checkDepsOut{
			Missing: []missingDep{
				{
					Required: attached{Res: resA1, Origin: pkg1},
					BestCandidates: []depCandidate{{
						Provided: attached{Res: resC3, Origin: pkg2},
						Errs:     []error{errDiffA, errDiff1},
					}},
				},
				{
					Required: attached{Res: resA2, Origin: pkg1},
					BestCandidates: []depCandidate{{
						Provided: attached{Res: resC3, Origin: pkg2},
						Errs:     []error{errDiffA, errDiff2},
					}},
				},
			},
		},
	},
	"required-x2 provided-x1 | satisfied-x0 missing-x3": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA2, Origin: pkg1},
			},
			Provided: []attached{{Res: resB1, Origin: pkg2}},
		},
		out: checkDepsOut{
			Missing: []missingDep{
				{
					Required: attached{Res: resA1, Origin: pkg1},
					BestCandidates: []depCandidate{{
						Provided: attached{Res: resB1, Origin: pkg2},
						Errs:     []error{errDiffA},
					}},
				},
				{
					Required: attached{Res: resA2, Origin: pkg1},
					BestCandidates: []depCandidate{{
						Provided: attached{Res: resB1, Origin: pkg2},
						Errs:     []error{errDiffA, errDiff2},
					}},
				},
			},
		},
	},
	"required-x2 provided-x1 | satisfied-x1 missing-x2": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resB2, Origin: pkg1},
			},
			Provided: []attached{{Res: resA1, Origin: pkg2}},
		},
		out: checkDepsOut{
			Satisfied: []satisfiedDep{{
				Required: attached{Res: resA1, Origin: pkg1},
				Provided: attached{Res: resA1, Origin: pkg2},
			}},
			Missing: []missingDep{
				{
					Required: attached{Res: resB2, Origin: pkg1},
					BestCandidates: []depCandidate{{
						Provided: attached{Res: resA1, Origin: pkg2},
						Errs:     []error{errDiffB, errDiff2},
					}},
				},
			},
		},
	},
	"required-x2 provided-x1 | satisfied-x1 missing-x1": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA2, Origin: pkg1},
			},
			Provided: []attached{{Res: resA1, Origin: pkg2}},
		},
		out: checkDepsOut{
			Satisfied: []satisfiedDep{{
				Required: attached{Res: resA1, Origin: pkg1},
				Provided: attached{Res: resA1, Origin: pkg2},
			}},
			Missing: []missingDep{
				{
					Required: attached{Res: resA2, Origin: pkg1},
					BestCandidates: []depCandidate{{
						Provided: attached{Res: resA1, Origin: pkg2},
						Errs:     []error{errDiff2},
					}},
				},
			},
		},
	},
	"required-x2 provided-x1 | satisfied-x2 missing-x0": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA1, Origin: pkg2},
			},
			Provided: []attached{{Res: resA1, Origin: pkg3}},
		},
		out: checkDepsOut{
			Satisfied: []satisfiedDep{
				{
					Required: attached{Res: resA1, Origin: pkg1},
					Provided: attached{Res: resA1, Origin: pkg3},
				},
				{
					Required: attached{Res: resA1, Origin: pkg2},
					Provided: attached{Res: resA1, Origin: pkg3},
				},
			},
		},
	},
	"required-x2 provided-x2 | satisfied-x0 missing-x8": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resB1, Origin: pkg1},
			},
			Provided: []attached{
				{Res: resC2, Origin: pkg2},
				{Res: resC3, Origin: pkg2},
			},
		},
		out: checkDepsOut{
			Missing: []missingDep{
				{
					Required: attached{Res: resA1, Origin: pkg1},
					BestCandidates: []depCandidate{
						{
							Provided: attached{Res: resC2, Origin: pkg2},
							Errs:     []error{errDiffA, errDiff1},
						},
						{
							Provided: attached{Res: resC3, Origin: pkg2},
							Errs:     []error{errDiffA, errDiff1},
						},
					},
				},
				{
					Required: attached{Res: resB1, Origin: pkg1},
					BestCandidates: []depCandidate{
						{
							Provided: attached{Res: resC2, Origin: pkg2},
							Errs:     []error{errDiffB, errDiff1},
						},
						{
							Provided: attached{Res: resC3, Origin: pkg2},
							Errs:     []error{errDiffB, errDiff1},
						},
					},
				},
			},
		},
	},
	"required-x2 provided-x2 | satisfied-x0 missing-x6": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA2, Origin: pkg1},
			},
			Provided: []attached{
				{Res: resB2, Origin: pkg2},
				{Res: resC3, Origin: pkg2},
			},
		},
		out: checkDepsOut{
			Missing: []missingDep{
				{
					Required: attached{Res: resA1, Origin: pkg1},
					BestCandidates: []depCandidate{
						{
							Provided: attached{Res: resB2, Origin: pkg2},
							Errs:     []error{errDiffA, errDiff1},
						},
						{
							Provided: attached{Res: resC3, Origin: pkg2},
							Errs:     []error{errDiffA, errDiff1},
						},
					},
				},
				{
					Required: attached{Res: resA2, Origin: pkg1},
					BestCandidates: []depCandidate{{
						Provided: attached{Res: resB2, Origin: pkg2},
						Errs:     []error{errDiffA},
					}},
				},
			},
		},
	},
	"required-x2 provided-x2 | satisfied-x0 missing-x5": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resB1, Origin: pkg1},
			},
			Provided: []attached{
				{Res: resB2, Origin: pkg2},
				{Res: resC3, Origin: pkg2},
			},
		},
		out: checkDepsOut{
			Missing: []missingDep{
				{
					Required: attached{Res: resA1, Origin: pkg1},
					BestCandidates: []depCandidate{
						{
							Provided: attached{Res: resB2, Origin: pkg2},
							Errs:     []error{errDiffA, errDiff1},
						},
						{
							Provided: attached{Res: resC3, Origin: pkg2},
							Errs:     []error{errDiffA, errDiff1},
						},
					},
				},
				{
					Required: attached{Res: resB1, Origin: pkg1},
					BestCandidates: []depCandidate{
						{
							Provided: attached{Res: resB2, Origin: pkg2},
							Errs:     []error{errDiff1},
						},
					},
				},
			},
		},
	},
	"required-x2 provided-x2 | satisfied-x0 missing-x4": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resB2, Origin: pkg1},
			},
			Provided: []attached{
				{Res: resB1, Origin: pkg2},
				{Res: resA2, Origin: pkg2},
			},
		},
		out: checkDepsOut{
			Missing: []missingDep{
				{
					Required: attached{Res: resA1, Origin: pkg1},
					BestCandidates: []depCandidate{
						{
							Provided: attached{Res: resB1, Origin: pkg2},
							Errs:     []error{errDiffA},
						},
						{
							Provided: attached{Res: resA2, Origin: pkg2},
							Errs:     []error{errDiff1},
						},
					},
				},
				{
					Required: attached{Res: resB2, Origin: pkg1},
					BestCandidates: []depCandidate{
						{
							Provided: attached{Res: resB1, Origin: pkg2},
							Errs:     []error{errDiff2},
						},
						{
							Provided: attached{Res: resA2, Origin: pkg2},
							Errs:     []error{errDiffB},
						},
					},
				},
			},
		},
	},
	"required-x2 provided-x2 | satisfied-x0 missing-x3": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA2, Origin: pkg1},
				{Res: resC2, Origin: pkg1},
			},
			Provided: []attached{
				{Res: resB2, Origin: pkg2},
				{Res: resC3, Origin: pkg2},
			},
		},
		out: checkDepsOut{
			Missing: []missingDep{
				{
					Required: attached{Res: resA2, Origin: pkg1},
					BestCandidates: []depCandidate{
						{
							Provided: attached{Res: resB2, Origin: pkg2},
							Errs:     []error{errDiffA},
						},
					},
				},
				{
					Required: attached{Res: resC2, Origin: pkg1},
					BestCandidates: []depCandidate{
						{
							Provided: attached{Res: resB2, Origin: pkg2},
							Errs:     []error{errDiffC},
						},
						{
							Provided: attached{Res: resC3, Origin: pkg2},
							Errs:     []error{errDiff2},
						},
					},
				},
			},
		},
	},
	"required-x2 provided-x2 | satisfied-x0 missing-x2": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA2, Origin: pkg1},
			},
			Provided: []attached{
				{Res: resB1, Origin: pkg2},
				{Res: resB2, Origin: pkg2},
			},
		},
		out: checkDepsOut{
			Missing: []missingDep{
				{
					Required: attached{Res: resA1, Origin: pkg1},
					BestCandidates: []depCandidate{
						{
							Provided: attached{Res: resB1, Origin: pkg2},
							Errs:     []error{errDiffA},
						},
					},
				},
				{
					Required: attached{Res: resA2, Origin: pkg1},
					BestCandidates: []depCandidate{{
						Provided: attached{Res: resB2, Origin: pkg2},
						Errs:     []error{errDiffA},
					}},
				},
			},
		},
	},
	"required-x2 provided-x2 | satisfied-x0 missing-x1": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA1, Origin: pkg2},
			},
			Provided: []attached{
				{Res: resB1, Origin: pkg3},
				{Res: resC3, Origin: pkg3},
			},
		},
		out: checkDepsOut{
			Missing: []missingDep{
				{
					Required: attached{Res: resA1, Origin: pkg1},
					BestCandidates: []depCandidate{
						{
							Provided: attached{Res: resB1, Origin: pkg3},
							Errs:     []error{errDiffA},
						},
					},
				},
				{
					Required: attached{Res: resA1, Origin: pkg2},
					BestCandidates: []depCandidate{{
						Provided: attached{Res: resB1, Origin: pkg3},
						Errs:     []error{errDiffA},
					}},
				},
			},
		},
	},
	"required-x2 provided-x2 | satisfied-x1 missing-x4": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resB2, Origin: pkg1},
			},
			Provided: []attached{
				{Res: resA1, Origin: pkg2},
				{Res: resC3, Origin: pkg2},
			},
		},
		out: checkDepsOut{
			Satisfied: []satisfiedDep{{
				Required: attached{Res: resA1, Origin: pkg1},
				Provided: attached{Res: resA1, Origin: pkg2},
			}},
			Missing: []missingDep{
				{
					Required: attached{Res: resB2, Origin: pkg1},
					BestCandidates: []depCandidate{
						{
							Provided: attached{Res: resA1, Origin: pkg2},
							Errs:     []error{errDiffB, errDiff2},
						},
						{
							Provided: attached{Res: resC3, Origin: pkg2},
							Errs:     []error{errDiffB, errDiff2},
						},
					},
				},
			},
		},
	},
	"required-x2 provided-x2 | satisfied-x1 missing-x2": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA2, Origin: pkg1},
			},
			Provided: []attached{
				{Res: resA1, Origin: pkg2},
				{Res: resB2, Origin: pkg2},
			},
		},
		out: checkDepsOut{
			Satisfied: []satisfiedDep{{
				Required: attached{Res: resA1, Origin: pkg1},
				Provided: attached{Res: resA1, Origin: pkg2},
			}},
			Missing: []missingDep{
				{
					Required: attached{Res: resA2, Origin: pkg1},
					BestCandidates: []depCandidate{
						{
							Provided: attached{Res: resA1, Origin: pkg2},
							Errs:     []error{errDiff2},
						},
						{
							Provided: attached{Res: resB2, Origin: pkg2},
							Errs:     []error{errDiffA},
						},
					},
				},
			},
		},
	},
	"required-x2 provided-x2 | satisfied-x1 missing-x1": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA2, Origin: pkg1},
			},
			Provided: []attached{
				{Res: resA1, Origin: pkg2},
				{Res: resC3, Origin: pkg2},
			},
		},
		out: checkDepsOut{
			Satisfied: []satisfiedDep{{
				Required: attached{Res: resA1, Origin: pkg1},
				Provided: attached{Res: resA1, Origin: pkg2},
			}},
			Missing: []missingDep{
				{
					Required: attached{Res: resA2, Origin: pkg1},
					BestCandidates: []depCandidate{{
						Provided: attached{Res: resA1, Origin: pkg2},
						Errs:     []error{errDiff2},
					}},
				},
			},
		},
	},
	"required-x2 provided-x2 | satisfied-x2 missing-x0": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resB2, Origin: pkg1},
			},
			Provided: []attached{
				{Res: resA1, Origin: pkg2},
				{Res: resB2, Origin: pkg2},
			},
		},
		out: checkDepsOut{
			Satisfied: []satisfiedDep{
				{
					Required: attached{Res: resA1, Origin: pkg1},
					Provided: attached{Res: resA1, Origin: pkg2},
				},
				{
					Required: attached{Res: resB2, Origin: pkg1},
					Provided: attached{Res: resB2, Origin: pkg2},
				},
			},
		},
	},
	"required-x2-ident provided-x2 | satisfied-x2 missing-x0": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA1, Origin: pkg2},
			},
			Provided: []attached{
				{Res: resA1, Origin: pkg3},
				{Res: resB2, Origin: pkg3},
			},
		},
		out: checkDepsOut{
			Satisfied: []satisfiedDep{
				{
					Required: attached{Res: resA1, Origin: pkg1},
					Provided: attached{Res: resA1, Origin: pkg3},
				},
				{
					Required: attached{Res: resA1, Origin: pkg2},
					Provided: attached{Res: resA1, Origin: pkg3},
				},
			},
		},
	},
	"required-x2-ident provided-x2-ident | satisfied-x2 missing-x0": {
		in: checkDepsIn{
			Required: []attached{
				{Res: resA1, Origin: pkg1},
				{Res: resA1, Origin: pkg2},
			},
			Provided: []attached{
				{Res: resA1, Origin: pkg2},
				{Res: resA1, Origin: pkg3},
			},
		},
		out: checkDepsOut{
			Satisfied: []satisfiedDep{
				{
					Required: attached{Res: resA1, Origin: pkg1},
					Provided: attached{Res: resA1, Origin: pkg2},
				},
				{
					Required: attached{Res: resA1, Origin: pkg2},
					Provided: attached{Res: resA1, Origin: pkg2},
				},
			},
		},
	},
}

func TestCheckDeps(t *testing.T) {
	t.Parallel()
	for name, test := range checkDepsTests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			t.Log(name)
			want := test.out
			var got checkDepsOut
			if got.Satisfied, got.Missing = CheckDeps(
				test.in.Required, test.in.Provided,
			); !cmp.Equal(got, want, cmpopts.EquateEmpty(), cmpopts.EquateErrors()) {
				t.Errorf("diff (-want +got):\n%+v", cmp.Diff(want, got, cmpopts.EquateErrors()))
			}
		})
	}
}
