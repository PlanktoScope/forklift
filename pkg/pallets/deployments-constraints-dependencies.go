package pallets

import (
	"cmp"
	"strings"

	"github.com/pkg/errors"

	fpkg "github.com/forklift-run/forklift/pkg/packaging"
	res "github.com/forklift-run/forklift/pkg/resources"
	"github.com/forklift-run/forklift/pkg/structures"
)

type SatisfiedDeplDeps struct {
	Depl *ResolvedDepl

	Networks []res.SatisfiedDep[fpkg.NetworkRes, []string]
	Services []res.SatisfiedDep[fpkg.ServiceRes, []string]
	Filesets []res.SatisfiedDep[fpkg.FilesetRes, []string]
}

type MissingDeplDeps struct {
	Depl *ResolvedDepl

	Networks []res.MissingDep[fpkg.NetworkRes, []string]
	Services []res.MissingDep[fpkg.ServiceRes, []string]
	Filesets []res.MissingDep[fpkg.FilesetRes, []string]
}

// SatisfiedDeplDeps

func (d SatisfiedDeplDeps) HasSatisfiedNetworkDep() bool {
	return len(d.Networks) > 0
}

func (d SatisfiedDeplDeps) HasSatisfiedServiceDep() bool {
	return len(d.Services) > 0
}

func (d SatisfiedDeplDeps) HasSatisfiedFilesetDep() bool {
	return len(d.Filesets) > 0
}

func (d SatisfiedDeplDeps) HasSatisfiedDep() bool {
	return cmp.Or(
		d.HasSatisfiedNetworkDep(),
		d.HasSatisfiedServiceDep(),
		d.HasSatisfiedFilesetDep(),
	)
}

// MissingDeplDeps

func (d MissingDeplDeps) HasMissingNetworkDep() bool {
	return len(d.Networks) > 0
}

func (d MissingDeplDeps) HasMissingServiceDep() bool {
	return len(d.Services) > 0
}

func (d MissingDeplDeps) HasMissingFilesetDep() bool {
	return len(d.Filesets) > 0
}

func (d MissingDeplDeps) HasMissingDep() bool {
	return cmp.Or(
		d.HasMissingNetworkDep(),
		d.HasMissingServiceDep(),
		d.HasMissingFilesetDep(),
	)
}

// ResolvedDepl: Constraints: Resource Dependencies

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
