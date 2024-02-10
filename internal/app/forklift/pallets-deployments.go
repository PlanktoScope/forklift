package forklift

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/PlanktoScope/forklift/pkg/core"
	"github.com/PlanktoScope/forklift/pkg/structures"
)

// ResolvedDepl

// ResolveDepl loads the package from the [FSPkgLoader] instance based on the requirements in the
// provided deployment and the package requirement loader.
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

// EnabledFeatures returns a map of the package features enabled by the deployment's configuration,
// with feature names as the keys of the map.
func (d *ResolvedDepl) EnabledFeatures() (enabled map[string]core.PkgFeatureSpec, err error) {
	all := d.Pkg.Def.Features
	enabled = make(map[string]core.PkgFeatureSpec)
	for _, name := range d.Def.Features {
		featureSpec, ok := all[name]
		if !ok {
			return nil, errors.Errorf("unrecognized feature %s", name)
		}
		enabled[name] = featureSpec
	}
	return enabled, nil
}

// DisabledFeatures returns a map of the package features not enabled by the deployment's
// configuration, with feature names as the keys of the map.
func (d *ResolvedDepl) DisabledFeatures() map[string]core.PkgFeatureSpec {
	all := d.Pkg.Def.Features
	enabled := make(map[string]struct{})
	for _, name := range d.Def.Features {
		enabled[name] = struct{}{}
	}
	disabled := make(map[string]core.PkgFeatureSpec)
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

// GetComposeFilenames returns a list of the names of the Compose files which must be merged into
// the Compose app, with feature-flagged Compose files ordered based on the alphabetical order of
// enabled feature flags.
func (d *ResolvedDepl) GetComposeFilenames() ([]string, error) {
	composeFiles := append([]string{}, d.Pkg.Def.Deployment.ComposeFiles...)

	// Add compose files from features
	enabledFeatures, err := d.EnabledFeatures()
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't determine enabled features of deployment %s", d.Name)
	}
	orderedNames := make([]string, 0, len(enabledFeatures))
	for name := range enabledFeatures {
		orderedNames = append(orderedNames, name)
	}
	sort.Strings(orderedNames)
	for _, name := range orderedNames {
		composeFiles = append(composeFiles, enabledFeatures[name].ComposeFiles...)
	}
	return composeFiles, nil
}

// DefinesApp determines whether the deployment defines a Docker Compose app to be deployed.
func (d *ResolvedDepl) DefinesApp() (bool, error) {
	composeFiles, err := d.GetComposeFilenames()
	if err != nil {
		return false, errors.Wrap(err, "couldn't determine Compose files for deployment")
	}
	if len(composeFiles) == 0 {
		return false, nil
	}
	for _, file := range composeFiles {
		if len(file) > 0 {
			return true, nil
		}
	}
	return false, nil
}

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
		Listeners: core.CheckResConflicts(
			d.providedListeners(enabledFeatures), candidate.providedListeners(candidateEnabledFeatures),
		),
		Networks: core.CheckResConflicts(
			d.providedNetworks(enabledFeatures), candidate.providedNetworks(candidateEnabledFeatures),
		),
		Services: core.CheckResConflicts(
			d.providedServices(enabledFeatures), candidate.providedServices(candidateEnabledFeatures),
		),
		Filesets: core.CheckResConflicts(
			d.providedFilesets(enabledFeatures), candidate.providedFilesets(candidateEnabledFeatures),
		),
	}, nil
}

// providedListeners returns a slice of all host port listeners provided by the package deployment,
// depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) providedListeners(
	enabledFeatures map[string]core.PkgFeatureSpec,
) (provided []core.AttachedRes[core.ListenerRes]) {
	return d.Pkg.ProvidedListeners(d.ResAttachmentSource(), sortKeys(enabledFeatures))
}

// requiredNetworks returns a slice of all Docker networks required by the package deployment,
// depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) requiredNetworks(
	enabledFeatures map[string]core.PkgFeatureSpec,
) (required []core.AttachedRes[core.NetworkRes]) {
	return d.Pkg.RequiredNetworks(d.ResAttachmentSource(), sortKeys(enabledFeatures))
}

// providedNetworks returns a slice of all Docker networks provided by the package deployment,
// depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) providedNetworks(
	enabledFeatures map[string]core.PkgFeatureSpec,
) (provided []core.AttachedRes[core.NetworkRes]) {
	return d.Pkg.ProvidedNetworks(d.ResAttachmentSource(), sortKeys(enabledFeatures))
}

// requiredServices returns a slice of all network services required by the package deployment,
// depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) requiredServices(
	enabledFeatures map[string]core.PkgFeatureSpec,
) (required []core.AttachedRes[core.ServiceRes]) {
	return d.Pkg.RequiredServices(d.ResAttachmentSource(), sortKeys(enabledFeatures))
}

// providedServices returns a slice of all network services provided by the package deployment,
// depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) providedServices(
	enabledFeatures map[string]core.PkgFeatureSpec,
) (provided []core.AttachedRes[core.ServiceRes]) {
	return d.Pkg.ProvidedServices(d.ResAttachmentSource(), sortKeys(enabledFeatures))
}

// requiredFilesets returns a slice of all filesets required by the package deployment,
// depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) requiredFilesets(
	enabledFeatures map[string]core.PkgFeatureSpec,
) (required []core.AttachedRes[core.FilesetRes]) {
	return d.Pkg.RequiredFilesets(d.ResAttachmentSource(), sortKeys(enabledFeatures))
}

// providedFilesets returns a slice of all filesets provided by the package deployment,
// depending on the enabled features, with feature names as the keys of the map.
func (d *ResolvedDepl) providedFilesets(
	enabledFeatures map[string]core.PkgFeatureSpec,
) (provided []core.AttachedRes[core.FilesetRes]) {
	return d.Pkg.ProvidedFilesets(d.ResAttachmentSource(), sortKeys(enabledFeatures))
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
	candidateEnabledFeatures := make([]map[string]core.PkgFeatureSpec, 0, len(candidates))
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
		allProvidedNetworks []core.AttachedRes[core.NetworkRes]
		allProvidedServices []core.AttachedRes[core.ServiceRes]
		allProvidedFilesets []core.AttachedRes[core.FilesetRes]
	)
	for i, candidate := range candidates {
		enabled := candidateEnabledFeatures[i]
		allProvidedNetworks = append(allProvidedNetworks, candidate.providedNetworks(enabled)...)
		allProvidedServices = append(allProvidedServices, candidate.providedServices(enabled)...)
		allProvidedFilesets = append(allProvidedFilesets, candidate.providedFilesets(enabled)...)
	}

	satisfied.Depl = d
	missing.Depl = d
	satisfied.Networks, missing.Networks = core.CheckResDeps(
		d.requiredNetworks(enabledFeatures), allProvidedNetworks,
	)
	satisfied.Services, missing.Services = core.CheckResDeps(
		core.SplitServicesByPath(d.requiredServices(enabledFeatures)), allProvidedServices,
	)
	satisfied.Filesets, missing.Filesets = core.CheckResDeps(
		core.SplitFilesetsByPath(d.requiredFilesets(enabledFeatures)), allProvidedFilesets,
	)
	return satisfied, missing, nil
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
			provider := strings.TrimPrefix(network.Provided.Source[0], "deployment ")
			if provider == satisfied.Depl.Name { // i.e. the deployment requires a resource it provides
				continue
			}
			deps.AddEdge(satisfied.Depl.Name, provider)
		}
		for _, service := range satisfied.Services {
			provider := strings.TrimPrefix(service.Provided.Source[0], "deployment ")
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
			provider := strings.TrimPrefix(fileset.Provided.Source[0], "deployment ")
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

// Depl

// FilterDeplsForEnabled filters a slice of Depls to only include those which are not disabled.
func FilterDeplsForEnabled(depls []Depl) []Depl {
	filtered := make([]Depl, 0, len(depls))
	for _, depl := range depls {
		if depl.Def.Disabled {
			continue
		}
		filtered = append(filtered, depl)
	}
	return filtered
}

// loadDepl loads the Depl from a file path in the provided base filesystem, assuming the file path
// is the specified name of the deployment followed by the deployment config file extension.
func loadDepl(fsys core.PathedFS, name string) (depl Depl, err error) {
	depl.Name = name
	if depl.Def, err = loadDeplDef(fsys, name+DeplDefFileExt); err != nil {
		return Depl{}, errors.Wrapf(err, "couldn't load version depl config")
	}
	return depl, nil
}

// loadDepls loads all package deployment configurations from the provided base filesystem matching
// the specified search pattern.
// The search pattern should not include the file extension for deployment specification files - the
// file extension will be appended to the search pattern by LoadDepls.
func loadDepls(fsys core.PathedFS, searchPattern string) ([]Depl, error) {
	searchPattern += DeplDefFileExt
	deplDefFiles, err := doublestar.Glob(fsys, searchPattern)
	if err != nil {
		return nil, errors.Wrapf(
			err, "couldn't search for package deployment configs matching %s/%s",
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

// ResAttachmentSource returns the source path for resources under the Depl instance.
// The resulting slice is useful for constructing [core.AttachedRes] instances.
func (d *Depl) ResAttachmentSource() []string {
	return []string{
		fmt.Sprintf("deployment %s", d.Name),
	}
}

// DeplDef

// loadDeplDef loads a DeplDef from the specified file path in the provided base filesystem.
func loadDeplDef(fsys core.PathedFS, filePath string) (DeplDef, error) {
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
