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

func fprintBundleDownloads(
	indent int, out io.Writer, downloads map[string]forklift.BundleDeplDownloads,
) {
	sortedDeplNames := make([]string, 0, len(downloads))
	for deplName := range downloads {
		sortedDeplNames = append(sortedDeplNames, deplName)
	}
	slices.Sort(sortedDeplNames)
	for _, deplName := range sortedDeplNames {
		depl := downloads[deplName]
		if len(depl.All()) == 0 {
			continue
		}
		IndentedFprintf(indent, out, "%s:\n", deplName)
		deplIndent := indent + 1
		if len(depl.HTTPFile) > 0 {
			IndentedFprintln(deplIndent, out, "HTTP Files:")
			for _, targetPath := range depl.HTTPFile {
				BulletedFprintln(deplIndent+1, out, targetPath)
			}
		}
		if len(depl.OCIImage) > 0 {
			IndentedFprintln(deplIndent, out, "OCI Images:")
			for _, targetPath := range depl.OCIImage {
				BulletedFprintln(deplIndent+1, out, targetPath)
			}
		}
	}
}

func fprintBundleExports(indent int, out io.Writer, exports map[string]forklift.BundleDeplExports) {
	sortedDeplNames := make([]string, 0, len(exports))
	for deplName := range exports {
		sortedDeplNames = append(sortedDeplNames, deplName)
	}
	slices.Sort(sortedDeplNames)
	for _, deplName := range sortedDeplNames {
		depl := exports[deplName]
		if len(depl.All()) == 0 {
			continue
		}
		IndentedFprintf(indent, out, "%s:\n", deplName)
		deplIndent := indent + 1
		if len(depl.File) > 0 {
			IndentedFprintln(deplIndent, out, "Files:")
			for _, targetPath := range depl.File {
				BulletedFprintln(deplIndent+1, out, targetPath)
			}
		}
		if depl.ComposeApp.Name != "" {
			IndentedFprintf(deplIndent, out, "Compose App: %s\n", depl.ComposeApp.Name)
			fprintOptionalList(deplIndent+1, out, "Services", depl.ComposeApp.Services)
			fprintOptionalList(deplIndent+1, out, "Images", depl.ComposeApp.Images)
			fprintOptionalList(
				deplIndent+1, out, "Bind Mounts (auto-created)", depl.ComposeApp.CreatedBindMounts,
			)
			fprintOptionalList(
				deplIndent+1, out, "Bind Mounts (required)", depl.ComposeApp.RequiredBindMounts,
			)
			fprintOptionalList(
				deplIndent+1, out, "Volumes (auto-created)", depl.ComposeApp.CreatedVolumes,
			)
			fprintOptionalList(
				deplIndent+1, out, "Volumes (required)", depl.ComposeApp.RequiredVolumes,
			)
			fprintOptionalList(
				deplIndent+1, out, "Networks (auto-created)", depl.ComposeApp.CreatedNetworks,
			)
			fprintOptionalList(
				deplIndent+1, out, "Networks (required)", depl.ComposeApp.RequiredNetworks,
			)
		}
	}
}

func fprintOptionalList(indent int, out io.Writer, name string, items []string) {
	if len(items) == 0 {
		return
	}
	IndentedFprintf(indent, out, "%s:\n", name)
	for _, item := range items {
		BulletedFprintln(indent+1, out, item)
	}
}
