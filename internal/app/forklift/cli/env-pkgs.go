package cli

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Print

func PrintEnvPkgs(
	indent int,
	env *forklift.FSEnv, cache *forklift.FSCache, replacementRepos map[string]*pallets.FSRepo,
) error {
	repos, err := env.LoadFSRepoRequirements("**")
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify Pallet repositories in environment %s", env.FS.Path(),
		)
	}
	pkgs, err := forklift.ListVersionedPkgs(cache, replacementRepos, repos)
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet packages")
	}
	sort.Slice(pkgs, func(i, j int) bool {
		return pallets.ComparePkgs(pkgs[i].Pkg, pkgs[j].Pkg) < 0
	})
	for _, pkg := range pkgs {
		IndentedPrintf(indent, "%s\n", pkg.Path())
	}
	return nil
}

func PrintPkgInfo(
	indent int,
	env *forklift.FSEnv, cache *forklift.FSCache, replacementRepos map[string]*pallets.FSRepo,
	pkgPath string,
) error {
	var pkg *forklift.VersionedPkg
	var err error
	if repo, ok := forklift.FindExternalRepoOfPkg(replacementRepos, pkgPath); ok {
		externalPkg, perr := repo.LoadFSPkg(repo.GetPkgSubdir(pkgPath))
		if perr != nil {
			return errors.Wrapf(
				perr, "couldn't find external package %s from replacement repo %s", pkgPath, repo.FS.Path(),
			)
		}
		pkg = forklift.AsVersionedPkg(externalPkg)
	} else if pkg, err = env.LoadVersionedPkg(cache, pkgPath); err != nil {
		return errors.Wrapf(
			err, "couldn't look up information about package %s in environment %s", pkgPath,
			env.FS.Path(),
		)
	}

	printVersionedPkg(indent, cache, pkg)
	return nil
}

func printVersionedPkg(indent int, cache *forklift.FSCache, pkg *forklift.VersionedPkg) {
	IndentedPrintf(indent, "Pallet package: %s\n", pkg.Path())
	indent++

	printVersionedPkgRepo(indent, cache, pkg)
	if cache.CoversPath(pkg.FS.Path()) {
		IndentedPrintf(indent, "Path in cache: %s\n", cache.TrimCachePathPrefix(pkg.FS.Path()))
	} else {
		IndentedPrintf(indent, "External path (replacing cached package): %s\n", pkg.FS.Path())
	}

	PrintPkgSpec(indent, pkg.Config.Package)
	fmt.Println()
	PrintDeplSpec(indent, pkg.Config.Deployment)
	fmt.Println()
	PrintFeatureSpecs(indent, pkg.Config.Features)
}

func printVersionedPkgRepo(indent int, cache *forklift.FSCache, pkg *forklift.VersionedPkg) {
	IndentedPrintf(indent, "Provided by Pallet repository: %s\n", pkg.Repo.Path())
	indent++

	if cache.CoversPath(pkg.FS.Path()) {
		IndentedPrintf(indent, "Version: %s\n", pkg.Repo.Version)
	} else {
		IndentedPrintf(
			indent, "External path (replacing cached repository): %s\n", pkg.Repo.FS.Path(),
		)
	}

	IndentedPrintf(indent, "Description: %s\n", pkg.Repo.Config.Repository.Description)
	IndentedPrintf(indent, "Provided by Git repository: %s\n", pkg.Repo.VCSRepoPath)
}
