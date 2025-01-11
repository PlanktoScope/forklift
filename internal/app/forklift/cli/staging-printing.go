package cli

import (
	"fmt"
	"io"
	"slices"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
)

// Bundles

func FprintStagedBundle(
	indent int, out io.Writer,
	store *forklift.FSStageStore, bundle *forklift.FSBundle, index int, names []string,
) {
	IndentedFprintf(indent, out, "Staged pallet bundle: %d\n", index)
	indent++

	IndentedFprint(indent, out, "Staged names:")
	if len(names) == 0 {
		_, _ = fmt.Fprintln(out, " (none)")
	} else {
		_, _ = fmt.Fprintln(out)
		for _, name := range names {
			BulletedFprintln(indent+1, out, name)
		}
	}

	IndentedFprintln(indent, out, "Pallet:")
	fprintBundlePallet(indent+1, out, bundle.Manifest.Pallet)

	IndentedFprint(indent, out, "Includes:")
	if !bundle.Manifest.Includes.HasInclusions() {
		_, _ = fmt.Fprintln(out, " (none)")
	} else {
		_, _ = fmt.Fprintln(out)
		fprintBundleInclusions(indent+1, out, bundle.Manifest.Includes)
	}

	IndentedFprint(indent, out, "Deploys:")
	if len(bundle.Manifest.Deploys) == 0 {
		_, _ = fmt.Fprintln(out, " (none)")
	} else {
		_, _ = fmt.Fprintln(out)
		fprintBundleDeployments(indent+1, out, bundle.Manifest.Deploys)
	}

	IndentedFprint(indent, out, "Downloads:")
	if len(bundle.Manifest.Downloads.All()) == 0 {
		_, _ = fmt.Fprintln(out, " (none)")
	} else {
		_, _ = fmt.Fprintln(out)
		fprintBundleDownloads(indent+1, out, bundle.Manifest.Downloads)
	}

	IndentedFprint(indent, out, "Exports:")
	if len(bundle.Manifest.Exports) == 0 {
		_, _ = fmt.Fprintln(out, " (none)")
	} else {
		_, _ = fmt.Fprintln(out)
		fprintBundleExports(indent+1, out, bundle.Manifest.Exports)
	}
}

func fprintBundlePallet(indent int, out io.Writer, pallet forklift.BundlePallet) {
	IndentedFprintf(indent, out, "Path: %s\n", pallet.Path)
	IndentedFprintf(indent, out, "Version: %s", pallet.Version)
	if !pallet.Clean {
		_, _ = fmt.Fprint(out, " (includes uncommitted changes)")
	}
	_, _ = fmt.Fprintln(out)
	IndentedFprintf(indent, out, "Description: %s", pallet.Description)
	_, _ = fmt.Fprintln(out)
}

func fprintBundleInclusions(indent int, out io.Writer, inclusions forklift.BundleInclusions) {
	IndentedFprint(indent, out, "Pallets:")
	if len(inclusions.Pallets) == 0 {
		_, _ = fmt.Fprintln(out, " (none)")
	} else {
		_, _ = fmt.Fprintln(out, " (unimplemented)")
		// TODO: implement this once we add support for including pallets
	}
	IndentedFprint(indent, out, "Repos:")
	if len(inclusions.Repos) == 0 {
		_, _ = fmt.Fprintln(out, " (none)")
	} else {
		_, _ = fmt.Fprintln(out)
		sortedPaths := make([]string, 0, len(inclusions.Repos))
		for path := range inclusions.Repos {
			sortedPaths = append(sortedPaths, path)
		}
		slices.Sort(sortedPaths)
		for _, path := range sortedPaths {
			fprintBundleRepoInclusion(indent+1, out, path, inclusions.Repos[path])
		}
	}
}

func fprintBundleRepoInclusion(
	indent int, out io.Writer, path string, inclusion forklift.BundleRepoInclusion,
) {
	IndentedFprintf(indent, out, "%s:\n", path)
	indent++
	IndentedFprint(indent, out, "Required version")
	if inclusion.Override == (forklift.BundleInclusionOverride{}) {
		_, _ = fmt.Fprint(out, " (overridden)")
	}
	_, _ = fmt.Fprint(out, ": ")
	_, _ = fmt.Fprintln(out, inclusion.Req.VersionLock.Version)

	if inclusion.Override == (forklift.BundleInclusionOverride{}) {
		return
	}
	IndentedFprintln(indent, out, "Override:")
	IndentedFprintf(indent+1, out, "Path: %s\n", inclusion.Override.Path)
	IndentedFprint(indent+1, out, "Version: ")
	if inclusion.Override.Version == "" {
		_, _ = fmt.Fprint(out, "(unknown)")
	} else {
		_, _ = fmt.Fprint(out, inclusion.Override.Version)
	}
	if !inclusion.Override.Clean {
		_, _ = fmt.Fprint(out, " (includes uncommitted changes)")
	}
	_, _ = fmt.Fprintln(out)
}

func fprintBundleDeployments(indent int, out io.Writer, deployments map[string]forklift.DeplDef) {
	sortedDeplNames := make([]string, 0, len(deployments))
	for deplName := range deployments {
		sortedDeplNames = append(sortedDeplNames, deplName)
	}
	slices.Sort(sortedDeplNames)
	for _, deplName := range sortedDeplNames {
		IndentedFprintf(indent, out, "%s: %s\n", deplName, deployments[deplName].Package)
	}
}

func fprintBundleDownloads(indent int, out io.Writer, downloads forklift.BundleDownloads) {
	IndentedFprint(indent, out, "HTTP files:")
	if len(downloads.HTTPFile) == 0 {
		_, _ = fmt.Fprintln(out, " (none)")
	} else {
		_, _ = fmt.Fprintln(out)
		fprintBundleTypedDownloads(indent+1, out, downloads.HTTPFile)
	}

	IndentedFprint(indent, out, "OCI images:")
	if len(downloads.OCIImage) == 0 {
		_, _ = fmt.Fprintln(out, " (none)")
	} else {
		_, _ = fmt.Fprintln(out)
		fprintBundleTypedDownloads(indent+1, out, downloads.OCIImage)
	}
}

func fprintBundleTypedDownloads(indent int, out io.Writer, downloads map[string][]string) {
	sortedDeplNames := make([]string, 0, len(downloads))
	for deplName := range downloads {
		sortedDeplNames = append(sortedDeplNames, deplName)
	}
	slices.Sort(sortedDeplNames)
	for _, deplName := range sortedDeplNames {
		if len(downloads[deplName]) == 0 {
			continue
		}
		IndentedFprintf(indent, out, "%s:\n", deplName)
		for _, targetPath := range downloads[deplName] {
			BulletedFprintln(indent+1, out, targetPath)
		}
	}
}

func fprintBundleExports(indent int, out io.Writer, exports map[string][]string) {
	sortedDeplNames := make([]string, 0, len(exports))
	for deplName := range exports {
		sortedDeplNames = append(sortedDeplNames, deplName)
	}
	slices.Sort(sortedDeplNames)
	for _, deplName := range sortedDeplNames {
		if len(exports[deplName]) == 0 {
			continue
		}
		IndentedFprintf(indent, out, "%s:\n", deplName)
		for _, targetPath := range exports[deplName] {
			BulletedFprintln(indent+1, out, targetPath)
		}
	}
}
