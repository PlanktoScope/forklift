package forklift

import (
	"slices"
	"strings"

	"github.com/pkg/errors"

	fpkg "github.com/forklift-run/forklift/pkg/packaging"
	res "github.com/forklift-run/forklift/pkg/resources"
	"github.com/forklift-run/forklift/pkg/structures"
)

// ResolvedDepl: Constraints

// CheckConflicts produces a report of all resource conflicts between the ResolvedDepl instance and
// a candidate ResolvedDepl.
func (d *ResolvedDepl) CheckConflicts(candidate *ResolvedDepl) (DeplConflict, error) {
	enabledFeatures, err := d.EnabledFeatures()
	if err != nil {
		return DeplConflict{}, errors.Wrapf(
			err, "couldn't determine enabled features of deployment %s", d.Name,
		)
	}
	candidateEnabledFeatures, err := candidate.EnabledFeatures()
	if err != nil {
		return DeplConflict{}, errors.Wrapf(
			err, "couldn't determine enabled features of deployment %s", candidate.Name,
		)
	}
	return DeplConflict{
		First:  d,
		Second: candidate,
		Name:   d.Name == candidate.Name,
		Listeners: res.CheckConflicts(
			d.providedListeners(enabledFeatures), candidate.providedListeners(candidateEnabledFeatures),
		),
		Networks: res.CheckConflicts(
			d.providedNetworks(enabledFeatures), candidate.providedNetworks(candidateEnabledFeatures),
		),
		Services: res.CheckConflicts(
			d.providedServices(enabledFeatures), candidate.providedServices(candidateEnabledFeatures),
		),
		Filesets: res.CheckConflicts(
			d.providedFilesets(enabledFeatures), candidate.providedFilesets(candidateEnabledFeatures),
		),
		FileExports: res.CheckConflicts(
			d.providedFileExports(enabledFeatures),
			candidate.providedFileExports(candidateEnabledFeatures),
		),
	}, nil
}

// providedListeners returns a slice of all host port listeners provided by the package deployment,
// depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) providedListeners(
	enabledFeatures map[string]fpkg.PkgFeatureSpec,
) (provided []res.Attached[fpkg.ListenerRes, []string]) {
	return d.Pkg.ProvidedListeners(d.ResAttachmentSource(), sortKeys(enabledFeatures))
}

// sortKeys returns an alphabetically sorted slice of the keys of a map with string keys.
func sortKeys[Value any](m map[string]Value) (sorted []string) {
	sorted = make([]string, 0, len(m))
	for key := range m {
		sorted = append(sorted, key)
	}
	slices.Sort(sorted)
	return sorted
}

// requiredNetworks returns a slice of all Docker networks required by the package deployment,
// depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) requiredNetworks(
	enabledFeatures map[string]fpkg.PkgFeatureSpec,
) (required []res.Attached[fpkg.NetworkRes, []string]) {
	return d.Pkg.RequiredNetworks(d.ResAttachmentSource(), sortKeys(enabledFeatures))
}

// providedNetworks returns a slice of all Docker networks provided by the package deployment,
// depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) providedNetworks(
	enabledFeatures map[string]fpkg.PkgFeatureSpec,
) (provided []res.Attached[fpkg.NetworkRes, []string]) {
	return d.Pkg.ProvidedNetworks(d.ResAttachmentSource(), sortKeys(enabledFeatures))
}

// requiredServices returns a slice of all network services required by the package deployment,
// depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) requiredServices(
	enabledFeatures map[string]fpkg.PkgFeatureSpec,
) (required []res.Attached[fpkg.ServiceRes, []string]) {
	return d.Pkg.RequiredServices(d.ResAttachmentSource(), sortKeys(enabledFeatures))
}

// providedServices returns a slice of all network services provided by the package deployment,
// depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) providedServices(
	enabledFeatures map[string]fpkg.PkgFeatureSpec,
) (provided []res.Attached[fpkg.ServiceRes, []string]) {
	return d.Pkg.ProvidedServices(d.ResAttachmentSource(), sortKeys(enabledFeatures))
}

// requiredFilesets returns a slice of all filesets required by the package deployment,
// depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) requiredFilesets(
	enabledFeatures map[string]fpkg.PkgFeatureSpec,
) (required []res.Attached[fpkg.FilesetRes, []string]) {
	return d.Pkg.RequiredFilesets(d.ResAttachmentSource(), sortKeys(enabledFeatures))
}

// providedFilesets returns a slice of all filesets provided by the package deployment,
// depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) providedFilesets(
	enabledFeatures map[string]fpkg.PkgFeatureSpec,
) (provided []res.Attached[fpkg.FilesetRes, []string]) {
	return d.Pkg.ProvidedFilesets(d.ResAttachmentSource(), sortKeys(enabledFeatures))
}

// providedFileExports returns a slice of all file exports provided by the package deployment,
// depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) providedFileExports(
	enabledFeatures map[string]fpkg.PkgFeatureSpec,
) (provided []res.Attached[fpkg.FileExportRes, []string]) {
	return d.Pkg.ProvidedFileExports(d.ResAttachmentSource(), sortKeys(enabledFeatures))
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

// CheckDeps produces a report of all resource requirements from the ResolvedDepl
// instance and which were and were not met by any candidate ResolvedDepl.
func (d *ResolvedDepl) CheckDeps(
	candidates []*ResolvedDepl,
) (satisfied SatisfiedDeplDeps, missing MissingDeplDeps, err error) {
	enabledFeatures, err := d.EnabledFeatures()
	if err != nil {
		return SatisfiedDeplDeps{}, MissingDeplDeps{}, errors.Wrapf(
			err, "couldn't determine enabled features of deployment %s", d.Name,
		)
	}
	candidateEnabledFeatures := make([]map[string]fpkg.PkgFeatureSpec, 0, len(candidates))
	for _, candidate := range candidates {
		f, err := candidate.EnabledFeatures()
		if err != nil {
			return SatisfiedDeplDeps{}, MissingDeplDeps{}, errors.Wrapf(
				err, "couldn't determine enabled features of deployment %s", candidate.Name,
			)
		}
		candidateEnabledFeatures = append(candidateEnabledFeatures, f)
	}

	var (
		allProvidedNetworks []res.Attached[fpkg.NetworkRes, []string]
		allProvidedServices []res.Attached[fpkg.ServiceRes, []string]
		allProvidedFilesets []res.Attached[fpkg.FilesetRes, []string]
	)
	for i, candidate := range candidates {
		enabled := candidateEnabledFeatures[i]
		allProvidedNetworks = append(allProvidedNetworks, candidate.providedNetworks(enabled)...)
		allProvidedServices = append(allProvidedServices, candidate.providedServices(enabled)...)
		allProvidedFilesets = append(allProvidedFilesets, candidate.providedFilesets(enabled)...)
	}

	satisfied.Depl = d
	missing.Depl = d
	satisfied.Networks, missing.Networks = res.CheckDeps(
		d.requiredNetworks(enabledFeatures), allProvidedNetworks,
	)
	satisfied.Services, missing.Services = res.CheckDeps(
		fpkg.SplitServicesByPath(d.requiredServices(enabledFeatures)), allProvidedServices,
	)
	satisfied.Filesets, missing.Filesets = res.CheckDeps(
		fpkg.SplitFilesetsByPath(d.requiredFilesets(enabledFeatures)), allProvidedFilesets,
	)
	return satisfied, missing, nil
}

// Checking

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

// CheckDeplDeps produces reports of all satisfied and unsatisfied resource dependencies
// among all provided ResolvedDepl instances.
func CheckDeplDeps(
	depls []*ResolvedDepl,
) (satisfiedDeps []SatisfiedDeplDeps, missingDeps []MissingDeplDeps, err error) {
	for _, depl := range depls {
		satisfied, missing, err := depl.CheckDeps(depls)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "couldn't check dependencies of deployment %s", depl.Name)
		}
		if missing.HasMissingDep() {
			missingDeps = append(missingDeps, missing)
			continue
		}
		satisfiedDeps = append(satisfiedDeps, satisfied)
	}
	return satisfiedDeps, missingDeps, nil
}

// ResolveDeps returns a digraph where each node is the name of a deployment and each edge goes from
// a deployment which requires some resource to a deployment which provides that resource. Thus, the
// returned graph is a graph of direct dependencies among deployments, excluding deployments with
// no dependency relationships. If the skipNonblocking arg is set, then nonblocking resource
// requirements are ignored as if they didn't exist.
func ResolveDeps(
	satisfiedDeps []SatisfiedDeplDeps, skipNonblocking bool,
) structures.Digraph[string] {
	deps := make(structures.Digraph[string])
	for _, satisfied := range satisfiedDeps {
		for _, network := range satisfied.Networks {
			provider := strings.TrimPrefix(network.Provided.Origin[0], "deployment ")
			if provider == satisfied.Depl.Name { // i.e. the deployment requires a resource it provides
				continue
			}
			deps.AddEdge(satisfied.Depl.Name, provider)
		}
		for _, service := range satisfied.Services {
			provider := strings.TrimPrefix(service.Provided.Origin[0], "deployment ")
			deps.AddNode(provider)
			if provider == satisfied.Depl.Name { // i.e. the deployment requires a resource it provides
				continue
			}
			if service.Required.Res.Nonblocking && skipNonblocking {
				continue
			}
			deps.AddEdge(satisfied.Depl.Name, provider)
		}
		for _, fileset := range satisfied.Filesets {
			provider := strings.TrimPrefix(fileset.Provided.Origin[0], "deployment ")
			deps.AddNode(provider)
			if provider == satisfied.Depl.Name { // i.e. the deployment requires a resource it provides
				continue
			}
			if fileset.Required.Res.Nonblocking && skipNonblocking {
				continue
			}
			deps.AddEdge(satisfied.Depl.Name, provider)
		}
	}
	return deps
}
