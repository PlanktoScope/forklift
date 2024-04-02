package cli

import (
	"fmt"
	"slices"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
)

// Print

func PrintStagedBundle(
	indent int, store *forklift.FSStageStore, bundle *forklift.FSBundle, index int,
) {
	IndentedPrintf(indent, "Staged pallet bundle: %d\n", index)
	indent++

	IndentedPrintf(indent, "Forklift version: %s\n", bundle.Def.ForkliftVersion)

	IndentedPrintln(indent, "Pallet:")
	printBundlePallet(indent+1, bundle.Def.Pallet)

	IndentedPrint(indent, "Includes:")
	if !bundle.Def.Includes.HasInclusions() {
		fmt.Println(" (none)")
	} else {
		fmt.Println()
		printBundleInclusions(indent+1, bundle.Def.Includes)
	}

	IndentedPrint(indent, "Deploys:")
	if len(bundle.Def.Deploys) == 0 {
		fmt.Println(" (none)")
	} else {
		fmt.Println()
		printBundleDeployments(indent+1, bundle.Def.Deploys)
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
	sortedPaths := make([]string, 0, len(deployments))
	for path := range deployments {
		sortedPaths = append(sortedPaths, path)
	}
	slices.Sort(sortedPaths)
	for _, path := range sortedPaths {
		IndentedPrintln(indent, path)
	}
}
