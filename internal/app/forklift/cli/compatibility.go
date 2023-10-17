package cli

import (
	"fmt"

	"github.com/pkg/errors"
	"golang.org/x/mod/semver"
)

// CheckCompatibility determines whether the version of Forklift required by an artifact (a repo or
// pallet), as declared by that artifact's Forklift version, is compatible with the actual version
// of the Forklift tool, and whether the artifact's Forklift version is compatible with the tool's
// minimum supported Forklift version for artifacts.  compatErr is non-nil when the versions fail
// the compatibility check, while checkErr is non-nil when any specified version is invalid.
func CheckCompatibility(
	artifactVersion, toolVersion, minArtifactVersion, artifactPath string, ignoreTool bool,
) error {
	if artifactVersion == "" { // special case for pre-v0.4.0 pallets/repos
		return errors.Errorf(
			"%s doesn't specify a Forklift version (so it probably requires something below v0.4.0)",
			artifactPath,
		)
	}

	if ignoreTool {
		fmt.Printf(
			"Warning: ignoring the tool's version (%s) for version compatibility checking!\n",
			toolVersion,
		)
	} else if semver.Compare(toolVersion, artifactVersion) < 0 {
		return errors.Errorf(
			"the tool's version is %s, but %s requires at least %s",
			toolVersion, artifactPath, artifactVersion,
		)
	}
	if semver.Compare(artifactVersion, minArtifactVersion) < 0 {
		return errors.Errorf(
			"%s version is %s, but the tool requires at least %s",
			artifactPath, artifactVersion, minArtifactVersion,
		)
	}
	return nil
}
