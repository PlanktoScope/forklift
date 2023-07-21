package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Print

func PrintEnvPkgs(
	indent int, envPath, cachePath string, replacementRepos map[string]*pallets.FSRepo,
) error {
	repos, err := forklift.ListVersionedRepos(os.DirFS(envPath))
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet repositories in environment %s", envPath)
	}
	pkgs, err := forklift.ListVersionedPkgs(os.DirFS(cachePath), replacementRepos, repos)
	if err != nil {
		return errors.Wrapf(err, "couldn't identify Pallet packages")
	}
	sort.Slice(pkgs, func(i, j int) bool {
		return forklift.CompareCachedPkgs(pkgs[i], pkgs[j]) < 0
	})
	for _, pkg := range pkgs {
		IndentedPrintf(indent, "%s\n", pkg.Path())
	}
	return nil
}

func PrintPkgInfo(
	indent int, envPath, cachePath string, replacementRepos map[string]*pallets.FSRepo,
	pkgPath string,
) error {
	reposFS, err := forklift.VersionedReposFS(os.DirFS(envPath))
	if err != nil {
		return errors.Wrapf(
			err, "couldn't open directory for Pallet repositories in environment %s", envPath,
		)
	}

	var pkg *forklift.VersionedPkg
	repo, ok := forklift.FindExternalRepoOfPkg(replacementRepos, pkgPath)
	if ok {
		externalPkg, perr := forklift.FindExternalPkg(repo, pkgPath)
		if perr != nil {
			return errors.Wrapf(
				err, "couldn't find external package %s from replacement repo %s", pkgPath, repo.FS.Path(),
			)
		}
		pkg = forklift.AsVersionedPkg(externalPkg)
	} else if pkg, err = forklift.LoadVersionedPkg(reposFS, os.DirFS(cachePath), pkgPath); err != nil {
		return errors.Wrapf(
			err, "couldn't look up information about package %s in environment %s", pkgPath, envPath,
		)
	}

	printVersionedPkg(indent, pkg)
	return nil
}

func printVersionedPkg(indent int, pkg *forklift.VersionedPkg) {
	IndentedPrintf(indent, "Pallet package: %s\n", pkg.Path())
	indent++

	printVersionedPkgRepo(indent, pkg)
	if filepath.IsAbs(pkg.FS.Path()) {
		IndentedPrint(indent, "External path (replacing cached package): ")
	} else {
		IndentedPrint(indent, "Path in cache: ")
	}
	fmt.Println(pkg.FS.Path())
	fmt.Println()

	PrintPkgSpec(indent, pkg.Config.Package)
	fmt.Println()
	PrintDeplSpec(indent, pkg.Config.Deployment)
	fmt.Println()
	PrintFeatureSpecs(indent, pkg.Config.Features)
}

func printVersionedPkgRepo(indent int, pkg *forklift.VersionedPkg) {
	IndentedPrintf(indent, "Provided by Pallet repository: %s\n", pkg.Repo.Path())
	indent++

	if filepath.IsAbs(pkg.FS.Path()) {
		IndentedPrintf(
			indent, "External path (replacing cached repository): %s\n", pkg.Repo.FS.Path(),
		)
	} else {
		IndentedPrintf(indent, "Version: %s\n", pkg.Repo.Version)
	}

	IndentedPrintf(indent, "Description: %s\n", pkg.Repo.Config.Repository.Description)
	IndentedPrintf(indent, "Provided by Git repository: %s\n", pkg.Repo.VCSRepoPath)
}
