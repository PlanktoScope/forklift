package forklift

import (
	"github.com/pkg/errors"

	fpkg "github.com/forklift-run/forklift/pkg/packaging"
	"github.com/forklift-run/forklift/pkg/structures"
)

// A ResolvedDepl is a deployment with a loaded package.
type ResolvedDepl struct {
	// Depl is the declared deployment of the package represented by Pkg.
	Depl
	// PkgReq is the package requirement for the deployment.
	PkgReq PkgReq
	// Pkg is the package to be deployed.
	Pkg *fpkg.FSPkg
}

// ResolveDepl loads the package from the [FSPkgLoader] instance based on the requirements in the
// provided deployment and the package requirement loader.
func ResolveDepl(
	pkgReqLoader PkgReqLoader, pkgLoader FSPkgLoader, depl Depl,
) (resolved *ResolvedDepl, err error) {
	resolved = &ResolvedDepl{
		Depl: depl,
	}
	pkgPath := resolved.Decl.Package
	if resolved.Pkg, resolved.PkgReq, err = LoadRequiredFSPkg(
		pkgReqLoader, pkgLoader, pkgPath,
	); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load package %s to resolve from package deployment %s", pkgPath, depl.Name,
		)
	}
	return resolved, nil
}

// ResolveDepls loads the packages from the [FSPkgLoader] instance based on the requirements in the
// provided deployments and the package requirement loader.
func ResolveDepls(
	pkgReqLoader PkgReqLoader, pkgLoader FSPkgLoader, depls []Depl,
) (resolved []*ResolvedDepl, err error) {
	resolvedDepls := make([]*ResolvedDepl, 0, len(depls))
	for _, depl := range depls {
		resolved, err := ResolveDepl(pkgReqLoader, pkgLoader, depl)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't resolve package deployment %s", depl.Name)
		}
		resolvedDepls = append(resolvedDepls, resolved)
	}

	return resolvedDepls, nil
}

func (d *ResolvedDepl) Check() (errs []error) {
	if d.PkgReq.Path() != d.Decl.Package {
		errs = append(errs, errors.Errorf(
			"required package %s does not match required package %s in deployment declaration",
			d.PkgReq.Path(), d.Decl.Package,
		))
	}
	if d.PkgReq.Path() != d.Pkg.Path() {
		errs = append(errs, errors.Errorf(
			"resolved package %s does not match required package %s", d.Pkg.Path(), d.PkgReq.Path(),
		))
	}
	// An empty version is treated as "any version" for this check, so that packages loaded from
	// overriding pkg trees (where versioning is ignored) will not fail this version check:
	if d.Pkg.FSPkgTree.Version != "" &&
		d.PkgReq.Pallet.VersionLock.Version != d.Pkg.FSPkgTree.Version {
		errs = append(errs, errors.Errorf(
			"resolved package version %s does not match required package version %s",
			d.Pkg.FSPkgTree.Version, d.PkgReq.Pallet.VersionLock.Version,
		))
	}
	return errs
}

// EnabledFeatures returns a map of the package features enabled by the deployment's declaration,
// with feature names as the keys of the map.
func (d *ResolvedDepl) EnabledFeatures() (enabled map[string]fpkg.PkgFeatureSpec, err error) {
	all := d.Pkg.Decl.Features
	enabled = make(map[string]fpkg.PkgFeatureSpec)
	unrecognized := make([]string, 0, len(d.Decl.Features))
	for _, name := range d.Decl.Features {
		featureSpec, ok := all[name]
		if !ok {
			unrecognized = append(unrecognized, name)
			continue
		}
		enabled[name] = featureSpec
	}
	if len(unrecognized) > 0 {
		return enabled, errors.Errorf("unrecognized feature flags: %+v", unrecognized)
	}
	return enabled, nil
}

// DisabledFeatures returns a map of the package features not enabled by the deployment's
// declaration, with feature names as the keys of the map.
func (d *ResolvedDepl) DisabledFeatures() map[string]fpkg.PkgFeatureSpec {
	all := d.Pkg.Decl.Features
	enabled := make(structures.Set[string])
	for _, name := range d.Decl.Features {
		enabled.Add(name)
	}
	disabled := make(map[string]fpkg.PkgFeatureSpec)
	for name := range all {
		if enabled.Has(name) {
			continue
		}
		disabled[name] = all[name]
	}
	return disabled
}
