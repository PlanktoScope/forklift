package forklift

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	fbun "github.com/forklift-run/forklift/exp/bundling"
)

func CheckBundleShallowCompat(
	bundle *fbun.FSBundle, toolVersion, bundleMinVersion string, ignoreTool bool,
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
