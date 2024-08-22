package cli

import (
	"fmt"
	"slices"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
)

// Bundles

func PrintStagedBundle(
	indent int, store *forklift.FSStageStore, bundle *forklift.FSBundle, index int, names []string,
) {
	IndentedPrintf(indent, "Staged pallet bundle: %d\n", index)
	indent++

	IndentedPrint(indent, "Staged names:")
	if len(names) == 0 {
		fmt.Println(" (none)")
	} else {
		fmt.Println()
		for _, name := range names {
			BulletedPrintln(indent+1, name)
		}
	}

	IndentedPrintln(indent, "Pallet:")
	printBundlePallet(indent+1, bundle.Manifest.Pallet)

	IndentedPrint(indent, "Includes:")
	if !bundle.Manifest.Includes.HasInclusions() {
		fmt.Println(" (none)")
	} else {
		fmt.Println()
		printBundleInclusions(indent+1, bundle.Manifest.Includes)
	}

	IndentedPrint(indent, "Deploys:")
	if len(bundle.Manifest.Deploys) == 0 {
		fmt.Println(" (none)")
	} else {
		fmt.Println()
		printBundleDeployments(indent+1, bundle.Manifest.Deploys)
	}

	IndentedPrint(indent, "Downloads:")
	if len(bundle.Manifest.Downloads) == 0 {
		fmt.Println(" (none)")
	} else {
		fmt.Println()
		printBundleDownloads(indent+1, bundle.Manifest.Downloads)
	}

	IndentedPrint(indent, "Exports:")
	if len(bundle.Manifest.Exports) == 0 {
		fmt.Println(" (none)")
	} else {
		fmt.Println()
		printBundleExports(indent+1, bundle.Manifest.Exports)
	}
}

func printBundlePallet(indent int, pallet forklift.BundlePallet) {
	IndentedPrintf(indent, "Path: %s\n", pallet.Path)
	IndentedPrintf(indent, "Version: %s", pallet.Version)
	if !pallet.Clean {
		fmt.Print(" (includes uncommitted changes)")
	}
	fmt.Println()
	IndentedPrintf(indent, "Description: %s", pallet.Description)
	fmt.Println()
}

func printBundleInclusions(indent int, inclusions forklift.BundleInclusions) {
	IndentedPrint(indent, "Pallets:")
	if len(inclusions.Pallets) == 0 {
		fmt.Println(" (none)")
	} else {
		fmt.Println(" (unimplemented)")
		// TODO: implement this once we add support for including pallets
	}
	IndentedPrint(indent, "Repos:")
	if len(inclusions.Repos) == 0 {
		fmt.Println(" (none)")
	} else {
		fmt.Println()
		sortedPaths := make([]string, 0, len(inclusions.Repos))
		for path := range inclusions.Repos {
			sortedPaths = append(sortedPaths, path)
		}
		slices.Sort(sortedPaths)
		for _, path := range sortedPaths {
			printBundleRepoInclusion(indent+1, path, inclusions.Repos[path])
		}
	}
}

func printBundleRepoInclusion(indent int, path string, inclusion forklift.BundleRepoInclusion) {
	IndentedPrintf(indent, "%s:\n", path)
	indent++
	IndentedPrintf(indent, "Required version")
	if inclusion.Override == (forklift.BundleInclusionOverride{}) {
		fmt.Print(" (overridden)")
	}
	fmt.Print(": ")
	fmt.Println(inclusion.Req.VersionLock.Version)

	if inclusion.Override == (forklift.BundleInclusionOverride{}) {
		return
	}
	IndentedPrintln(indent, "Override:")
	IndentedPrintf(indent+1, "Path: %s\n", inclusion.Override.Path)
	IndentedPrint(indent+1, "Version: ")
	if inclusion.Override.Version == "" {
		fmt.Print("(unknown)")
	} else {
		fmt.Print(inclusion.Override.Version)
	}
	if !inclusion.Override.Clean {
		fmt.Print(" (includes uncommitted changes)")
	}
	fmt.Println()
}

func printBundleDeployments(indent int, deployments map[string]forklift.DeplDef) {
	sortedDeplNames := make([]string, 0, len(deployments))
	for deplName := range deployments {
		sortedDeplNames = append(sortedDeplNames, deplName)
	}
	slices.Sort(sortedDeplNames)
	for _, deplName := range sortedDeplNames {
		IndentedPrintf(indent, "%s: %s\n", deplName, deployments[deplName].Package)
	}
}

func printBundleDownloads(indent int, downloads map[string][]string) {
	sortedDeplNames := make([]string, 0, len(downloads))
	for deplName := range downloads {
		sortedDeplNames = append(sortedDeplNames, deplName)
	}
	slices.Sort(sortedDeplNames)
	for _, deplName := range sortedDeplNames {
		IndentedPrintf(indent, "%s:", deplName)
		if len(downloads[deplName]) == 0 {
			fmt.Print(" (none)")
		}
		fmt.Println()
		for _, targetPath := range downloads[deplName] {
			BulletedPrintln(indent+1, targetPath)
		}
	}
}

func printBundleExports(indent int, exports map[string][]string) {
	sortedDeplNames := make([]string, 0, len(exports))
	for deplName := range exports {
		sortedDeplNames = append(sortedDeplNames, deplName)
	}
	slices.Sort(sortedDeplNames)
	for _, deplName := range sortedDeplNames {
		IndentedPrintf(indent, "%s:", deplName)
		if len(exports[deplName]) == 0 {
			fmt.Print(" (none)")
		}
		fmt.Println()
		for _, targetPath := range exports[deplName] {
			BulletedPrintln(indent+1, targetPath)
		}
	}
}
