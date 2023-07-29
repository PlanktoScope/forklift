package cli

import (
	"sort"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Print

func PrintEnvPkgs(indent int, env *forklift.FSEnv, loader forklift.FSPkgLoader) error {
	reqs, err := env.LoadFSRepoReqs("**")
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify Pallet repositories in environment %s", env.FS.Path(),
		)
	}
	pkgs, err := forklift.ListVersionedPkgsOfRepos(loader, reqs)
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

func PrintPkgInfo(indent int, env *forklift.FSEnv, cache forklift.PathedCache, pkgPath string) error {
	pkg, _, err := forklift.LoadRequiredFSPkg(env, cache, pkgPath)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't look up information about package %s in environment %s", pkgPath,
			env.FS.Path(),
		)
	}
	PrintPkg(indent, cache, pkg)
	return nil
}
