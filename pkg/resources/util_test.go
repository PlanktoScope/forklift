package resources

import "fmt"

type Error string

func (e Error) Error() string {
	return string(e)
}

type res struct {
	Addr string
	Port int
}

func (r res) CheckConflict(candidate res) []error {
	errs := make([]error, 0)
	if r.Addr == candidate.Addr {
		errs = append(errs, Error(fmt.Sprintf("same addr: %s", r.Addr)))
	}
	if r.Port == candidate.Port {
		errs = append(errs, Error(fmt.Sprintf("same port: %d", r.Port)))
	}
	return errs
}

func (r res) CheckDep(candidate res) []error {
	errs := make([]error, 0)
	if r.Addr != candidate.Addr {
		errs = append(errs, Error(fmt.Sprintf("diff addr: %s", r.Addr)))
	}
	if r.Port != candidate.Port {
		errs = append(errs, Error(fmt.Sprintf("diff port: %d", r.Port)))
	}
	return errs
}

type (
	attached     = Attached[res, string]
	conflict     = Conflict[res, string]
	satisfiedDep = SatisfiedDep[res, string]
	missingDep   = MissingDep[res, string]
	depCandidate = DepCandidate[res, string]
)

var (
	resA1 = res{Addr: "a", Port: 1}
	resA2 = res{Addr: "a", Port: 2}
	resB1 = res{Addr: "b", Port: 1}
	resB2 = res{Addr: "b", Port: 2}
	resC2 = res{Addr: "c", Port: 2}
	resC3 = res{Addr: "c", Port: 3}
)

const (
	pkg1 = "pkg 1"
	pkg2 = "pkg 2"
	pkg3 = "pkg 3"
)

var (
	errSameA = Error("same addr: a")
	errSame1 = Error("same port: 1")
	errSame2 = Error("same port: 2")
	errDiffA = Error("diff addr: a")
	errDiffB = Error("diff addr: b")
	errDiffC = Error("diff addr: c")
	errDiff1 = Error("diff port: 1")
	errDiff2 = Error("diff port: 2")
)
