package cli

import (
	"path"
	"sort"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// Print

func PrintEnvPkgs(indent int, env *forklift.FSEnv, loader forklift.FSPkgLoader) error {
	reqs, err := env.LoadFSPalletReqs("**")
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify pallets in environment %s", env.FS.Path(),
		)
	}
	pkgs := make([]*pallets.FSPkg, 0)
	for _, req := range reqs {
		palletCachePath := req.GetCachePath()
		loaded, err := loader.LoadFSPkgs(path.Join(palletCachePath, "**"))
		if err != nil {
			return errors.Wrapf(err, "couldn't load packages from pallet cached at %s", palletCachePath)
		}
		pkgs = append(pkgs, loaded...)
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
	indent int, env *forklift.FSEnv, cache forklift.PathedPalletCache, pkgPath string,
) error {
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
