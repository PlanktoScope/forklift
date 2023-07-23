package forklift

import (
	"fmt"
	"io/fs"
	"sort"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/pkg/pallets"
)

// ResolvedDepl

func (d *ResolvedDepl) Check() (errs []error) {
	if d.PkgReq.Path() != d.Config.Package {
		errs = append(errs, errors.Errorf(
			"required package %s does not match required package %s in deployment configuration",
			d.PkgReq.Path(), d.Config.Package,
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
	all := d.Pkg.Config.Features
	enabled = make(map[string]pallets.PkgFeatureSpec)
	for _, name := range d.Config.Features {
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
	all := d.Pkg.Config.Features
	enabled := make(map[string]struct{})
	for _, name := range d.Config.Features {
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

// ResourceAttachmentSource returns the source path for resources under the Depl instance.
// The resulting slice is useful for constructing [pallets.AttachedResource] instances.
func (d *Depl) ResourceAttachmentSource() []string {
	return []string{
		fmt.Sprintf("deployment %s", d.Name),
	}
}

// DeplConfig

// loadDeplConfig loads a DeplConfig from the specified file path in the provided base filesystem.
func loadDeplConfig(fsys pallets.PathedFS, filePath string) (DeplConfig, error) {
	bytes, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return DeplConfig{}, errors.Wrapf(
			err, "couldn't read deployment config file %s/%s", fsys.Path(), filePath,
		)
	}
	config := DeplConfig{}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return DeplConfig{}, errors.Wrap(err, "couldn't parse deployment config")
	}
	return config, nil
}
