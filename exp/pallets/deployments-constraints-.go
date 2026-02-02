package pallets

import (
	"slices"

	fpkg "github.com/forklift-run/forklift/exp/packaging"
	res "github.com/forklift-run/forklift/exp/resources"
)

// ResolvedDepl: Constraints

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
