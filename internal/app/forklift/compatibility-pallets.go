package forklift

import (
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"

	fplt "github.com/forklift-run/forklift/pkg/pallets"
)

// Pallets

type Versions struct {
	Tool               string
	MinSupportedPallet string
}

// CheckArtifactCompat determines whether the version of Forklift required by an artifact (a
// pallet), as declared by that artifact's Forklift version, is compatible with the actual
// version of the Forklift tool, and whether the artifact's Forklift version is compatible with the
// tool's minimum supported Forklift version for artifacts.  compatErr is non-nil when the versions
// fail the compatibility check, while checkErr is non-nil when any specified version is invalid.
func CheckArtifactCompat(
	artifactVersion, toolVersion, minArtifactVersion, artifactPath string, ignoreTool bool,
) error {
	if artifactVersion == "" { // special case for pre-v0.4.0 pallets
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

func CheckReqPalletVersions(
	pallet *fplt.FSPallet, palletLoader fplt.FSPalletLoader,
	toolVersions Versions, ignoreTool bool,
) error {
	versions, err := loadPalletReqForkliftVersions(pallet, palletLoader)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't determine Forklift versions of pallet %s's pallet requirements", pallet.Path(),
		)
	}
	if err = checkVersionConsistency(pallet.Decl.ForkliftVersion, versions); err != nil {
		return errors.Wrapf(
			err, "pallet %s has a version incompatibility with a required pallet", pallet.Path(),
		)
	}
	for _, v := range versions {
		if err := CheckArtifactCompat(
			v.forkliftVersion, toolVersions.Tool, toolVersions.MinSupportedPallet,
			v.reqPath+"@"+v.reqVersion, ignoreTool,
		); err != nil {
			return errors.Wrapf(
				err, "forklift tool has a version incompatibility with required pallet %s", v.reqPath,
			)
		}
	}
	return nil
}

type reqForkliftVersion struct {
	reqPath         string
	reqVersion      string
	forkliftVersion string
}

func loadPalletReqForkliftVersions(
	pallet *fplt.FSPallet, palletLoader fplt.FSPalletLoader,
) ([]reqForkliftVersion, error) {
	palletReqs, err := pallet.LoadFSPalletReqs("**")
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load pallet requirements")
	}
	versions := make([]reqForkliftVersion, 0, len(palletReqs))
	for _, req := range palletReqs {
		fsPallet, err := palletLoader.LoadFSPallet(req.Path(), req.VersionLock.Version)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't load pallet %s@%s", req.Path(), req.VersionLock.Version,
			)
		}
		versions = append(versions, reqForkliftVersion{
			reqPath:         req.Path(),
			reqVersion:      req.VersionLock.Version,
			forkliftVersion: fsPallet.Pallet.Decl.ForkliftVersion,
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
