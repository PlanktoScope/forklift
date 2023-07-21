// Package dev handles tasks and workflows for developing and maintaining Forklift environments and
// Pallet repositories and packages
package dev

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// TODO: this should return an FSEnv instead
func FindParentEnv(cwd string) (string, error) {
	envCandidatePath, err := filepath.Abs(cwd)
	if err != nil {
		return "", errors.Wrapf(err, "couldn't convert '%s' into an absolute path", cwd)
	}
	for envCandidatePath != "." && envCandidatePath != "/" {
		f := os.DirFS(envCandidatePath)
		_, err := fs.ReadFile(f, "forklift-env.yml")
		if err == nil {
			return envCandidatePath, nil
		}
		envCandidatePath = filepath.Dir(envCandidatePath)
	}
	return "", errors.Errorf(
		"no environment config file found in any parent directory of %s", cwd,
	)
}
