package cli

import (
	"fmt"
	"io"
	"slices"

	"github.com/forklift-run/forklift/internal/app/forklift"
	"github.com/forklift-run/forklift/pkg/structures"
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
	if len(bundle.Manifest.Downloads) == 0 {
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
		_, _ = fmt.Fprintln(out)
		sortedPaths := make([]string, 0, len(inclusions.Pallets))
		for path := range inclusions.Pallets {
			sortedPaths = append(sortedPaths, path)
		}
		slices.Sort(sortedPaths)
		for _, path := range sortedPaths {
			inclusion := inclusions.Pallets[path]
			fprintBundleInclusion(indent+1, out, path, inclusion.Override, inclusion.Req.VersionLock)
		}
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
			inclusion := inclusions.Repos[path]
			fprintBundleInclusion(indent+1, out, path, inclusion.Override, inclusion.Req.VersionLock)
		}
	}
}

func fprintBundleInclusion(
	indent int, out io.Writer, path string,
	inclOverride forklift.BundleInclusionOverride, inclReqVersionLock forklift.VersionLock,
) {
	IndentedFprintf(indent, out, "%s:\n", path)
	indent++
	IndentedFprint(indent, out, "Required version")
	if inclOverride != (forklift.BundleInclusionOverride{}) {
		_, _ = fmt.Fprint(out, " (overridden)")
	}
	_, _ = fmt.Fprint(out, ": ")
	_, _ = fmt.Fprintln(out, inclReqVersionLock.Version)

	if inclOverride == (forklift.BundleInclusionOverride{}) {
		return
	}
	IndentedFprintln(indent, out, "Override:")
	IndentedFprintf(indent+1, out, "Path: %s\n", inclOverride.Path)
	IndentedFprint(indent+1, out, "Version: ")
	if inclOverride.Version == "" {
		_, _ = fmt.Fprint(out, "(unknown)")
	} else {
		_, _ = fmt.Fprint(out, inclOverride.Version)
	}
	if !inclOverride.Clean {
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

func fprintBundleDownloads(
	indent int, out io.Writer, downloads map[string]forklift.BundleDeplDownloads,
) {
	lists := []string{"httpFiles", "ociImages"}
	aggs := make(map[string]structures.Set[string])
	for _, l := range lists {
		aggs[l] = make(structures.Set[string])
	}

	for _, depl := range downloads {
		aggs["httpFiles"].Add(depl.HTTPFile...)
		aggs["ociImages"].Add(depl.OCIImage...)
	}
	fprintOptionalSet(indent, out, "HTTP Files", aggs["httpFiles"])
	fprintOptionalSet(indent, out, "OCI Images", aggs["ociImages"])
}

func fprintOptionalSet(indent int, out io.Writer, name string, items structures.Set[string]) {
	if len(items) == 0 {
		return
	}
	IndentedFprintf(indent, out, "%s:\n", name)
	for _, item := range slices.Sorted(items.All()) {
		BulletedFprintln(indent+1, out, item)
	}
}

func fprintBundleExports(indent int, out io.Writer, exports map[string]forklift.BundleDeplExports) {
	lists := []string{
		"files", "appNames", "appServices", "appImages", "appNewBindMounts", "appReqBindMounts",
		"appNewVolumes", "appReqVolumes", "appNewNetworks", "appReqNetworks",
	}
	aggs := make(map[string]structures.Set[string])
	for _, l := range lists {
		aggs[l] = make(structures.Set[string])
	}

	for _, depl := range exports {
		aggs["files"].Add(depl.File...)
		if depl.ComposeApp.Name != "" {
			aggs["appNames"].Add(depl.ComposeApp.Name)
			for _, service := range depl.ComposeApp.Services {
				aggs["appServices"].Add(fmt.Sprintf("%s-%s", depl.ComposeApp.Name, service))
			}
			aggs["appImages"].Add(depl.ComposeApp.Images...)
			aggs["appNewBindMounts"].Add(depl.ComposeApp.CreatedBindMounts...)
			aggs["appReqBindMounts"].Add(depl.ComposeApp.RequiredBindMounts...)
			aggs["appNewVolumes"].Add(depl.ComposeApp.CreatedVolumes...)
			aggs["appReqVolumes"].Add(depl.ComposeApp.RequiredVolumes...)
			aggs["appNewNetworks"].Add(depl.ComposeApp.CreatedNetworks...)
			aggs["appReqNetworks"].Add(depl.ComposeApp.RequiredNetworks...)
		}
	}

	aggs["appReqBindMounts"] = aggs["appReqBindMounts"].Difference(aggs["appNewBindMounts"])
	aggs["appReqVolumes"] = aggs["appReqVolumes"].Difference(aggs["appNewVolumes"])
	aggs["appReqNetworks"] = aggs["appReqNetworks"].Difference(aggs["appNewNetworks"])

	fprintOptionalSet(indent, out, "Files", aggs["files"])
	fprintOptionalSet(indent, out, "Compose Apps", aggs["appNames"])
	fprintOptionalSet(indent, out, "Compose App Services", aggs["appServices"])
	fprintOptionalSet(indent, out, "Compose App Images", aggs["appImages"])
	fprintOptionalSet(indent, out, "Compose App Bind Mounts (auto-created)", aggs["appNewBindMounts"])
	fprintOptionalSet(indent, out, "Compose App Bind Mounts (required)", aggs["appReqBindMounts"])
	fprintOptionalSet(indent, out, "Compose App Volumes (auto-created)", aggs["appNewVolumes"])
	fprintOptionalSet(indent, out, "Compose App Volumes (required)", aggs["appReqVolumes"])
	fprintOptionalSet(indent, out, "Compose App Networks (auto-created)", aggs["appNewNetworks"])
	fprintOptionalSet(indent, out, "Compose App Networks (required)", aggs["appReqNetworks"])
}
