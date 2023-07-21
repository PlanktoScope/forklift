package cli

import (
	"fmt"
	"sort"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Print

func PrintPkgSpec(indent int, spec pallets.PkgSpec) {
	IndentedPrintf(indent, "Description: %s\n", spec.Description)

	IndentedPrint(indent, "Maintainers:")
	if len(spec.Maintainers) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for _, maintainer := range spec.Maintainers {
		printMaintainer(indent+1, maintainer)
	}

	if spec.License != "" {
		IndentedPrintf(indent, "License: %s\n", spec.License)
	} else {
		IndentedPrintf(indent, "License: (custom license)\n")
	}

	IndentedPrint(indent, "Sources:")
	if len(spec.Sources) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	for _, source := range spec.Sources {
		BulletedPrintf(indent+1, "%s\n", source)
	}
}

func printMaintainer(indent int, maintainer pallets.PkgMaintainer) {
	if maintainer.Email != "" {
		BulletedPrintf(indent, "%s <%s>\n", maintainer.Name, maintainer.Email)
	} else {
		BulletedPrintf(indent, "%s\n", maintainer.Name)
	}
}

func PrintDeplSpec(indent int, spec pallets.PkgDeplSpec) {
	IndentedPrintf(indent, "Deployment:\n")
	indent++

	// TODO: actually display the definition file?
	IndentedPrintf(indent, "Definition file: ")
	if len(spec.DefinitionFile) == 0 {
		fmt.Println("(none)")
		return
	}
	fmt.Println(spec.DefinitionFile)
}

func PrintFeatureSpecs(indent int, features map[string]pallets.PkgFeatureSpec) {
	IndentedPrint(indent, "Optional features:")
	names := make([]string, 0, len(features))
	for name := range features {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++

	for _, name := range names {
		if description := features[name].Description; description != "" {
			IndentedPrintf(indent, "%s: %s\n", name, description)
			continue
		}
		IndentedPrintf(indent, "%s\n", name)
	}
}
