package core

import (
	"github.com/pkg/errors"
)

func ErrsWrap(errs []error, message string) []error {
	wrapped := make([]error, 0, len(errs))
	for _, err := range errs {
		wrapped = append(wrapped, errors.Wrap(err, message))
	}
	return wrapped
}

func ErrsWrapf(errs []error, format string, a ...any) []error {
	wrapped := make([]error, 0, len(errs))
	for _, err := range errs {
		wrapped = append(wrapped, errors.Wrapf(err, format, a...))
	}
	return wrapped
}
