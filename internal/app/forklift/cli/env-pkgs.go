package cli

import (
	"path"
	"sort"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
	"github.com/PlanktoScope/forklift/pkg/core"
)

// Print

func PrintEnvPkgs(indent int, env *forklift.FSEnv, loader forklift.FSPkgLoader) error {
	reqs, err := env.LoadFSRepoReqs("**")
	if err != nil {
		return errors.Wrapf(
			err, "couldn't identify repos in environment %s", env.FS.Path(),
		)
	}
	pkgs := make([]*core.FSPkg, 0)
	for _, req := range reqs {
		repoCachePath := req.GetCachePath()
		loaded, err := loader.LoadFSPkgs(path.Join(repoCachePath, "**"))
		if err != nil {
			return errors.Wrapf(err, "couldn't load packages from repo cached at %s", repoCachePath)
		}
		pkgs = append(pkgs, loaded...)
	}
	sort.Slice(pkgs, func(i, j int) bool {
		return core.ComparePkgs(pkgs[i].Pkg, pkgs[j].Pkg) < 0
	})
	for _, pkg := range pkgs {
		IndentedPrintf(indent, "%s\n", pkg.Path())
	}
	return nil
}

func PrintPkgInfo(
	indent int, env *forklift.FSEnv, cache forklift.PathedRepoCache, pkgPath string,
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
