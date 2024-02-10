package cli

import (
	"fmt"
	"sort"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/pkg/core"
)

// Print

func PrintPkg(indent int, cache forklift.PathedRepoCache, pkg *core.FSPkg) {
	IndentedPrintf(indent, "Package: %s\n", pkg.Path())
	indent++

	printPkgRepo(indent, cache, pkg)
	if core.CoversPath(cache, pkg.FS.Path()) {
		IndentedPrintf(indent, "Path in cache: %s\n", core.GetSubdirPath(cache, pkg.FS.Path()))
	} else {
		IndentedPrintf(indent, "Absolute path (replacing any cached copy): %s\n", pkg.FS.Path())
	}

	PrintPkgSpec(indent, pkg.Def.Package)
	fmt.Println()
	PrintDeplSpec(indent, pkg.Def.Deployment)
	fmt.Println()
	PrintFeatureSpecs(indent, pkg.Def.Features)
}

func printPkgRepo(indent int, cache forklift.PathedRepoCache, pkg *core.FSPkg) {
	IndentedPrintf(indent, "Provided by repo: %s\n", pkg.Repo.Path())
	indent++

	if core.CoversPath(cache, pkg.FS.Path()) {
		IndentedPrintf(indent, "Version: %s\n", pkg.Repo.Version)
	} else {
		IndentedPrintf(
			indent, "Absolute path (replacing any cached copy): %s\n", pkg.Repo.FS.Path(),
		)
	}

	IndentedPrintf(indent, "Description: %s\n", pkg.Repo.Def.Repo.Description)
}

func PrintPkgSpec(indent int, spec core.PkgSpec) {
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
		BulletedPrintln(indent+1, source)
	}
}

func printMaintainer(indent int, maintainer core.PkgMaintainer) {
	if maintainer.Email != "" {
		BulletedPrintf(indent, "%s <%s>\n", maintainer.Name, maintainer.Email)
	} else {
		BulletedPrintln(indent, maintainer.Name)
	}
}

func PrintDeplSpec(indent int, spec core.PkgDeplSpec) {
	IndentedPrintf(indent, "Deployment:\n")
	indent++

	IndentedPrintf(indent, "Compose files: ")
	if len(spec.ComposeFiles) == 0 {
		fmt.Printf("(none)")
		return
	}
	fmt.Println()
	for _, file := range spec.ComposeFiles {
		BulletedPrintln(indent+1, file)
	}
}

func PrintFeatureSpecs(indent int, features map[string]core.PkgFeatureSpec) {
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
		PrintFeatureSpec(indent, name, features[name])
	}
}

func PrintFeatureSpec(indent int, name string, spec core.PkgFeatureSpec) {
	IndentedPrintf(indent, "%s:\n", name)
	indent++

	IndentedPrintf(indent, "Description: %s\n", spec.Description)

	IndentedPrintf(indent, "Compose files: ")
	if len(spec.ComposeFiles) == 0 {
		fmt.Printf("(none)")
		return
	}
	fmt.Println()
	for _, file := range spec.ComposeFiles {
		BulletedPrintln(indent+1, file)
	}
}
