package cli

import (
	"fmt"

	"github.com/pkg/errors"
	"golang.org/x/mod/semver"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
)

// CheckCompatibility returns an error upon any version compatibility errors between a pallet, its
// required pallets & repos (as loaded by repoLoader), and - unless the ignoreTool flag is set - the
// Forklift tool (whose version is specified as toolVersion, and whose minimum compatible pallet
// version is specified as minArtifactVersion).
func CheckCompatibility(
	pallet *forklift.FSPallet, repoLoader forklift.FSRepoLoader,
	toolVersion, minArtifactVersion string, ignoreTool bool,
) error {
	if ignoreTool {
		fmt.Printf(
			"Warning: ignoring the tool's version (%s) for version compatibility checking!\n",
			toolVersion,
		)
	}

	if err := checkArtifactCompatibility(
		pallet.Def.ForkliftVersion, toolVersion, minArtifactVersion, pallet.Path(), ignoreTool,
	); err != nil {
		return errors.Wrapf(
			err, "forklift tool has a version incompatibility with pallet %s", pallet.Path(),
		)
	}
	versions, err := loadReqForkliftVersions(pallet, repoLoader)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't determine Forklift versions of pallet %s's requirements", pallet.Path(),
		)
	}
	if err = checkVersionConsistency(pallet.Def.ForkliftVersion, versions); err != nil {
		return errors.Wrapf(
			err, "pallet %s has a version incompatibility with one of its requirements", pallet.Path(),
		)
	}
	for _, v := range versions {
		if err := checkArtifactCompatibility(
			v.forkliftVersion, toolVersion, minArtifactVersion, v.reqPath+"@"+v.reqVersion, ignoreTool,
		); err != nil {
			return errors.Wrapf(
				err, "forklift tool has a version incompatibility with pallet requirement %s", v.reqPath,
			)
		}
	}
	return nil
}

// checkArtifactCompatibility determines whether the version of Forklift required by an artifact (a
// repo or pallet), as declared by that artifact's Forklift version, is compatible with the actual
// version of the Forklift tool, and whether the artifact's Forklift version is compatible with the
// tool's minimum supported Forklift version for artifacts.  compatErr is non-nil when the versions
// fail the compatibility check, while checkErr is non-nil when any specified version is invalid.
func checkArtifactCompatibility(
	artifactVersion, toolVersion, minArtifactVersion, artifactPath string, ignoreTool bool,
) error {
	if artifactVersion == "" { // special case for pre-v0.4.0 pallets/repos
		return errors.Errorf(
			"%s doesn't specify a Forklift version (so it probably requires something below v0.4.0)",
			artifactPath,
		)
	}

	if !ignoreTool && semver.Compare(toolVersion, artifactVersion) < 0 {
		return errors.Errorf(
			"the tool's version is %s, but %s requires at least %s",
			toolVersion, artifactPath, artifactVersion,
		)
	}
	if semver.Compare(artifactVersion, minArtifactVersion) < 0 {
		return errors.Errorf(
			"%s's Forklift version is %s, but the tool requires at least %s",
			artifactPath, artifactVersion, minArtifactVersion,
		)
	}
	return nil
}

type reqForkliftVersion struct {
	reqPath         string
	reqVersion      string
	forkliftVersion string
}

func loadReqForkliftVersions(
	pallet *forklift.FSPallet, repoLoader forklift.FSRepoLoader,
) ([]reqForkliftVersion, error) {
	repoReqs, err := pallet.LoadFSRepoReqs("**")
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load repo requirements")
	}
	versions := make([]reqForkliftVersion, 0, len(repoReqs))
	for _, req := range repoReqs {
		fsRepo, err := repoLoader.LoadFSRepo(req.Path(), req.VersionLock.Version)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load repo %s@%s", req.Path(), req.VersionLock.Version)
		}
		versions = append(versions, reqForkliftVersion{
			reqPath:         req.Path(),
			reqVersion:      req.VersionLock.Version,
			forkliftVersion: fsRepo.Repo.Def.ForkliftVersion,
		})
	}
	return versions, nil
}

func checkVersionConsistency(
	palletForkliftVersion string, reqForkliftVersions []reqForkliftVersion,
) error {
	for _, v := range reqForkliftVersions {
		if semver.Compare(palletForkliftVersion, v.forkliftVersion) < 0 {
			return errors.Errorf(
				"the pallet's requirements cannot have Forklift versions above %s, but requirement %s has "+
					"Forklift version %s",
				palletForkliftVersion, v.reqPath, v.forkliftVersion,
			)
		}
	}
	return nil
}
