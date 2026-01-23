package cli

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/mod/semver"

	"github.com/forklift-run/forklift/internal/app/forklift"
)

// Pallets

type Versions struct {
	Tool               string
	MinSupportedPallet string
}

// CheckPltCompat returns an error upon any version compatibility errors between a pallet
// and - unless the ignoreTool flag is set - the Forklift tool (as specified by toolVersions). Note
// that minimum versions are still enforced even if the ignoreTool flag is set.
func CheckPltCompat(
	pallet *forklift.FSPallet, toolVersions Versions, ignoreTool bool,
) error {
	if ignoreTool {
		fmt.Fprintf(
			os.Stderr, "Warning: ignoring the tool's version (%s) for version compatibility checking!\n",
			toolVersions.Tool,
		)
	}

	if err := CheckArtifactCompat(
		pallet.Def.ForkliftVersion, toolVersions.Tool, toolVersions.MinSupportedPallet, pallet.Path(),
		ignoreTool,
	); err != nil {
		return errors.Wrapf(
			err, "forklift tool has a version incompatibility with pallet %s", pallet.Path(),
		)
	}
	return nil
}

// CheckArtifactCompat determines whether the version of Forklift required by an artifact (a
// repo or pallet), as declared by that artifact's Forklift version, is compatible with the actual
// version of the Forklift tool, and whether the artifact's Forklift version is compatible with the
// tool's minimum supported Forklift version for artifacts.  compatErr is non-nil when the versions
// fail the compatibility check, while checkErr is non-nil when any specified version is invalid.
func CheckArtifactCompat(
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

// CheckDeepCompat returns an error upon any version compatibility errors between a pallet, its
// required pallets & repos (as loaded by repoLoader), and - unless the ignoreTool flag is set - the
// Forklift tool (as specified by toolVersions). Note that minimum versions are still enforced even
// if the ignoreTool flag is set.
func CheckDeepCompat(
	pallet *forklift.FSPallet, palletLoader forklift.FSPalletLoader, repoLoader forklift.FSRepoLoader,
	toolVersions Versions, ignoreTool bool,
) error {
	if ignoreTool {
		fmt.Fprintf(
			os.Stderr, "Warning: ignoring the tool's version (%s) for version compatibility checking!\n",
			toolVersions.Tool,
		)
	}
	// FIXME: merge the pallet with file imports from its required pallets

	if err := CheckArtifactCompat(
		pallet.Def.ForkliftVersion, toolVersions.Tool, toolVersions.MinSupportedPallet, pallet.Path(),
		ignoreTool,
	); err != nil {
		return errors.Wrapf(
			err, "forklift tool has a version incompatibility with pallet %s", pallet.Path(),
		)
	}

	if err := checkReqPalletVersions(pallet, palletLoader, toolVersions, ignoreTool); err != nil {
		return err
	}

	return nil
}

func checkReqPalletVersions(
	pallet *forklift.FSPallet, palletLoader forklift.FSPalletLoader,
	toolVersions Versions, ignoreTool bool,
) error {
	versions, err := loadPalletReqForkliftVersions(pallet, palletLoader)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't determine Forklift versions of pallet %s's pallet requirements", pallet.Path(),
		)
	}
	if err = checkVersionConsistency(pallet.Def.ForkliftVersion, versions); err != nil {
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
	pallet *forklift.FSPallet, palletLoader forklift.FSPalletLoader,
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
			forkliftVersion: fsPallet.Pallet.Def.ForkliftVersion,
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

// Bundles

func CheckBundleShallowCompat(
	bundle *forklift.FSBundle, toolVersion, bundleMinVersion string, ignoreTool bool,
) error {
	if ignoreTool {
		fmt.Fprintf(
			os.Stderr, "Warning: ignoring the tool's version (%s) for version compatibility checking!\n",
			toolVersion,
		)
	}

	if err := CheckArtifactCompat(
		bundle.Manifest.ForkliftVersion, toolVersion, bundleMinVersion, bundle.Path(),
		ignoreTool,
	); err != nil {
		return errors.Wrapf(
			err, "forklift tool has a version incompatibility with staged pallet bundle %s",
			bundle.Path(),
		)
	}
	return nil
}
