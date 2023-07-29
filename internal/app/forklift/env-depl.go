package forklift

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// ResolvedDepl

// ResolveDepl loads the Pallet package from the [FSPkgLoader] instance based on the requirements in
// the provided deployment and the package requirement loader.
func ResolveDepl(
	pkgReqLoader PkgReqLoader, pkgLoader FSPkgLoader, depl Depl,
) (resolved *ResolvedDepl, err error) {
	resolved = &ResolvedDepl{
		Depl: depl,
	}
	pkgPath := resolved.Def.Package
	if resolved.Pkg, resolved.PkgReq, err = LoadRequiredFSPkg(
		pkgReqLoader, pkgLoader, pkgPath,
	); err != nil {
		return nil, errors.Wrapf(
			err, "couldn't load package %s to resolved from package deployment %s",
			pkgPath, depl.Name,
		)
	}
	return resolved, nil
}

// ResolveDepls loads the Pallet packages from the [FSPkgLoader] instance based on the requirements
// in the provided deployments and the package requirement loader.
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
	if d.PkgReq.Path() != d.Def.Package {
		errs = append(errs, errors.Errorf(
			"required package %s does not match required package %s in deployment configuration",
			d.PkgReq.Path(), d.Def.Package,
		))
	}
	if d.PkgReq.Path() != d.Pkg.Path() {
		errs = append(errs, errors.Errorf(
			"resolved package %s does not match required package %s", d.Pkg.Path(), d.PkgReq.Path(),
		))
	}
	// An empty version is treated as "any version" for this check, so that packages loaded from
	// overriding repos (where versioning is ignored) will not fail this version check:
	if d.Pkg.Repo.Version != "" && d.PkgReq.Repo.VersionLock.Version != d.Pkg.Repo.Version {
		errs = append(errs, errors.Errorf(
			"resolved package version %s does not match required package version %s",
			d.Pkg.Repo.Version, d.PkgReq.Repo.VersionLock.Version,
		))
	}
	return errs
}

// EnabledFeatures returns a map of the Pallet package features enabled by the deployment's
// configuration, with feature names as the keys of the map.
func (d *ResolvedDepl) EnabledFeatures() (enabled map[string]pallets.PkgFeatureSpec, err error) {
	all := d.Pkg.Def.Features
	enabled = make(map[string]pallets.PkgFeatureSpec)
	for _, name := range d.Def.Features {
		featureSpec, ok := all[name]
		if !ok {
			return nil, errors.Errorf("unrecognized feature %s", name)
		}
		enabled[name] = featureSpec
	}
	return enabled, nil
}

// DisabledFeatures returns a map of the Pallet package features not enabled by the deployment's
// configuration, with feature names as the keys of the map.
func (d *ResolvedDepl) DisabledFeatures() map[string]pallets.PkgFeatureSpec {
	all := d.Pkg.Def.Features
	enabled := make(map[string]struct{})
	for _, name := range d.Def.Features {
		enabled[name] = struct{}{}
	}
	disabled := make(map[string]pallets.PkgFeatureSpec)
	for name := range all {
		if _, ok := enabled[name]; ok {
			continue
		}
		disabled[name] = all[name]
	}
	return disabled
}

// sortKeys returns an alphabetically sorted slice of the keys of a map with string keys.
func sortKeys[Value any](m map[string]Value) (sorted []string) {
	sorted = make([]string, 0, len(m))
	for key := range m {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)
	return sorted
}

// CheckConflicts produces a report of all resource conflicts between the ResolvedDepl instance and
// a candidate ResolvedDepl.
func (d *ResolvedDepl) CheckConflicts(candidate *ResolvedDepl) (DeplConflict, error) {
	enabledFeatures, err := d.EnabledFeatures()
	if err != nil {
		return DeplConflict{}, errors.Errorf(
			"couldn't determine enabled features of deployment %s", d.Name,
		)
	}
	candidateEnabledFeatures, err := candidate.EnabledFeatures()
	if err != nil {
		return DeplConflict{}, errors.Errorf(
			"couldn't determine enabled features of deployment %s", candidate.Name,
		)
	}
	return DeplConflict{
		First:  d,
		Second: candidate,
		Name:   d.Name == candidate.Name,
		Listeners: pallets.CheckResourcesConflicts(
			d.providedListeners(enabledFeatures), candidate.providedListeners(candidateEnabledFeatures),
		),
		Networks: pallets.CheckResourcesConflicts(
			d.providedNetworks(enabledFeatures), candidate.providedNetworks(candidateEnabledFeatures),
		),
		Services: pallets.CheckResourcesConflicts(
			d.providedServices(enabledFeatures), candidate.providedServices(candidateEnabledFeatures),
		),
	}, nil
}

// providedListeners returns a slice of all host port listeners provided by the Pallet package
// deployment, depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) providedListeners(
	enabledFeatures map[string]pallets.PkgFeatureSpec,
) (provided []pallets.AttachedResource[pallets.ListenerResource]) {
	return d.Pkg.ProvidedListeners(d.ResourceAttachmentSource(), sortKeys(enabledFeatures))
}

// requiredNetworks returns a slice of all Docker networks required by the Pallet package
// deployment, depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) requiredNetworks(
	enabledFeatures map[string]pallets.PkgFeatureSpec,
) (required []pallets.AttachedResource[pallets.NetworkResource]) {
	return d.Pkg.RequiredNetworks(d.ResourceAttachmentSource(), sortKeys(enabledFeatures))
}

// providedNetworks returns a slice of all Docker networks provided by the Pallet package
// deployment, depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) providedNetworks(
	enabledFeatures map[string]pallets.PkgFeatureSpec,
) (provided []pallets.AttachedResource[pallets.NetworkResource]) {
	return d.Pkg.ProvidedNetworks(d.ResourceAttachmentSource(), sortKeys(enabledFeatures))
}

// requiredServices returns a slice of all network services required by the Pallet package
// deployment, depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) requiredServices(
	enabledFeatures map[string]pallets.PkgFeatureSpec,
) (required []pallets.AttachedResource[pallets.ServiceResource]) {
	return d.Pkg.RequiredServices(d.ResourceAttachmentSource(), sortKeys(enabledFeatures))
}

// providedServices returns a slice of all network services provided by the Pallet package
// deployment, depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) providedServices(
	enabledFeatures map[string]pallets.PkgFeatureSpec,
) (provided []pallets.AttachedResource[pallets.ServiceResource]) {
	return d.Pkg.ProvidedServices(d.ResourceAttachmentSource(), sortKeys(enabledFeatures))
}

// CheckAllConflicts produces a slice of reports of all resource conflicts between the ResolvedDepl
// instance and each candidate ResolvedDepl.
func (d *ResolvedDepl) CheckAllConflicts(
	candidates []*ResolvedDepl,
) (conflicts []DeplConflict, err error) {
	conflicts = make([]DeplConflict, 0, len(candidates))
	for _, candidate := range candidates {
		conflict, err := d.CheckConflicts(candidate)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't check conflicts with deployment %s", candidate.Name)
		}
		if conflict.HasConflict() {
			conflicts = append(conflicts, conflict)
		}
	}
	return conflicts, nil
}

// CheckMissingDependencies produces a report of all resource requirements from the ResolvedDepl
// instance and which were not met by any candidate ResolvedDepl.
func (d *ResolvedDepl) CheckMissingDependencies(
	candidates []*ResolvedDepl,
) (MissingDeplDependencies, error) {
	enabledFeatures, err := d.EnabledFeatures()
	if err != nil {
		return MissingDeplDependencies{}, errors.Errorf(
			"couldn't determine enabled features of deployment %s", d.Name,
		)
	}
	candidateEnabledFeatures := make([]map[string]pallets.PkgFeatureSpec, 0, len(candidates))
	for _, candidate := range candidates {
		f, err := candidate.EnabledFeatures()
		if err != nil {
			return MissingDeplDependencies{}, errors.Errorf(
				"couldn't determine enabled features of deployment %s", candidate.Name,
			)
		}
		candidateEnabledFeatures = append(candidateEnabledFeatures, f)
	}

	var (
		allProvidedNetworks []pallets.AttachedResource[pallets.NetworkResource]
		allProvidedServices []pallets.AttachedResource[pallets.ServiceResource]
	)
	for i, candidate := range candidates {
		allProvidedNetworks = append(
			allProvidedNetworks, candidate.providedNetworks(candidateEnabledFeatures[i])...,
		)
		allProvidedServices = append(
			allProvidedServices, candidate.providedServices(candidateEnabledFeatures[i])...,
		)
	}

	return MissingDeplDependencies{
		Depl: d,
		Networks: pallets.CheckResourcesDependencies(
			d.requiredNetworks(enabledFeatures), allProvidedNetworks,
		),
		Services: pallets.CheckResourcesDependencies(
			pallets.SplitServicesByPath(d.requiredServices(enabledFeatures)), allProvidedServices,
		),
	}, nil
}

// CheckDeplConflicts produces a slice of reports of all resource conflicts among all provided
// ResolvedDepl instances.
func CheckDeplConflicts(depls []*ResolvedDepl) (conflicts []DeplConflict, err error) {
	for i, depl := range depls {
		deplConflicts, err := depl.CheckAllConflicts(depls[i+1:])
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't check for conflicts with deployment %s", depl.Name)
		}
		conflicts = append(conflicts, deplConflicts...)
	}
	return conflicts, nil
}

// CheckDeplDependencies produces a slice of reports of all unsatisfied resource dependencies among
// all provided ResolvedDepl instances.
func CheckDeplDependencies(
	depls []*ResolvedDepl,
) (missingDeps []MissingDeplDependencies, err error) {
	for _, depl := range depls {
		deplMissingDeps, err := depl.CheckMissingDependencies(depls)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't check dependencies of deployment %s", depl.Name)
		}
		if deplMissingDeps.HasMissingDependency() {
			missingDeps = append(missingDeps, deplMissingDeps)
		}
	}
	return missingDeps, nil
}

// Depl

// loadDepl loads the Depl from a file path in the provided base filesystem, assuming the file path
// is the specified name of the deployment followed by the deployment config file extension.
func loadDepl(fsys pallets.PathedFS, name string) (depl Depl, err error) {
	depl.Name = name
	if depl.Def, err = loadDeplDef(fsys, name+DeplDefFileExt); err != nil {
		return Depl{}, errors.Wrapf(err, "couldn't load version depl config")
	}
	return depl, nil
}

// loadDepls loads all Pallet package deployment configurations from the provided base filesystem
// matching the specified search pattern.
// The search pattern should not include the file extension for deployment specification files - the
// file extension will be appended to the search pattern by LoadDepls.
func loadDepls(fsys pallets.PathedFS, searchPattern string) ([]Depl, error) {
	searchPattern = fmt.Sprintf("%s%s", searchPattern, DeplDefFileExt)
	deplDefFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for Pallet package deployment configs matching %s/%s",
			fsys.Path(), searchPattern,
		)
	}

	depls := make([]Depl, 0, len(deplDefFiles))
	for _, deplDefFilePath := range deplDefFiles {
		if !strings.HasSuffix(deplDefFilePath, DeplDefFileExt) {
			continue
		}

		deplName := strings.TrimSuffix(deplDefFilePath, DeplDefFileExt)
		depl, err := loadDepl(fsys, deplName)
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't load package deployment config from %s", deplDefFilePath,
			)
		}
		depls = append(depls, depl)
	}
	return depls, nil
}

// ResourceAttachmentSource returns the source path for resources under the Depl instance.
// The resulting slice is useful for constructing [pallets.AttachedResource] instances.
func (d *Depl) ResourceAttachmentSource() []string {
	return []string{
		fmt.Sprintf("deployment %s", d.Name),
	}
}

// DeplDef

// loadDeplDef loads a DeplDef from the specified file path in the provided base filesystem.
func loadDeplDef(fsys pallets.PathedFS, filePath string) (DeplDef, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return DeplDef{}, errors.Wrapf(
			err, "couldn't read deployment config file %s/%s", fsys.Path(), filePath,
		)
	}
	config := DeplDef{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return DeplDef{}, errors.Wrap(err, "couldn't parse deployment config")
	}
	return config, nil
}
