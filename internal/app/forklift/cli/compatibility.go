package cli

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/forklift-run/forklift/internal/app/forklift"
	fbun "github.com/forklift-run/forklift/pkg/bundling"
	fplt "github.com/forklift-run/forklift/pkg/pallets"
)

// Pallets

// CheckPltCompat returns an error upon any version compatibility errors between a pallet
// and - unless the ignoreTool flag is set - the Forklift tool (as specified by toolVersions). Note
// that minimum versions are still enforced even if the ignoreTool flag is set.
func CheckPltCompat(
	pallet *fplt.FSPallet, toolVersions forklift.Versions, ignoreTool bool,
) error {
	if ignoreTool {
		fmt.Fprintf(
			os.Stderr, "Warning: ignoring the tool's version (%s) for version compatibility checking!\n",
			toolVersions.Tool,
		)
	}

	if err := forklift.CheckArtifactCompat(
		pallet.Decl.ForkliftVersion, toolVersions.Tool, toolVersions.MinSupportedPallet, pallet.Path(),
		ignoreTool,
	); err != nil {
		return errors.Wrapf(
			err, "forklift tool has a version incompatibility with pallet %s", pallet.Path(),
		)
	}
	return nil
}

// CheckDeepCompat returns an error upon any version compatibility errors between a pallet, its
// required pallets (as loaded by palletLoader), and - unless the ignoreTool flag is set - the
// Forklift tool (as specified by toolVersions). Note that minimum versions are still enforced even
// if the ignoreTool flag is set.
func CheckDeepCompat(
	pallet *fplt.FSPallet, palletLoader fplt.FSPalletLoader,
	toolVersions forklift.Versions, ignoreTool bool,
) error {
	if ignoreTool {
		fmt.Fprintf(
			os.Stderr, "Warning: ignoring the tool's version (%s) for version compatibility checking!\n",
			toolVersions.Tool,
		)
	}

	if err := forklift.CheckArtifactCompat(
		pallet.Decl.ForkliftVersion, toolVersions.Tool, toolVersions.MinSupportedPallet, pallet.Path(),
		ignoreTool,
	); err != nil {
		return errors.Wrapf(
			err, "forklift tool has a version incompatibility with pallet %s", pallet.Path(),
		)
	}

	if err := forklift.CheckReqPalletVersions(
		pallet, palletLoader, toolVersions, ignoreTool,
	); err != nil {
		return err
	}

	return nil
}

// Bundles

func CheckBundleShallowCompat(
	bundle *fbun.FSBundle, toolVersion, bundleMinVersion string, ignoreTool bool,
) error {
	if ignoreTool {
		fmt.Fprintf(
			os.Stderr, "Warning: ignoring the tool's version (%s) for version compatibility checking!\n",
			toolVersion,
		)
	}

	if err := forklift.CheckArtifactCompat(
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
